# State-only PR teardown: no resource definitions.
# CI runs `terraform apply` (not `destroy`) — destroy only removes configured resources;
# with none here, state entries are orphans torn down by apply.
terraform {
  required_version = ">= 1.0"

  backend "s3" {
    bucket = "densestvoid-terraform"
    key    = "pr/{deployment_id}.tfstate" # overridden by -backend-config at init
    region = "us-east-1"
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

provider "digitalocean" {
  token = var.do_token
}
