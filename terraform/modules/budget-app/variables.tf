# DigitalOcean API Token
variable "do_token" {
  description = "DigitalOcean API token"
  type        = string
  sensitive   = true
}

# Region configuration
variable "region" {
  description = "DigitalOcean region"
  type        = string
  validation {
    condition = contains([
      "nyc1", "nyc3", "ams2", "ams3", "sfo1", "sfo2", "sfo3",
      "sgp1", "lon1", "fra1", "tor1", "blr1", "syd1"
    ], var.region)
    error_message = "Region must be a valid DigitalOcean region."
  }
}

# Deployment identification
variable "deployment_id" {
  description = "Unique deployment ID"
  type        = string
}

# Project name
variable "project_name" {
  description = "DigitalOcean project name"
  type        = string
}

variable "docker_image_tag" {
  description = "Docker image tag to deploy"
  type        = string
}

# Database configuration
variable "database_cluster_id" {
  description = "ID of database cluster (required)"
  type        = string
}

variable "database_name" {
  description = "Name of database (required)"
  type        = string
}

variable "database_user_name" {
  description = "Name of database user (required)"
  type        = string
}

variable "database_user_password" {
  description = "Password of database user (required)"
  type        = string
  sensitive   = true
}

variable "database_private_host" {
  description = "Private host of database cluster (required)"
  type        = string
}

variable "database_port" {
  description = "Port of database cluster (required)"
  type        = number
}

variable "vpc_id" {
  description = "Optional VPC ID to use instead of creating a new one (for production with existing database)"
  type        = string
  default     = null
}

