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

output "vpc_id" {
  description = "VPC ID (created or existing)"
  value       = local.vpc_id
}

output "app_urn" {
  description = "DigitalOcean App Platform application URN for project assignment"
  value       = digitalocean_app.budget_app.urn
}

output "migration_app_urn" {
  description = "DigitalOcean App Platform migration application URN for project assignment"
  value       = digitalocean_app.budget_migrations.urn
}

