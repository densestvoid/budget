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

locals {
  project_name = "budget-develop"

  # PR pool: 254×254 /24 slots as 10.S.T.0/24 (S=1..254, T=0..253).
  # Production uses 10.0.0.0/16 so S starts at 1.
  pr_number    = tonumber(regex("^pr-(\\d+)$", var.deployment_id)[0])
  vpc_slots    = 254 * 254
  vpc_slot     = local.pr_number % local.vpc_slots
  vpc_second   = floor(local.vpc_slot / 254) + 1
  vpc_third    = local.vpc_slot % 254
  vpc_ip_range = format("10.%d.%d.0/24", local.vpc_second, local.vpc_third)
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

# Use the budget-app module
module "budget_app" {
  source = "../modules/budget-app"

  do_token      = var.do_token
  region        = var.region
  deployment_id = var.deployment_id
  project_name  = local.project_name
  # github_repo is auto-detected from GITHUB_REPOSITORY env var in the module
  docker_image_tag = var.docker_image_tag

  domain = var.domain_name != "" ? {
    hostname = "${var.deployment_id}.${var.domain_name}"
    zone     = var.domain_name
  } : null

  # VPC configuration - use the VPC created above
  vpc_id = digitalocean_vpc.budget_vpc.id

  # Database configuration from newly created resources
  database_cluster_id    = digitalocean_database_cluster.budget_db.id
  database_name          = digitalocean_database_db.budget_database.name
  database_user_name     = digitalocean_database_user.budget_user.name
  database_user_password = digitalocean_database_user.budget_user.password
  database_private_host   = digitalocean_database_cluster.budget_db.private_host
  database_port           = digitalocean_database_cluster.budget_db.port
  database_admin_user     = digitalocean_database_cluster.budget_db.user
  database_admin_password = digitalocean_database_cluster.budget_db.password
}
