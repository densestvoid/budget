# Outputs for app module
output "app_id" {
  description = "Main application ID"
  value       = digitalocean_app.budget_app.id
}

output "app_urn" {
  description = "Main application URN"
  value       = digitalocean_app.budget_app.urn
}

output "app_default_ingress" {
  description = "Main application default ingress URL"
  value       = digitalocean_app.budget_app.default_ingress
}

output "migration_app_id" {
  description = "Migration application ID"
  value       = digitalocean_app.budget_migrations.id
}

output "migration_app_urn" {
  description = "Migration application URN"
  value       = digitalocean_app.budget_migrations.urn
}

output "migration_app" {
  description = "Migration application resource"
  value       = digitalocean_app.budget_migrations
}

output "app" {
  description = "Main application resource"
  value       = digitalocean_app.budget_app
}