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

# Droplet configuration - optimized for minimal cost
variable "droplet_size" {
  description = "Size of the application droplet"
  type        = string
  default     = "s-1vcpu-512mb-10gb"  # Cheapest option at $4/month
  validation {
    condition = contains([
      "s-1vcpu-512mb-10gb", "s-1vcpu-1gb", "s-1vcpu-2gb", "s-2vcpu-2gb", "s-2vcpu-4gb",
      "s-4vcpu-8gb", "s-6vcpu-16gb", "s-8vcpu-32gb"
    ], var.droplet_size)
    error_message = "Droplet size must be a valid DigitalOcean size."
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

# SSH Key configuration
variable "ssh_public_key_path" {
  description = "Path to SSH public key file"
  type        = string
  default     = "~/.ssh/id_rsa.pub"
}

variable "ssh_private_key_path" {
  description = "Path to SSH private key file"
  type        = string
  default     = "~/.ssh/id_rsa"
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
  description = "Number of minutes after which to automatically terminate the deployment"
  type        = number
  default     = 30
}

# GitHub deployment configuration
variable "github_repo" {
  description = "GitHub repository for deployment triggers"
  type        = string
  default     = ""
}

variable "github_branch" {
  description = "GitHub branch that triggered this deployment"
  type        = string
  default     = "main"
}