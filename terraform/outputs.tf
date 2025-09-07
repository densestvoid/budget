# App Platform outputs
output "app_url" {
  description = "URL of the deployed application"
  value       = "https://${digitalocean_app.budget_app.default_ingress}"
}

output "app_id" {
  description = "DigitalOcean App Platform application ID"
  value       = digitalocean_app.budget_app.id
}

output "app_status" {
  description = "Application deployment status"
  value       = digitalocean_app.budget_app.active_deployment_id
}

# Database outputs (only if managed database is used)
output "database_host" {
  description = "Database host (private) - only available if use_managed_db is true"
  value       = var.use_managed_db ? digitalocean_database_cluster.budget_db[0].private_host : "PostgreSQL (containerized in app)"
  sensitive   = true
}

output "database_type" {
  description = "Database type being used"
  value       = var.use_managed_db ? "PostgreSQL (managed)" : "PostgreSQL (containerized)"
}

output "database_connection_string" {
  description = "Database connection string"
  value       = var.use_managed_db ? "postgres://${digitalocean_database_user.budget_user[0].name}:${digitalocean_database_user.budget_user[0].password}@${digitalocean_database_cluster.budget_db[0].private_host}:${digitalocean_database_cluster.budget_db[0].port}/${digitalocean_database_db.budget_database[0].name}?sslmode=require" : "Internal PostgreSQL container"
  sensitive   = true
}

# Cost optimization info
output "estimated_monthly_cost" {
  description = "Estimated monthly cost in USD"
  value       = var.use_managed_db ? "$20 (app $5 + database $15)" : "$10 (app $5 + postgres $5)"
}

# Auto-termination info
output "auto_termination_info" {
  description = "Auto-termination configuration"
  value       = var.auto_terminate_minutes > 0 ? "App will auto-terminate after ${var.auto_terminate_minutes} minutes" : "Auto-termination disabled"
}

# Deployment info
output "deployment_info" {
  description = "Deployment information"
  value = {
    app_name     = digitalocean_app.budget_app.spec[0].name
    region       = var.region
    github_repo  = var.github_repo
    github_branch = var.github_branch
    docker_tag   = var.docker_image_tag
  }
}