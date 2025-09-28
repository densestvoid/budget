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