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

# Calculate precise termination time and cron schedule
locals {
  # Calculate termination time (current time + specified minutes)
  termination_timestamp = timeadd(timestamp(), "${var.auto_terminate_minutes}m")
  
  # Round up to the next minute for precise cron scheduling
  base_minute = tonumber(formatdate("m", local.termination_timestamp))
  base_hour = tonumber(formatdate("h", local.termination_timestamp))
  base_day = tonumber(formatdate("D", local.termination_timestamp))
  base_month = tonumber(formatdate("M", local.termination_timestamp))
  base_seconds = tonumber(formatdate("s", local.termination_timestamp))
  
  # Round up to next minute if there are seconds
  adjusted_minute = local.base_seconds > 0 ? local.base_minute + 1 : local.base_minute
  
  # Handle minute overflow (59 + 1 = 0, hour++)
  final_minute = local.adjusted_minute >= 60 ? 0 : local.adjusted_minute
  final_hour = local.adjusted_minute >= 60 ? local.base_hour + 1 : local.base_hour
  
  # Handle hour overflow (23 + 1 = 0, day++)  
  adjusted_hour = local.final_hour >= 24 ? 0 : local.final_hour
  adjusted_day = local.final_hour >= 24 ? local.base_day + 1 : local.base_day
  
  # Create precise cron expression: "minute hour day month *"
  precise_cron = "${local.final_minute} ${local.adjusted_hour} ${local.adjusted_day} ${local.base_month} *"
  
  # Human-readable termination time
  termination_display = formatdate("YYYY-MM-DD hh:mm:ss UTC", timeadd(local.termination_timestamp, local.base_seconds > 0 ? "1m" : "0m"))
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

    # Precise auto-termination function (deletes the entire app including itself)
    function {
      name = "terminate"
      
      source_dir = "${path.module}/termination-function"
      
      env {
        key   = "DO_TOKEN"
        value = var.do_token
        scope = "RUN_TIME"
        type  = "SECRET"
      }

      env {
        key   = "TARGET_APP_ID"
        value = digitalocean_app.budget_app.id
        scope = "RUN_TIME"
      }

      env {
        key   = "TERMINATION_TIME"
        value = local.termination_timestamp
        scope = "RUN_TIME"
      }

      # Precise cron schedule - runs exactly when termination should occur
      triggers {
        name = "precise_termination"
        type = "SCHEDULED"
        config {
          cron = local.precise_cron
        }
      }
    }
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