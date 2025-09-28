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
        repository    = "${var.github_repo}/budget-app"
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
        repository    = "${var.github_repo}/budget-app"
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