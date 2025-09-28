# Networking module for both PR and Production deployments
# This module creates VPC and networking resources

# Create VPC for private networking
resource "digitalocean_vpc" "budget_vpc" {
  name     = var.vpc_name
  region   = var.region
  ip_range = "172.16.0.0/16"  # Private IP range (avoiding conflicts)
  
  # Note: VPC doesn't support tags, but name includes deployment_id for identification
}