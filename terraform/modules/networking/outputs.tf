# Outputs for networking module
output "vpc_id" {
  description = "VPC ID"
  value       = digitalocean_vpc.budget_vpc.id
}

output "vpc_name" {
  description = "VPC name"
  value       = digitalocean_vpc.budget_vpc.name
}