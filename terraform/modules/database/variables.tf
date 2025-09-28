# Variables for database module
variable "database_name" {
  description = "Name of the database"
  type        = string
}

variable "database_user_name" {
  description = "Name of the database user"
  type        = string
}

variable "database_size" {
  description = "Database cluster size"
  type        = string
  default     = "db-s-1vcpu-1gb"
}

variable "region" {
  description = "DigitalOcean region"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID for private networking"
  type        = string
}

variable "tags" {
  description = "Tags to apply to resources"
  type        = list(string)
  default     = []
}