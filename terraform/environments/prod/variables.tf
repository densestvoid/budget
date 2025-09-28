# Production environment variables

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

# Docker image configuration for App Platform
variable "github_repo" {
  description = "GitHub repository (user/repo format)"
  type        = string
}

variable "docker_image_tag" {
  description = "Docker image tag to deploy"
  type        = string
}

# Database configuration
variable "database_size" {
  description = "Database cluster size"
  type        = string
  default     = "db-s-1vcpu-1gb"
  validation {
    condition = contains([
      "db-s-1vcpu-1gb", "db-s-1vcpu-2gb", "db-s-2vcpu-4gb",
      "db-s-4vcpu-8gb", "db-s-6vcpu-16gb", "db-s-8vcpu-32gb"
    ], var.database_size)
    error_message = "Database size must be a valid DigitalOcean database size."
  }
}

# App configuration
variable "instance_count" {
  description = "Number of app instances"
  type        = number
  default     = 1
  validation {
    condition     = var.instance_count >= 1 && var.instance_count <= 10
    error_message = "Instance count must be between 1 and 10."
  }
}

variable "instance_size" {
  description = "App instance size"
  type        = string
  default     = "basic-xxs"
  validation {
    condition = contains([
      "basic-xxs", "basic-xs", "basic-s", "basic-m", "basic-l", "basic-xl"
    ], var.instance_size)
    error_message = "Instance size must be a valid DigitalOcean App Platform size."
  }
}

# Health check configuration
variable "health_check_initial_delay" {
  description = "Initial delay before health checks start (seconds)"
  type        = number
  default     = 30
}

variable "health_check_period" {
  description = "Health check period (seconds)"
  type        = number
  default     = 15
}

variable "health_check_timeout" {
  description = "Health check timeout (seconds)"
  type        = number
  default     = 5
}

variable "health_check_failure_threshold" {
  description = "Number of consecutive failures before marking unhealthy"
  type        = number
  default     = 3
}