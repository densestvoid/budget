# Outputs for production deployment
output "app_default_ingress" {
  description = "Main application default ingress URL"
  value       = module.app.app_default_ingress
}

output "app_id" {
  description = "Main application ID"
  value       = module.app.app_id
}

output "migration_app_id" {
  description = "Migration application ID"
  value       = module.app.migration_app_id
}