# Networking module for both PR and Production deployments
# This module creates VPC and networking resources

terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

# Variables for networking module
variable "vpc_name" {
  description = "Name of the VPC"
  type        = string
}

variable "region" {
  description = "DigitalOcean region"
  type        = string
}

# Create VPC for private networking
resource "digitalocean_vpc" "budget_vpc" {
  name     = var.vpc_name
  region   = var.region
  ip_range = "172.16.0.0/16"  # Private IP range (avoiding conflicts)
  
  # Note: VPC doesn't support tags, but name includes deployment_id for identification
}

# Outputs for networking module
output "vpc_id" {
  description = "VPC ID"
  value       = digitalocean_vpc.budget_vpc.id
}

output "vpc_name" {
  description = "VPC name"
  value       = digitalocean_vpc.budget_vpc.name
}