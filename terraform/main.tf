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

# Create a managed PostgreSQL database (only if use_managed_db is true)
resource "digitalocean_database_cluster" "budget_db" {
  count      = var.use_managed_db ? 1 : 0
  name       = "budget-db-${random_id.deployment.hex}"
  engine     = "pg"
  version    = "16"
  size       = var.db_size
  region     = var.region
  node_count = 1

  tags = ["budget", "database", "production"]
}

# Create a database within the cluster (only if use_managed_db is true)
resource "digitalocean_database_db" "budget_database" {
  count      = var.use_managed_db ? 1 : 0
  cluster_id = digitalocean_database_cluster.budget_db[0].id
  name       = "budget"
}

# Create a database user (only if use_managed_db is true)
resource "digitalocean_database_user" "budget_user" {
  count      = var.use_managed_db ? 1 : 0
  cluster_id = digitalocean_database_cluster.budget_db[0].id
  name       = "budget_app"
}

# Create DigitalOcean App Platform application
resource "digitalocean_app" "budget_app" {
  spec {
    name   = "budget-app-${random_id.deployment.hex}"
    region = var.region

    # Main application service
    service {
      name               = "web"
      environment_slug   = "docker"
      instance_count     = 1
      instance_size_slug = "basic-xxs"  # $5/month: 0.5 vCPU, 512MB RAM

      image {
        registry_type = "GHCR"
        registry      = "ghcr.io"
        repository    = "${var.github_repo}/budget-app"
        tag           = var.docker_image_tag
        registry_credentials = var.github_token
      }

      # Environment variables
      env {
        key   = "DATABASE_URL"
        value = var.use_managed_db ? "postgres://${digitalocean_database_user.budget_user[0].name}:${digitalocean_database_user.budget_user[0].password}@${digitalocean_database_cluster.budget_db[0].private_host}:${digitalocean_database_cluster.budget_db[0].port}/${digitalocean_database_db.budget_database[0].name}?sslmode=require" : "postgres://budget_user:budget_password@postgres:5432/budget?sslmode=disable"
        scope = "RUN_TIME"
        type  = "SECRET"
      }

      env {
        key   = "PORT"
        value = var.app_port
        scope = "RUN_TIME"
      }

      env {
        key   = "ENV"
        value = "production"
        scope = "RUN_TIME"
      }

      env {
        key   = "LOG_LEVEL"
        value = var.log_level
        scope = "RUN_TIME"
      }

      # Health check
      health_check {
        http_path = "/health"
      }

      # HTTP port
      http_port = var.app_port

      # Routes
      routes {
        path = "/"
      }
    }

    # PostgreSQL database service (only if not using managed DB)
    dynamic "service" {
      for_each = var.use_managed_db ? [] : [1]
      content {
        name               = "postgres"
        environment_slug   = "docker"
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
          value = "budget_user"
          scope = "RUN_TIME"
        }

        env {
          key   = "POSTGRES_PASSWORD"
          value = "budget_password"
          scope = "RUN_TIME"
          type  = "SECRET"
        }

        # Internal service - no external routes
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