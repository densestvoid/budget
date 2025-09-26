# App Platform outputs
output "app_url" {
  description = "URL of the deployed application"
  value       = digitalocean_app.budget_app.default_ingress
}

output "app_id" {
  description = "DigitalOcean App Platform application ID"
  value       = digitalocean_app.budget_app.id
}

output "migration_app_id" {
  description = "DigitalOcean App Platform migration application ID"
  value       = digitalocean_app.budget_migrations.id
}

output "deployment_id" {
  description = "Unique deployment identifier"
  value       = var.deployment_id
}

# Database outputs
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

# Project info
output "project_id" {
  description = "Existing DigitalOcean project ID"
  value       = data.digitalocean_project.budget.id
}

# Cost information
output "estimated_total_cost" {
  description = "Total estimated cost for 30-minute deployment"
  value       = "$0.02"  # ($15 DB + $5 App + $5 Migration) ÷ 730.56 hours/month × 0.5 hours = $0.017, rounded to $0.02
}

# Termination info
output "termination_info" {
  description = "Auto-termination details"
  value = {
    minutes_until_termination = var.auto_terminate_minutes
    method = "workflow_dispatch with environment wait timer"
  }
}