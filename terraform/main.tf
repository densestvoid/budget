terraform {
  required_version = ">= 1.0"
  
  backend "s3" {
    endpoint                    = "https://nyc3.digitaloceanspaces.com"
    bucket                      = "budget-develop-terraform-states"
    # key will be set dynamically via terraform init -backend-config
    region                      = "us-east-1"
    skip_credentials_validation = true
    skip_requesting_account_id  = true
    skip_metadata_api_check     = true
    skip_region_validation      = true
    skip_s3_checksum            = true
  }
  
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

# Configure the DigitalOcean Provider
provider "digitalocean" {
  token = var.do_token
}

# Reference existing DigitalOcean project
data "digitalocean_project" "budget" {
  name = "budget-develop"
}

# Managed PostgreSQL database with no public access (secured via firewall)
resource "digitalocean_database_cluster" "budget_db" {
  name       = "budget-db-${var.deployment_id}"
  engine     = "pg"
  version    = "16"
  size       = "db-s-1vcpu-1gb"  # Cheapest managed DB option
  region     = var.region
  node_count = 1

  tags = ["deployment-id:${var.deployment_id}"]
}

# Create database within the cluster
resource "digitalocean_database_db" "budget_database" {
  cluster_id = digitalocean_database_cluster.budget_db.id
  name       = "budget"
}

# Create database user
resource "digitalocean_database_user" "budget_user" {
  cluster_id = digitalocean_database_cluster.budget_db.id
  name       = "budget_app"
}

# Create migration app that runs migrations and exits
resource "digitalocean_app" "budget_migrations" {
  # Ensure database is created first
  depends_on = [
    digitalocean_database_cluster.budget_db,
    digitalocean_database_db.budget_database,
    digitalocean_database_user.budget_user
  ]

  spec {
    name   = substr("${var.deployment_id}-migrations", 0, 32)  # Trim to 32 chars max
    region = var.region

    # Migration job - runs once and exits
    job {
      name = "migrate"
      kind = "DEPLOY"  # Runs during deployment, not pre-deploy
      
      image {
        registry_type = "GHCR"
        registry      = "ghcr.io"
        repository    = "${var.github_repo}/budget-app"
        tag           = var.docker_image_tag
      }

      # Environment variables for migration
      env {
        key   = "BUDGET_DATABASE_URL"
        value = "postgres://${digitalocean_database_user.budget_user.name}:${digitalocean_database_user.budget_user.password}@${digitalocean_database_cluster.budget_db.host}:${digitalocean_database_cluster.budget_db.port}/${digitalocean_database_db.budget_database.name}?sslmode=require"
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

      # Run migrations with retry logic and exit
      run_command = "sh -c 'for i in 1 2 3; do echo \"Migration attempt $i/3...\"; if ./budget migrate; then echo \"✅ Migration successful\"; exit 0; else echo \"❌ Migration attempt $i failed\"; if [ $i -eq 3 ]; then echo \"💥 All migration attempts failed\"; exit 1; fi; sleep 10; fi; done'"
    }
  }
}

# Create main application after migrations complete
resource "digitalocean_app" "budget_app" {
  # Ensure database and migrations are completed first
  depends_on = [
    digitalocean_app.budget_migrations
  ]

  spec {
    name   = substr(var.deployment_id, 0, 32)  # Trim to 32 chars max
    region = var.region

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
        value = "postgres://${digitalocean_database_user.budget_user.name}:${digitalocean_database_user.budget_user.password}@${digitalocean_database_cluster.budget_db.host}:${digitalocean_database_cluster.budget_db.port}/${digitalocean_database_db.budget_database.name}?sslmode=require"
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
        initial_delay_seconds    = 30    # Reduced from 60s
        period_seconds           = 10    # Check every 10s
        timeout_seconds          = 5     # 5s timeout per check
        failure_threshold        = 3     # Reduced from 5 (faster startup)
        success_threshold        = 1     # 1 success to mark healthy
      }

      # HTTP port
      http_port = 8080
    }
  }
}

# Configure database firewall to allow both migration and main app access
resource "digitalocean_database_firewall" "budget_db_firewall" {
  cluster_id = digitalocean_database_cluster.budget_db.id
  
  depends_on = [
    digitalocean_app.budget_migrations,
    digitalocean_app.budget_app
  ]

  # Allow access from migration app
  rule {
    type  = "app"
    value = digitalocean_app.budget_migrations.id
  }

  # Allow access from main app
  rule {
    type  = "app"
    value = digitalocean_app.budget_app.id
  }
}

# Assign resources to the existing project
resource "digitalocean_project_resources" "budget_resources" {
  project = data.digitalocean_project.budget.id
  resources = [
    digitalocean_app.budget_migrations.urn,
    digitalocean_app.budget_app.urn,
    digitalocean_database_cluster.budget_db.urn
  ]
  
  # Ensure resources are created before assignment
  depends_on = [
    digitalocean_app.budget_migrations,
    digitalocean_app.budget_app,
    digitalocean_database_cluster.budget_db,
    digitalocean_database_firewall.budget_db_firewall
  ]
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