# App module for both PR and Production deployments
# This module creates DigitalOcean App Platform applications

terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

# Variables for app module
variable "app_name" {
  description = "Name of the main application"
  type        = string
}

variable "migration_app_name" {
  description = "Name of the migration application"
  type        = string
}

variable "github_repo" {
  description = "GitHub repository (user/repo format)"
  type        = string
}

variable "docker_image_tag" {
  description = "Docker image tag to deploy"
  type        = string
}

variable "region" {
  description = "DigitalOcean region"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID for private networking"
  type        = string
}

variable "database_url" {
  description = "Database connection URL"
  type        = string
  sensitive   = true
}

variable "environment" {
  description = "Environment name (production, staging, etc.)"
  type        = string
}

variable "instance_count" {
  description = "Number of app instances"
  type        = number
  default     = 1
}

variable "instance_size" {
  description = "App instance size"
  type        = string
  default     = "basic-xxs"
}

variable "health_check_initial_delay" {
  description = "Initial delay before health checks start (seconds)"
  type        = number
  default     = 30
}

variable "health_check_period" {
  description = "Health check period (seconds)"
  type        = number
  default     = 15
}

variable "health_check_timeout" {
  description = "Health check timeout (seconds)"
  type        = number
  default     = 5
}

variable "health_check_failure_threshold" {
  description = "Number of consecutive failures before marking unhealthy"
  type        = number
  default     = 3
}

variable "database_dependencies" {
  description = "Database dependencies for migration app"
  type        = list(any)
  default     = []
}

# Create migration app that runs migrations and exits
resource "digitalocean_app" "budget_migrations" {
  # Ensure database is ready, health-checked, and schema configured
  depends_on = [
    var.database_dependencies
  ]
  
  # Terraform will automatically detect if image tag changed
  # If image tag is same (cache hit), no redeployment needed

  spec {
    name   = var.migration_app_name
    region = var.region
    
    # Enable VPC networking for database access
    vpc {
      id = var.vpc_id
    }

    # Migration job - runs once and exits
    job {
      name = "migrate"
      kind = "PRE_DEPLOY"  # Runs before main service deployment
      
      image {
        registry_type = "GHCR"
        registry      = "ghcr.io"
        repository    = var.github_repo
        tag           = var.docker_image_tag
      }

      # Environment variables for migration
      env {
        key   = "BUDGET_DATABASE_URL"
        value = var.database_url
        scope = "RUN_TIME"
        type  = "SECRET"
      }

      env {
        key   = "BUDGET_ENV"
        value = var.environment
        scope = "RUN_TIME"
      }

      env {
        key   = "BUDGET_LOG_LEVEL"
        value = "info"
        scope = "RUN_TIME"
      }

      # Run migrations with proper error handling
      run_command = "sh -c 'echo \"🔍 Migration job starting...\"; echo \"Environment:\"; env | grep BUDGET; echo \"🔍 Testing DB connection...\"; ./budget migrate status; echo \"🔄 Running migrations...\"; if ./budget migrate; then echo \"✅ Migration completed successfully\"; exit 0; else echo \"❌ Migration failed with exit code $?\"; exit 1; fi'"
    }
  }
}

# Create main application after migrations complete
resource "digitalocean_app" "budget_app" {
  # Ensure database and migrations are completed first
  depends_on = [
    digitalocean_app.budget_migrations
  ]
  
  # Terraform will automatically detect if image tag changed
  # If image tag is same (cache hit), no redeployment needed

  spec {
    name   = var.app_name
    region = var.region
    
    # Enable VPC networking for database access
    vpc {
      id = var.vpc_id
    }

    # Main application service
    service {
      name               = "web"
      instance_count      = var.instance_count
      instance_size_slug = var.instance_size

      image {
        registry_type = "GHCR"
        registry      = "ghcr.io"
        repository    = var.github_repo
        tag           = var.docker_image_tag
      }

      # Environment variables (with BUDGET_ prefix for viper)
      env {
        key   = "BUDGET_DATABASE_URL"
        value = var.database_url
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
        value = var.environment
        scope = "RUN_TIME"
      }

      env {
        key   = "BUDGET_LOG_LEVEL"
        value = "info"
        scope = "RUN_TIME"
      }

      # Health check - faster since migrations handled by pre-deploy job
      health_check {
        http_path                = "/health"
        initial_delay_seconds    = var.health_check_initial_delay
        period_seconds           = var.health_check_period
        timeout_seconds         = var.health_check_timeout
        failure_threshold        = var.health_check_failure_threshold
        success_threshold        = 1
      }

      # HTTP port
      http_port = 8080
    }
  }
}

# Outputs for app module
output "app_id" {
  description = "Main application ID"
  value       = digitalocean_app.budget_app.id
}

output "app_urn" {
  description = "Main application URN"
  value       = digitalocean_app.budget_app.urn
}

output "app_default_ingress" {
  description = "Main application default ingress URL"
  value       = digitalocean_app.budget_app.default_ingress
}

output "migration_app_id" {
  description = "Migration application ID"
  value       = digitalocean_app.budget_migrations.id
}

output "migration_app_urn" {
  description = "Migration application URN"
  value       = digitalocean_app.budget_migrations.urn
}

output "migration_app" {
  description = "Migration application resource"
  value       = digitalocean_app.budget_migrations
}

output "app" {
  description = "Main application resource"
  value       = digitalocean_app.budget_app
}