# Variables for app module
variable "app_name" {
  description = "Name of the main application"
  type        = string
}

variable "migration_app_name" {
  description = "Name of the migration application"
  type        = string
}

variable "github_repo" {
  description = "GitHub repository (user/repo format)"
  type        = string
}

variable "docker_image_tag" {
  description = "Docker image tag to deploy"
  type        = string
}

variable "region" {
  description = "DigitalOcean region"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID for private networking"
  type        = string
}

variable "database_url" {
  description = "Database connection URL"
  type        = string
  sensitive   = true
}

variable "environment" {
  description = "Environment name (production, staging, etc.)"
  type        = string
}

variable "instance_count" {
  description = "Number of app instances"
  type        = number
  default     = 1
}

variable "instance_size" {
  description = "App instance size"
  type        = string
  default     = "basic-xxs"
}

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

variable "database_dependencies" {
  description = "Database dependencies for migration app"
  type        = list(any)
  default     = []
}