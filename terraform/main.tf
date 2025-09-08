terraform {
  required_version = ">= 1.0"
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.1"
    }
  }
}

# Configure the DigitalOcean Provider
provider "digitalocean" {
  token = var.do_token
}

# Generate a random ID for unique resource naming
resource "random_id" "deployment" {
  byte_length = 4
}

# Calculate expected termination time for display
locals {
  # Calculate when the app should be terminated (for display purposes)
  termination_timestamp = timeadd(timestamp(), "${var.auto_terminate_minutes}m")
  termination_display = formatdate("YYYY-MM-DD hh:mm:ss UTC", local.termination_timestamp)
}

# Create DigitalOcean App Platform application
resource "digitalocean_app" "budget_app" {
  spec {
    name   = "budget-app-${random_id.deployment.hex}"
    region = var.region

    # PostgreSQL database service
    service {
      name               = "postgres"
      instance_count     = 1
      instance_size_slug = "basic-xxs"  # $5/month

      image {
        registry_type = "DOCKER_HUB"
        repository    = "postgres"
        tag           = "16-alpine"
      }

      env {
        key   = "POSTGRES_DB"
        value = "budget"
        scope = "RUN_TIME"
      }

      env {
        key   = "POSTGRES_USER"
        value = "postgres"
        scope = "RUN_TIME"
      }

      env {
        key   = "POSTGRES_PASSWORD"
        value = "budget_password"
        scope = "RUN_TIME"
        type  = "SECRET"
      }

      env {
        key   = "POSTGRES_HOST_AUTH_METHOD"
        value = "trust"
        scope = "RUN_TIME"
      }

      # Internal service - no external access
      internal_ports = [5432]
    }

    # Main application service
    service {
      name               = "web"
      instance_count     = 1
      instance_size_slug = "basic-xxs"  # $5/month: 0.5 vCPU, 512MB RAM

      image {
        registry_type = "GHCR"
        registry      = "ghcr.io"
        repository    = "${var.github_repo}/budget-app"
        tag           = var.docker_image_tag
      }

      # Environment variables (with BUDGET_ prefix for viper)
      env {
        key   = "BUDGET_DATABASE_URL"
        value = "postgres://postgres:budget_password@postgres:5432/budget?sslmode=disable"
        scope = "RUN_TIME"
        type  = "SECRET"
      }

      env {
        key   = "BUDGET_PORT"
        value = "8080"
        scope = "RUN_TIME"
      }

      env {
        key   = "BUDGET_ENV"
        value = "production"
        scope = "RUN_TIME"
      }

      env {
        key   = "BUDGET_LOG_LEVEL"
        value = "debug"
        scope = "RUN_TIME"
      }

      # Health check with longer startup time
      health_check {
        http_path             = "/health"
        initial_delay_seconds = 60  # Give PostgreSQL time to start
        period_seconds        = 10
        timeout_seconds       = 5
        success_threshold     = 1
        failure_threshold     = 3
      }

      # HTTP port
      http_port = 8080
    }

    # Note: DigitalOcean App Platform functions may not support cron triggers in Terraform
    # Let's use a simpler approach with scheduled cleanup workflow instead
  }
}

# Create a domain record (optional)
resource "digitalocean_domain" "budget_domain" {
  count = var.domain_name != "" ? 1 : 0
  name  = var.domain_name
}

resource "digitalocean_record" "budget_a_record" {
  count  = var.domain_name != "" ? 1 : 0
  domain = digitalocean_domain.budget_domain[0].id
  type   = "CNAME"
  name   = "@"
  value  = digitalocean_app.budget_app.default_ingress
  ttl    = 3600
}