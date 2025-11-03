terraform {
  required_version = ">= 1.0"
  
  backend "s3" {
    bucket = "densestvoid-terraform"
    key    = "production/production.tfstate"
    region = "us-east-1"  # This will be overridden by -backend-config
  }
  
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

# Configure the DigitalOcean Provider
provider "digitalocean" {
  token = var.do_token
}

# Local values
locals {
  project_name = "budget-prod"
  deployment_id = "production"
  # Hardcoded production values
  database_cluster_name = "production"
  database_name = "production"
  database_user_name = "production"
  domain_name = "budget.densestvoid.dev"
}

# Reference existing DigitalOcean project
data "digitalocean_project" "budget" {
  name = local.project_name
}

# Create VPC for private networking (shared between database and app)
resource "digitalocean_vpc" "budget_vpc" {
  name     = local.deployment_id
  region   = var.region
  ip_range = "172.16.0.0/16"
}

# Create database cluster (creates on first deployment, manages existing on subsequent deployments)
resource "digitalocean_database_cluster" "budget_db" {
  name                 = local.database_cluster_name
  engine               = "pg"
  version              = "16"
  size                 = "db-s-1vcpu-1gb"
  region               = var.region
  node_count           = 1
  private_network_uuid = digitalocean_vpc.budget_vpc.id

  tags = ["deployment:production"]
  
  # Prevent destruction - this is a long-living database
  lifecycle {
    prevent_destroy = true
    ignore_changes  = [size, node_count]
  }
}

# Create database within the cluster
resource "digitalocean_database_db" "budget_database" {
  cluster_id = digitalocean_database_cluster.budget_db.id
  name       = local.database_name
}

# Create database user
resource "digitalocean_database_user" "budget_user" {
  cluster_id = digitalocean_database_cluster.budget_db.id
  name       = local.database_user_name
  
  # Prevent destruction
  lifecycle {
    prevent_destroy = true
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

# Reference existing domain (pre-allocated, DNS records are preconfigured)
data "digitalocean_domain" "existing_domain" {
  name = local.domain_name
}

# Use the budget-app module
module "budget_app" {
  source = "../modules/budget-app"
  
  do_token         = var.do_token
  region           = var.region
  deployment_id    = local.deployment_id
  project_name     = local.project_name
  # github_repo is auto-detected from GITHUB_REPOSITORY env var in the module
  docker_image_tag = var.docker_image_tag
  
  # Database configuration
  database_cluster_id   = digitalocean_database_cluster.budget_db.id
  database_name         = digitalocean_database_db.budget_database.name
  database_user_name    = digitalocean_database_user.budget_user.name
  database_user_password = digitalocean_database_user.budget_user.password
  database_private_host = digitalocean_database_cluster.budget_db.private_host
  database_port         = digitalocean_database_cluster.budget_db.port
  
  # Use the same VPC as the database for private networking
  vpc_id = digitalocean_vpc.budget_vpc.id
  
  # Ensure schema setup completes before module is instantiated (and migrations run)
  depends_on = [null_resource.database_schema_setup]
}

# Assign database cluster to the budget-prod project
# (Apps are assigned via the module's project_resources resource)
resource "digitalocean_project_resources" "production_database" {
  project = data.digitalocean_project.budget.id
  resources = [
    digitalocean_database_cluster.budget_db.urn
  ]
  
  depends_on = [
    digitalocean_database_cluster.budget_db
  ]
}

# Note: DNS records are pre-allocated and managed outside of Terraform
# The domain is referenced above for informational purposes only

