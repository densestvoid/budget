terraform {
  required_version = ">= 1.0"
  
  backend "s3" {
    bucket = "densestvoid-terraform"
    key    = "environments/prod/main.tfstate"
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

# Reference existing DigitalOcean project
data "digitalocean_project" "budget_prod" {
  name = "budget-prod"
}

# Local values for production naming
locals {
  # Production-specific naming
  vpc_name = "budget-prod-vpc"
  database_name = "budget-prod"
  database_user_name = "budget-prod-app"
  app_name = "budget-prod"
  migration_app_name = "budget-prod-migrations"
  
  # Production tags
  tags = ["environment:production", "project:budget-prod"]
}

# Create VPC for production
module "networking" {
  source = "../../networking.tf"
  
  vpc_name = local.vpc_name
  region   = var.region
}

# Create production database
module "database" {
  source = "../../database.tf"
  
  database_name = local.database_name
  database_user_name = local.database_user_name
  database_size = var.database_size
  region = var.region
  vpc_id = module.networking.vpc_id
  tags = local.tags
}

# Create production application
module "app" {
  source = "../../app.tf"
  
  app_name = local.app_name
  migration_app_name = local.migration_app_name
  github_repo = var.github_repo
  docker_image_tag = var.docker_image_tag
  region = var.region
  vpc_id = module.networking.vpc_id
  database_url = module.database.database_url
  environment = "production"
  instance_count = var.instance_count
  instance_size = var.instance_size
  health_check_initial_delay = var.health_check_initial_delay
  health_check_period = var.health_check_period
  health_check_timeout = var.health_check_timeout
  health_check_failure_threshold = var.health_check_failure_threshold
  database_dependencies = [
    module.database.database_health_check,
    module.database.database_schema_setup
  ]
}

# Assign resources to the production project
resource "digitalocean_project_resources" "budget_prod_resources" {
  project = data.digitalocean_project.budget_prod.id
  resources = [
    module.app.migration_app_urn,
    module.app.app_urn,
    module.database.database_cluster_urn
    # Note: VPC cannot be assigned to projects (not in supported resource types)
  ]
  
  # Ensure resources are created before assignment
  depends_on = [
    module.app.migration_app,
    module.app.app,
    module.database.database_cluster
  ]
}

# Create a domain record for production
resource "digitalocean_domain" "budget_domain" {
  name = "budget.densestvoid.dev"
}

resource "digitalocean_record" "budget_a_record" {
  domain = digitalocean_domain.budget_domain.id
  type   = "CNAME"
  name   = "@"
  value  = module.app.app_default_ingress
  ttl    = 3600
}