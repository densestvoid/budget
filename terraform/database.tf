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

# Variables for database module
variable "database_name" {
  description = "Name of the database"
  type        = string
}

variable "database_user_name" {
  description = "Name of the database user"
  type        = string
}

variable "database_size" {
  description = "Database cluster size"
  type        = string
  default     = "db-s-1vcpu-1gb"
}

variable "region" {
  description = "DigitalOcean region"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID for private networking"
  type        = string
}

variable "tags" {
  description = "Tags to apply to resources"
  type        = list(string)
  default     = []
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

# Outputs for database module
output "database_cluster_id" {
  description = "Database cluster ID"
  value       = digitalocean_database_cluster.budget_db.id
}

output "database_cluster_urn" {
  description = "Database cluster URN"
  value       = digitalocean_database_cluster.budget_db.urn
}

output "database_url" {
  description = "Database connection URL"
  value       = "postgres://${digitalocean_database_user.budget_user.name}:${digitalocean_database_user.budget_user.password}@${digitalocean_database_cluster.budget_db.private_host}:${digitalocean_database_cluster.budget_db.port}/${digitalocean_database_db.budget_database.name}?sslmode=require&search_path=budget"
  sensitive   = true
}

output "database_health_check" {
  description = "Database health check resource"
  value       = null_resource.database_health_check
}

output "database_schema_setup" {
  description = "Database schema setup resource"
  value       = null_resource.database_schema_setup
}