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

# VPC is always provided by the parent configuration
# This module expects vpc_id to always be set (not null)
locals {
  vpc_id = var.vpc_id
}

# Reference existing DigitalOcean project
data "digitalocean_project" "budget" {
  name = var.project_name
}

# Health check to ensure database cluster is ready
# Uses DigitalOcean API instead of direct connection (private endpoints not accessible from GitHub Actions)
resource "null_resource" "database_health_check" {
  provisioner "local-exec" {
    command     = <<-EOT
      echo "🔍 Checking database cluster status via DigitalOcean API..."
      
      CLUSTER_ID="${var.database_cluster_id}"
      DO_TOKEN="${var.do_token}"
      
      if [ -z "$CLUSTER_ID" ] || [ -z "$DO_TOKEN" ]; then
        echo "⚠️ Missing cluster ID or DO token - skipping health check"
        exit 0
      fi
      
      # Check if cluster already exists and is online
      for i in {1..60}; do
        echo "Status check attempt $i/60..."
        
        # Get cluster status from DigitalOcean API
        STATUS_RESPONSE=$(curl -s -H "Authorization: Bearer $DO_TOKEN" \
          "https://api.digitalocean.com/v2/databases/$CLUSTER_ID" 2>/dev/null)
        
        if [ $? -ne 0 ]; then
          echo "⚠️ Failed to fetch cluster status from API, waiting 10s..."
          sleep 10
          continue
        fi
        
        # Extract status from API response (handle different response structures)
        STATUS=$(echo "$STATUS_RESPONSE" | jq -r '.database.status.state // .database.status // .status.state // .status // "unknown"' 2>/dev/null || echo "unknown")
        
        # If status is still unknown, check if we got a valid response
        if [ "$STATUS" = "unknown" ]; then
          ERROR=$(echo "$STATUS_RESPONSE" | jq -r '.error // .message // empty' 2>/dev/null || echo "")
          if [ -n "$ERROR" ]; then
            echo "⚠️ API error: $ERROR"
          else
            echo "⚠️ Could not parse status from API response"
          fi
        fi
        
        echo "📊 Database cluster status: $STATUS"
        
        # Check if cluster is ready (online/running)
        if [ "$STATUS" = "online" ] || [ "$STATUS" = "running" ]; then
          echo "✅ Database cluster is ready (status: $STATUS)"
          exit 0
        elif [ "$STATUS" = "creating" ] || [ "$STATUS" = "forking" ] || [ "$STATUS" = "resizing" ]; then
          echo "⏳ Database cluster is still $STATUS, waiting 10s..."
          sleep 10
        else
          echo "⚠️ Database cluster status is $STATUS, waiting 10s..."
          sleep 10
        fi
      done
      
      echo "❌ Database cluster failed to become ready after 10 minutes"
      echo "ℹ️ This may be acceptable if the cluster already exists and is in use"
      echo "⚠️ Continuing anyway - migration app will handle connection errors"
      exit 0  # Don't fail the deployment - let the migration app handle it
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

