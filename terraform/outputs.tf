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

# Database info
output "database_type" {
  description = "Database type being used"
  value       = "PostgreSQL (containerized in App Platform)"
}

output "database_connection_string" {
  description = "Database connection string"
  value       = "postgres://budget_user:budget_password@postgres:5432/budget?sslmode=disable"
  sensitive   = true
}

# Cost optimization info
output "estimated_monthly_cost" {
  description = "Estimated monthly cost in USD"
  value       = "$10 (web service $5 + postgres service $5)"
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