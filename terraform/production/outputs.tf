# Re-export module outputs
output "app_url" {
  description = "URL of the deployed application"
  value       = module.budget_app.app_url
}

output "app_id" {
  description = "DigitalOcean App Platform application ID"
  value       = module.budget_app.app_id
}

output "migration_app_id" {
  description = "DigitalOcean App Platform migration application ID"
  value       = module.budget_app.migration_app_id
}

output "deployment_id" {
  description = "Unique deployment identifier"
  value       = local.deployment_id
}

# Database outputs (for reference)
output "database_host" {
  description = "Managed PostgreSQL host"
  value       = digitalocean_database_cluster.budget_db.private_host
  sensitive   = true
}

output "database_connection_string" {
  description = "Database connection string"
  value       = "postgres://${digitalocean_database_user.budget_user.name}:${digitalocean_database_user.budget_user.password}@${digitalocean_database_cluster.budget_db.private_host}:${digitalocean_database_cluster.budget_db.port}/${digitalocean_database_db.budget_database.name}?sslmode=require"
  sensitive   = true
}

# Domain info
output "domain_name" {
  description = "Pre-allocated domain name"
  value       = data.digitalocean_domain.existing_domain.name
}

# Project info
output "project_id" {
  description = "DigitalOcean project ID"
  value       = data.digitalocean_project.budget.id
}

