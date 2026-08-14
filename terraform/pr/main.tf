terraform {
  required_version = ">= 1.0"

  backend "s3" {
    bucket = "densestvoid-terraform"
    key    = "pr/{deployment_id}.tfstate"
    region = "us-east-1" # This will be overridden by -backend-config
  }

  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
    http = {
      source  = "hashicorp/http"
      version = "~> 3.0"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.0"
    }
  }
}

# Configure the DigitalOcean Provider
provider "digitalocean" {
  token = var.do_token
}

data "http" "vpcs" {
  url = "https://api.digitalocean.com/v2/vpcs?per_page=200"
  request_headers = {
    Authorization = "Bearer ${var.do_token}"
    Content-Type  = "application/json"
  }
}

locals {
  project_name = "budget-develop"

  vpcs_response = jsondecode(data.http.vpcs.response_body)
  region_vpcs   = [for v in local.vpcs_response.vpcs : v if v.region == var.region]

  existing_vpc_by_name = try(one([for v in local.region_vpcs : v if v.name == var.deployment_id]), null)

  # Occupied 10.S.0.0/24 slots in this region (production uses 10.0.0.0/16; PR pool uses S = 1..254).
  occupied_octets = distinct([
    for v in local.region_vpcs :
    tonumber(regex("^10\\.([0-9]+)\\.", v.ip_range)[0])
    if can(regex("^10\\.([0-9]+)\\.", v.ip_range))
  ])

  candidate_octets = [for s in range(1, 255) : s]

  # Hash-based scan offset spreads allocations across 254 slots; picks first unclaimed from offset.
  octet_count  = length(local.candidate_octets)
  start_index  = parseint(substr(md5(var.deployment_id), 0, 8), 16) % local.octet_count

  rotated_octets = concat(
    slice(local.candidate_octets, local.start_index, local.octet_count),
    slice(local.candidate_octets, 0, local.start_index)
  )

  selected_octet = one([
    for s in local.rotated_octets :
    s if !contains(local.occupied_octets, s)
  ])

  vpc_ip_range = local.existing_vpc_by_name != null ? local.existing_vpc_by_name.ip_range : "10.${local.selected_octet}.0.0/24"
}

# Reference existing DigitalOcean project
data "digitalocean_project" "budget" {
  name = local.project_name
}

# Create VPC for private networking
resource "digitalocean_vpc" "budget_vpc" {
  name     = var.deployment_id
  region   = var.region
  ip_range = local.vpc_ip_range

  lifecycle {
    ignore_changes = [ip_range]
  }
}

# Managed PostgreSQL database with private VPC networking
resource "digitalocean_database_cluster" "budget_db" {
  name                 = var.deployment_id
  engine               = "pg"
  version              = "16"
  size                 = "db-s-1vcpu-1gb"
  region               = var.region
  node_count           = 1
  private_network_uuid = digitalocean_vpc.budget_vpc.id

  tags = ["deployment-id:${var.deployment_id}"]
}

# Create database within the cluster
resource "digitalocean_database_db" "budget_database" {
  cluster_id = digitalocean_database_cluster.budget_db.id
  name       = var.deployment_id
}

# Create database user
resource "digitalocean_database_user" "budget_user" {
  cluster_id = digitalocean_database_cluster.budget_db.id
  name       = var.deployment_id

  lifecycle {
    ignore_changes = all
  }
}

# Create budget schema and grant privileges using null_resource
resource "null_resource" "database_schema_setup" {
  depends_on = [
    digitalocean_database_cluster.budget_db,
    digitalocean_database_db.budget_database,
    digitalocean_database_user.budget_user
  ]

  provisioner "local-exec" {
    command = <<-EOT
      echo "🔍 Checking if database schema exists (first deployment check)..."
      
      # Install postgresql-client if not available
      which psql || (echo "Installing postgresql-client..." && apt-get update && apt-get install -y postgresql-client)
      
      # Connect as admin user to check if schema exists
      ADMIN_URL="postgres://${digitalocean_database_cluster.budget_db.user}:${digitalocean_database_cluster.budget_db.password}@${digitalocean_database_cluster.budget_db.host}:${digitalocean_database_cluster.budget_db.port}/${digitalocean_database_db.budget_database.name}?sslmode=require"
      
      # Check if budget schema exists
      SCHEMA_EXISTS=$(psql "$ADMIN_URL" -tAc "SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = 'budget');" 2>/dev/null || echo "f")
      
      if [ "$SCHEMA_EXISTS" = "t" ]; then
        echo "✅ Budget schema already exists - skipping schema setup (not first deployment)"
        echo "ℹ️ Schema migrations will run via migration app"
      else
        echo "🗄️ First deployment detected - setting up database schema and permissions..."
        
        psql "$ADMIN_URL" <<SQL
          -- Create budget schema
          CREATE SCHEMA budget;
          
          -- Grant all privileges on budget schema ONLY to our user
          GRANT ALL PRIVILEGES ON SCHEMA budget TO "${digitalocean_database_user.budget_user.name}";
          
          -- Set default privileges for future tables in budget schema
          ALTER DEFAULT PRIVILEGES IN SCHEMA budget GRANT ALL ON TABLES TO "${digitalocean_database_user.budget_user.name}";
          ALTER DEFAULT PRIVILEGES IN SCHEMA budget GRANT ALL ON SEQUENCES TO "${digitalocean_database_user.budget_user.name}";
          ALTER DEFAULT PRIVILEGES IN SCHEMA budget GRANT ALL ON FUNCTIONS TO "${digitalocean_database_user.budget_user.name}";
          
          -- Make budget user the owner of budget schema
          ALTER SCHEMA budget OWNER TO "${digitalocean_database_user.budget_user.name}";
SQL
        
        if [ $? -eq 0 ]; then
          echo "✅ Database schema and permissions configured successfully (first deployment)"
        else
          echo "❌ Failed to configure database schema and permissions"
          exit 1
        fi
      fi
    EOT
  }
}

# Use the budget-app module
module "budget_app" {
  source = "../modules/budget-app"

  do_token         = var.do_token
  region           = var.region
  deployment_id    = var.deployment_id
  project_name     = local.project_name
  # github_repo is auto-detected from GITHUB_REPOSITORY env var in the module
  docker_image_tag = var.docker_image_tag

  # VPC configuration - use the VPC created above
  vpc_id = digitalocean_vpc.budget_vpc.id

  # Database configuration from newly created resources
  database_cluster_id    = digitalocean_database_cluster.budget_db.id
  database_name          = digitalocean_database_db.budget_database.name
  database_user_name     = digitalocean_database_user.budget_user.name
  database_user_password = digitalocean_database_user.budget_user.password
  database_private_host  = digitalocean_database_cluster.budget_db.private_host
  database_port          = digitalocean_database_cluster.budget_db.port

  # Ensure schema setup completes before module is instantiated (and migrations run)
  depends_on = [null_resource.database_schema_setup]
}
