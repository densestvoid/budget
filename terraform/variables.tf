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
  default     = "nyc1"
  validation {
    condition = contains([
      "nyc1", "nyc3", "ams2", "ams3", "sfo1", "sfo2", "sfo3",
      "sgp1", "lon1", "fra1", "tor1", "blr1", "syd1"
    ], var.region)
    error_message = "Region must be a valid DigitalOcean region."
  }
}

# Use containerized PostgreSQL instead of managed PostgreSQL for cost optimization
variable "use_managed_db" {
  description = "Whether to use managed database (expensive) or containerized PostgreSQL (cheap)"
  type        = bool
  default     = false
}

# Database configuration - only used if use_managed_db is true
variable "db_size" {
  description = "Size of the database cluster (only used if use_managed_db is true)"
  type        = string
  default     = "db-s-1vcpu-1gb"
  validation {
    condition = contains([
      "db-s-1vcpu-1gb", "db-s-1vcpu-2gb", "db-s-2vcpu-4gb",
      "db-s-4vcpu-8gb", "db-s-6vcpu-16gb"
    ], var.db_size)
    error_message = "Database size must be a valid DigitalOcean database size."
  }
}

# Docker image configuration for App Platform
variable "github_repo" {
  description = "GitHub repository (user/repo format)"
  type        = string
  default     = ""
}

variable "docker_image_tag" {
  description = "Docker image tag to deploy"
  type        = string
  default     = "latest"
}

variable "github_token" {
  description = "GitHub token for pulling from GHCR"
  type        = string
  sensitive   = true
  default     = ""
}

# Application configuration
variable "app_port" {
  description = "Port the application runs on"
  type        = string
  default     = "8080"
}

# Domain configuration (optional)
variable "domain_name" {
  description = "Domain name for the application (optional)"
  type        = string
  default     = ""
}

# Environment variables
variable "app_env" {
  description = "Application environment"
  type        = string
  default     = "production"
}

variable "log_level" {
  description = "Application log level"
  type        = string
  default     = "info"
}

# Auto-termination configuration
variable "auto_terminate_minutes" {
  description = "Number of minutes after which to automatically terminate the deployment (0 to disable)"
  type        = number
  default     = 30
}

# GitHub deployment configuration
variable "github_branch" {
  description = "GitHub branch that triggered this deployment"
  type        = string
  default     = "main"
}