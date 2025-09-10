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

# Create VPC for private networking
resource "digitalocean_vpc" "budget_vpc" {
  name     = substr("budget-vpc-${var.deployment_id}", 0, 32)  # VPC name limit
  region   = var.region
  ip_range = "172.16.0.0/16"  # Private IP range (avoiding conflicts)
  
  # Note: VPC doesn't support tags, but name includes deployment_id for identification
}

# Managed PostgreSQL database with private VPC networking
resource "digitalocean_database_cluster" "budget_db" {
  name                 = "budget-db-${var.deployment_id}"
  engine               = "pg"
  version              = "16"
  size                 = "db-s-1vcpu-1gb"  # Cheapest managed DB option
  region               = var.region
  node_count           = 1
  private_network_uuid = digitalocean_vpc.budget_vpc.id

  tags = ["deployment-id:${var.deployment_id}"]
}

# Create database within the cluster
resource "digitalocean_database_db" "budget_database" {
  cluster_id = digitalocean_database_cluster.budget_db.id
  name       = "budget-${var.deployment_id}"
}

# Create database user
resource "digitalocean_database_user" "budget_user" {
  cluster_id = digitalocean_database_cluster.budget_db.id
  name       = "budget-app-${var.deployment_id}"
}

# Create budget schema and grant privileges using null_resource
resource "null_resource" "database_schema_setup" {
  depends_on = [
    digitalocean_database_cluster.budget_db,
    digitalocean_database_db.budget_database,
    digitalocean_database_user.budget_user
  ]
  
  provisioner "local-exec" {
    command = <<-EOT
      echo "🗄️ Setting up database schema and permissions..."
      
      # Install postgresql-client if not available
      which psql || (echo "Installing postgresql-client..." && apt-get update && apt-get install -y postgresql-client)
      
      # Connect as admin user to create schema and grant permissions
      ADMIN_URL="postgres://${digitalocean_database_cluster.budget_db.user}:${digitalocean_database_cluster.budget_db.password}@${digitalocean_database_cluster.budget_db.host}:${digitalocean_database_cluster.budget_db.port}/${digitalocean_database_db.budget_database.name}?sslmode=require"
      
      echo "Creating budget schema and setting up permissions..."
      psql "$ADMIN_URL" <<SQL
        -- Create budget schema
        CREATE SCHEMA IF NOT EXISTS budget;
        
        -- Grant all privileges on budget schema ONLY to our user
        GRANT ALL PRIVILEGES ON SCHEMA budget TO "${digitalocean_database_user.budget_user.name}";
        
        -- Set default privileges for future tables in budget schema
        ALTER DEFAULT PRIVILEGES IN SCHEMA budget GRANT ALL ON TABLES TO "${digitalocean_database_user.budget_user.name}";
        ALTER DEFAULT PRIVILEGES IN SCHEMA budget GRANT ALL ON SEQUENCES TO "${digitalocean_database_user.budget_user.name}";
        ALTER DEFAULT PRIVILEGES IN SCHEMA budget GRANT ALL ON FUNCTIONS TO "${digitalocean_database_user.budget_user.name}";
        
        -- Make budget user the owner of budget schema
        ALTER SCHEMA budget OWNER TO "${digitalocean_database_user.budget_user.name}";
SQL
      
      if [ $? -eq 0 ]; then
        echo "✅ Database schema and permissions configured successfully"
      else
        echo "❌ Failed to configure database schema and permissions"
        exit 1
      fi
    EOT
  }
}

# Note: No initial firewall rules = allow all connections during deployment
# Database is open during app deployment, then restricted by final firewall

# Health check to ensure database is ready for connections
resource "null_resource" "database_health_check" {
  depends_on = [
    digitalocean_database_cluster.budget_db,
    digitalocean_database_db.budget_database,
    digitalocean_database_user.budget_user
  ]
  
  provisioner "local-exec" {
    command = <<-EOT
      echo "🔍 Testing database connectivity..."
      
      # Install postgresql-client if not available
      which pg_isready || (echo "Installing postgresql-client..." && apt-get update && apt-get install -y postgresql-client)
      
      for i in {1..30}; do
        echo "Connection attempt $i/30..."
        if pg_isready -h ${digitalocean_database_cluster.budget_db.host} -p ${digitalocean_database_cluster.budget_db.port} -U ${digitalocean_database_user.budget_user.name}; then
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
  }
}

# Create migration app that runs migrations and exits
resource "digitalocean_app" "budget_migrations" {
  # Ensure database is ready, health-checked, and schema configured
  depends_on = [
    null_resource.database_health_check,
    null_resource.database_schema_setup
  ]
  
  # Note: App Platform doesn't support tags - using name for identification
  
  # Terraform will automatically detect if image tag changed
  # If image tag is same (cache hit), no redeployment needed

  spec {
    name   = substr("${var.deployment_id}-migrations", 0, 32)  # Trim to 32 chars max
    region = var.region
    
    # Enable VPC networking for database access
    vpc {
      id = digitalocean_vpc.budget_vpc.id
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
        value = "postgres://${digitalocean_database_user.budget_user.name}:${digitalocean_database_user.budget_user.password}@${digitalocean_database_cluster.budget_db.private_host}:${digitalocean_database_cluster.budget_db.port}/${digitalocean_database_db.budget_database.name}?sslmode=require&search_path=budget"
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
  # Ensure database and migrations are completed first
  depends_on = [
    digitalocean_app.budget_migrations
  ]
  
  # Note: App Platform doesn't support tags - using name for identification
  
  # Terraform will automatically detect if image tag changed
  # If image tag is same (cache hit), no redeployment needed

  spec {
    name   = substr(var.deployment_id, 0, 32)  # Trim to 32 chars max
    region = var.region
    
    # Enable VPC networking for database access
    vpc {
      id = digitalocean_vpc.budget_vpc.id
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
        value = "postgres://${digitalocean_database_user.budget_user.name}:${digitalocean_database_user.budget_user.password}@${digitalocean_database_cluster.budget_db.private_host}:${digitalocean_database_cluster.budget_db.port}/${digitalocean_database_db.budget_database.name}?sslmode=require&search_path=budget"
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

# VPC provides network-level security - no additional firewall rules needed
# Database is only accessible within the VPC private network

# Assign resources to the existing project
resource "digitalocean_project_resources" "budget_resources" {
  project = data.digitalocean_project.budget.id
  resources = [
    digitalocean_app.budget_migrations.urn,
    digitalocean_app.budget_app.urn,
    digitalocean_database_cluster.budget_db.urn
    # Note: VPC cannot be assigned to projects (not in supported resource types)
  ]
  
  # Ensure resources are created before assignment
  depends_on = [
    digitalocean_app.budget_migrations,
    digitalocean_app.budget_app,
    digitalocean_database_cluster.budget_db
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