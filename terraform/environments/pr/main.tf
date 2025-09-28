terraform {
  required_version = ">= 1.0"
  
  backend "s3" {
    bucket = "densestvoid-terraform"
    key    = "environments/pr/{deployment_id}.tfstate"
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
data "digitalocean_project" "budget_develop" {
  name = "budget-develop"
}

# Local values for PR deployment naming
locals {
  # Extract PR number from deployment_id (format: pr-{number}-{branch})
  pr_number = split("-", var.deployment_id)[1]
  
  # Simple, clean naming
  vpc_name = "pr-${local.pr_number}"
  database_name = "pr-${local.pr_number}"
  database_user_name = "pr-${local.pr_number}"
  app_name = "pr-${local.pr_number}"
  migration_app_name = "pr-${local.pr_number}-migrations"
  
  # PR tags
  tags = ["deployment-id:${var.deployment_id}", "environment:pr"]
}

# Create VPC for PR deployment
module "networking" {
  source = "../../modules/networking"
  
  vpc_name = local.vpc_name
  region   = var.region
}

# Create PR database
module "database" {
  source = "../../modules/database"
  
  database_name = local.database_name
  database_user_name = local.database_user_name
  database_size = var.database_size
  region = var.region
  vpc_id = module.networking.vpc_id
  tags = local.tags
}

# Create PR application
module "app" {
  source = "../../modules/app"
  
  app_name = local.app_name
  migration_app_name = local.migration_app_name
  github_repo = var.github_repo
  docker_image_tag = var.docker_image_tag
  region = var.region
  vpc_id = module.networking.vpc_id
  database_url = module.database.database_url
  environment = "production"
  instance_count = 1
  instance_size = "basic-xxs"
  health_check_initial_delay = 30
  health_check_period = 10
  health_check_timeout = 5
  health_check_failure_threshold = 5
  database_dependencies = [
    module.database.database_health_check,
    module.database.database_schema_setup
  ]
}

# Assign resources to the existing project
resource "digitalocean_project_resources" "budget_resources" {
  project = data.digitalocean_project.budget_develop.id
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