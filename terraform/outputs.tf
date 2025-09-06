# Application outputs
output "app_ip_address" {
  description = "Public IP address of the application droplet"
  value       = digitalocean_droplet.budget_app.ipv4_address
}

output "app_private_ip" {
  description = "Private IP address of the application droplet"
  value       = digitalocean_droplet.budget_app.ipv4_address_private
}

# No load balancer in cost-optimized setup

# Database outputs (only if managed database is used)
output "database_host" {
  description = "Database host (private) - only available if use_managed_db is true"
  value       = var.use_managed_db ? digitalocean_database_cluster.budget_db[0].private_host : "SQLite (local file)"
  sensitive   = true
}

output "database_type" {
  description = "Database type being used"
  value       = var.use_managed_db ? "PostgreSQL (managed)" : "SQLite (local)"
}

output "database_connection_string" {
  description = "Database connection string"
  value       = var.use_managed_db ? "postgres://${digitalocean_database_user.budget_user[0].name}:${digitalocean_database_user.budget_user[0].password}@${digitalocean_database_cluster.budget_db[0].private_host}:${digitalocean_database_cluster.budget_db[0].port}/${digitalocean_database_db.budget_database[0].name}?sslmode=require" : "sqlite:///app/data/budget.db"
  sensitive   = true
}

# VPC outputs
output "vpc_id" {
  description = "ID of the VPC"
  value       = digitalocean_vpc.budget_vpc.id
}

output "vpc_ip_range" {
  description = "IP range of the VPC"
  value       = digitalocean_vpc.budget_vpc.ip_range
}

# Domain outputs (if configured)
output "domain_name" {
  description = "Domain name (if configured)"
  value       = var.domain_name != "" ? digitalocean_domain.budget_domain[0].name : null
}

output "application_url" {
  description = "Application URL"
  value       = var.domain_name != "" ? "http://${var.domain_name}:${var.app_port}" : "http://${digitalocean_droplet.budget_app.ipv4_address}:${var.app_port}"
}

# Cost optimization info
output "estimated_monthly_cost" {
  description = "Estimated monthly cost in USD"
  value       = var.use_managed_db ? "$19 (droplet $4 + database $15)" : "$4 (droplet only with SQLite)"
}

# Auto-termination info
output "auto_termination_info" {
  description = "Auto-termination configuration"
  value       = "Deployment will auto-terminate after ${var.auto_terminate_minutes} minutes"
}