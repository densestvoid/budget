terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.0"
    }
    external = {
      source  = "hashicorp/external"
      version = "~> 2.0"
    }
  }
}

# Auto-detect GitHub repository from environment variable
data "external" "github_repo" {
  program = ["sh", "-c", "printf '{\"repo\":\"%s\"}' \"$${GITHUB_REPOSITORY:-}\""]
}

locals {
  github_repo = data.external.github_repo.result.repo != "" ? data.external.github_repo.result.repo : (
    # Fallback validation - should never happen in GitHub Actions
    # For local testing, set GITHUB_REPOSITORY environment variable
    null
  )
}

# Create VPC for private networking (only if not using existing VPC)
resource "digitalocean_vpc" "budget_vpc" {
  count    = var.vpc_id == null ? 1 : 0
  name     = var.deployment_id
  region   = var.region
  ip_range = "172.16.0.0/16"  # Private IP range (avoiding conflicts)
  
  # Note: VPC doesn't support tags, but name includes deployment_id for identification
}

# Local to reference either existing or created VPC
locals {
  vpc_id = var.vpc_id != null ? var.vpc_id : digitalocean_vpc.budget_vpc[0].id
}

# Reference existing DigitalOcean project
data "digitalocean_project" "budget" {
  name = var.project_name
}

# Health check to ensure database is ready for connections
resource "null_resource" "database_health_check" {
  provisioner "local-exec" {
    command     = <<-EOT
      echo "🔍 Testing database connectivity..."
      
      # Install postgresql-client if not available
      which pg_isready || (echo "Installing postgresql-client..." && apt-get update && apt-get install -y postgresql-client)
      
      for i in {1..30}; do
        echo "Connection attempt $i/30..."
        if pg_isready -h ${var.database_private_host} -p ${var.database_port} -U ${var.database_user_name}; then
          echo "✅ Database is ready for connections"
          exit 0
        else
          echo "⏳ Database not ready yet, waiting 10s..."
          sleep 10
        fi
      done
      echo "❌ Database failed to become ready after 5 minutes"
      exit 1
    EOT
    interpreter = ["/bin/bash", "-c"]
  }
}

# Create migration app that runs migrations and exits
# Schema migrations always execute
resource "digitalocean_app" "budget_migrations" {
  depends_on = [null_resource.database_health_check]
  
  spec {
    name   = "${var.deployment_id}-migrations"
    region = var.region
    
    # Enable VPC networking for database access
    vpc {
      id = local.vpc_id
    }

    # Migration job - runs once and exits
    job {
      name = "migrate"
      kind = "PRE_DEPLOY"  # Runs before main service deployment
      
      image {
        registry_type = "GHCR"
        registry      = "ghcr.io"
        repository    = "${local.github_repo}/budget-app"
        tag           = var.docker_image_tag
      }

      # Environment variables for migration
      env {
        key   = "BUDGET_DATABASE_URL"
        value = "postgres://${var.database_user_name}:${var.database_user_password}@${var.database_private_host}:${var.database_port}/${var.database_name}?sslmode=require&search_path=budget"
        scope = "RUN_TIME"
        type  = "SECRET"
      }

      env {
        key   = "BUDGET_ENV"
        value = "production"
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
  depends_on = [digitalocean_app.budget_migrations]
  
  spec {
    name   = var.deployment_id
    region = var.region
    
    # Enable VPC networking for database access
    vpc {
      id = local.vpc_id
    }

    # Main application service
    service {
      name               = "web"
      instance_count     = 1
      instance_size_slug = "basic-xxs"  # $5/month: 0.5 vCPU, 512MB RAM

      image {
        registry_type = "GHCR"
        registry      = "ghcr.io"
        repository    = "${local.github_repo}/budget-app"
        tag           = var.docker_image_tag
      }

      # Environment variables (with BUDGET_ prefix for viper)
      env {
        key   = "BUDGET_DATABASE_URL"
        value = "postgres://${var.database_user_name}:${var.database_user_password}@${var.database_private_host}:${var.database_port}/${var.database_name}?sslmode=require&search_path=budget"
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
        value = "info"
        scope = "RUN_TIME"
      }

      # Health check - faster since migrations handled by pre-deploy job
      health_check {
        http_path                = "/health"
        initial_delay_seconds    = 30
        period_seconds           = 10
        timeout_seconds          = 5
        failure_threshold        = 3
        success_threshold        = 1
      }

      # HTTP port
      http_port = 8080
    }
  }
}

# Assign resources to the existing project
resource "digitalocean_project_resources" "budget_resources" {
  project = data.digitalocean_project.budget.id
  resources = [
    digitalocean_app.budget_migrations.urn,
    digitalocean_app.budget_app.urn
    # Note: VPC cannot be assigned to projects (not in supported resource types)
  ]
  
  depends_on = [
    digitalocean_app.budget_migrations,
    digitalocean_app.budget_app
  ]
}

