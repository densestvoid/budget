# Application outputs
output "app_default_ingress" {
  description = "Main application default ingress URL"
  value       = digitalocean_app.budget_app.default_ingress
}

output "app_id" {
  description = "Main application ID"
  value       = digitalocean_app.budget_app.id
}

output "migration_app_id" {
  description = "Migration application ID"
  value       = digitalocean_app.budget_migrations.id
}

# Database outputs
output "database_url" {
  description = "Database connection URL"
  value       = "postgres://${digitalocean_database_user.budget_user.name}:${digitalocean_database_user.budget_user.password}@${digitalocean_database_cluster.budget_db.private_host}:${digitalocean_database_cluster.budget_db.port}/${digitalocean_database_db.budget_database.name}?sslmode=require"
  sensitive   = true
}

# VPC outputs
output "vpc_id" {
  description = "VPC ID"
  value       = digitalocean_vpc.budget_vpc.id
}