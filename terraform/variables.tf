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

# Deployment identification
variable "deployment_id" {
  description = "Unique deployment ID for this PR (format: pr-{number}-{branch})"
  type        = string
}

# TFE token for workspace management
variable "tfe_token" {
  description = "Terraform Cloud API token for workspace management"
  type        = string
  sensitive   = true
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

variable "github_token" {
  description = "GitHub token for pulling from GHCR"
  type        = string
  sensitive   = true
}

# Domain configuration (optional)
variable "domain_name" {
  description = "Domain name for the application (optional)"
  type        = string
  default     = ""
}

# Auto-termination configuration
variable "auto_terminate_minutes" {
  description = "Number of minutes after which to automatically terminate the deployment"
  type        = number
  default     = 30
}