terraform {
  required_version = ">= 1.0"

  backend "s3" {
    bucket = "densestvoid-terraform"
    key    = "production/production.tfstate"
    region = "us-east-1" # This will be overridden by -backend-config
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
  project_name  = "budget-prod"
  deployment_id = "production"
  # Hardcoded production values
  database_cluster_name = "production"
  database_name         = "production"
  database_user_name    = "production"
}

# Reference existing DigitalOcean project
data "digitalocean_project" "budget" {
  name = local.project_name
}

# Create VPC for private networking (shared between database and app)
resource "digitalocean_vpc" "budget_vpc" {
  name     = local.deployment_id
  region   = var.region
  ip_range = "10.0.0.0/16"
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
  # Allow Terraform to manage database scaling (size, node_count)
  lifecycle {
    prevent_destroy = true
  }
}

# Create database within the cluster
resource "digitalocean_database_db" "budget_database" {
  cluster_id = digitalocean_database_cluster.budget_db.id
  name       = local.database_name

  # Prevent destruction - this is a long-living database
  lifecycle {
    prevent_destroy = true
  }
}

# Create database user
resource "digitalocean_database_user" "budget_user" {
  cluster_id = digitalocean_database_cluster.budget_db.id
  name       = local.database_user_name

  # Prevent destruction
  lifecycle {
    prevent_destroy = true
    ignore_changes  = [settings]
  }
}

# Use the budget-app module
module "budget_app" {
  source = "../modules/budget-app"

  do_token      = var.do_token
  region        = var.region
  deployment_id = local.deployment_id
  project_name  = local.project_name
  # github_repo is auto-detected from GITHUB_REPOSITORY env var in the module
  docker_image_tag = var.docker_image_tag

  domain = var.domain_name != "" ? {
    hostname = var.domain_name
    zone     = var.domain_name
  } : null

  # Database configuration
  database_cluster_id    = digitalocean_database_cluster.budget_db.id
  database_name          = digitalocean_database_db.budget_database.name
  database_user_name     = digitalocean_database_user.budget_user.name
  database_user_password = digitalocean_database_user.budget_user.password
  database_private_host  = digitalocean_database_cluster.budget_db.private_host
  database_port          = digitalocean_database_cluster.budget_db.port
  database_admin_user     = digitalocean_database_cluster.budget_db.user
  database_admin_password = digitalocean_database_cluster.budget_db.password

  # Use the same VPC as the database for private networking
  vpc_id = digitalocean_vpc.budget_vpc.id
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

# When domain_name is set, App Platform manages DNS records in the pre-existing DO zone.

