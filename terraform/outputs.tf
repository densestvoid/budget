# Application outputs
output "app_ip_address" {
  description = "Public IP address of the application droplet"
  value       = digitalocean_droplet.budget_app.ipv4_address
}

output "app_private_ip" {
  description = "Private IP address of the application droplet"
  value       = digitalocean_droplet.budget_app.ipv4_address_private
}

# Load balancer outputs
output "load_balancer_ip" {
  description = "IP address of the load balancer"
  value       = digitalocean_loadbalancer.budget_lb.ip
}

output "load_balancer_status" {
  description = "Status of the load balancer"
  value       = digitalocean_loadbalancer.budget_lb.status
}

# Database outputs
output "database_host" {
  description = "Database host (private)"
  value       = digitalocean_database_cluster.budget_db.private_host
  sensitive   = true
}

output "database_port" {
  description = "Database port"
  value       = digitalocean_database_cluster.budget_db.port
}

output "database_name" {
  description = "Database name"
  value       = digitalocean_database_db.budget_database.name
}

output "database_user" {
  description = "Database username"
  value       = digitalocean_database_user.budget_user.name
  sensitive   = true
}

output "database_password" {
  description = "Database password"
  value       = digitalocean_database_user.budget_user.password
  sensitive   = true
}

output "database_connection_string" {
  description = "Full database connection string"
  value       = "postgres://${digitalocean_database_user.budget_user.name}:${digitalocean_database_user.budget_user.password}@${digitalocean_database_cluster.budget_db.private_host}:${digitalocean_database_cluster.budget_db.port}/${digitalocean_database_db.budget_database.name}?sslmode=require"
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
  value       = var.domain_name != "" ? "https://${var.domain_name}" : "http://${digitalocean_loadbalancer.budget_lb.ip}"
}