# Database module for both PR and Production deployments
# This module creates a managed PostgreSQL database cluster

terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

# Managed PostgreSQL database with private VPC networking
resource "digitalocean_database_cluster" "budget_db" {
  name                 = var.database_name
  engine               = "pg"
  version              = "16"
  size                 = var.database_size
  region               = var.region
  node_count           = 1
  private_network_uuid = var.vpc_id

  tags = var.tags
  
  # Database cluster is stable once created
  # Password is auto-generated and managed by DigitalOcean
}

# Create database within the cluster
resource "digitalocean_database_db" "budget_database" {
  cluster_id = digitalocean_database_cluster.budget_db.id
  name       = var.database_name
}

# Create database user
resource "digitalocean_database_user" "budget_user" {
  cluster_id = digitalocean_database_cluster.budget_db.id
  name       = var.database_user_name
  
  # Database user password is auto-generated and stable
  # DigitalOcean manages password lifecycle
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