terraform {
  required_version = ">= 1.0"
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.1"
    }
  }
}

# Configure the DigitalOcean Provider
provider "digitalocean" {
  token = var.do_token
}

# Generate a random ID for unique resource naming
resource "random_id" "deployment" {
  byte_length = 4
}

# Create a VPC
resource "digitalocean_vpc" "budget_vpc" {
  name     = "budget-vpc-${random_id.deployment.hex}"
  region   = var.region
  ip_range = "172.16.0.0/16"  # Use 172.16.x.x range which should be safe

  # Note: VPC resources don't support tags in DigitalOcean
}

# Create a managed PostgreSQL database (only if use_managed_db is true)
resource "digitalocean_database_cluster" "budget_db" {
  count      = var.use_managed_db ? 1 : 0
  name       = "budget-db-${random_id.deployment.hex}"
  engine     = "pg"
  version    = "16"
  size       = var.db_size
  region     = var.region
  node_count = 1

  private_network_uuid = digitalocean_vpc.budget_vpc.id

  tags = ["budget", "database", "production"]
}

# Create a database within the cluster (only if use_managed_db is true)
resource "digitalocean_database_db" "budget_database" {
  count      = var.use_managed_db ? 1 : 0
  cluster_id = digitalocean_database_cluster.budget_db[0].id
  name       = "budget"
}

# Create a database user (only if use_managed_db is true)
resource "digitalocean_database_user" "budget_user" {
  count      = var.use_managed_db ? 1 : 0
  cluster_id = digitalocean_database_cluster.budget_db[0].id
  name       = "budget_app"
}

# No SSH key needed - using cloud-init for deployment

# Create a Droplet for the application
resource "digitalocean_droplet" "budget_app" {
  image    = "ubuntu-22-04-x64"
  name     = "budget-app-${random_id.deployment.hex}"
  region   = var.region
  size     = var.droplet_size
  vpc_uuid = digitalocean_vpc.budget_vpc.id

  tags = ["budget", "app", "production"]

  user_data = templatefile("${path.module}/cloud-init.yml", {
    database_url = var.use_managed_db ? "postgres://${digitalocean_database_user.budget_user[0].name}:${digitalocean_database_user.budget_user[0].password}@${digitalocean_database_cluster.budget_db[0].private_host}:${digitalocean_database_cluster.budget_db[0].port}/${digitalocean_database_db.budget_database[0].name}?sslmode=require" : "postgres://budget_user:budget_password@postgres:5432/budget?sslmode=disable"
    app_port     = var.app_port
    auto_terminate_minutes = var.auto_terminate_minutes
    github_repo = var.github_repo
    github_branch = var.github_branch
    docker_image_url = var.docker_image_url
    github_token = var.github_token
  })
}

# No load balancer for cost optimization - direct access to droplet

# Create a domain record (optional)
resource "digitalocean_domain" "budget_domain" {
  count = var.domain_name != "" ? 1 : 0
  name  = var.domain_name
}

resource "digitalocean_record" "budget_a_record" {
  count  = var.domain_name != "" ? 1 : 0
  domain = digitalocean_domain.budget_domain[0].id
  type   = "A"
  name   = "@"
  value  = digitalocean_droplet.budget_app.ipv4_address
  ttl    = 3600
}

resource "digitalocean_record" "budget_www_record" {
  count  = var.domain_name != "" ? 1 : 0
  domain = digitalocean_domain.budget_domain[0].id
  type   = "CNAME"
  name   = "www"
  value  = "@"
  ttl    = 3600
}

# Create a firewall
resource "digitalocean_firewall" "budget_firewall" {
  name = "budget-app-firewall-${random_id.deployment.hex}"

  droplet_ids = [digitalocean_droplet.budget_app.id]

  # Application port - allow access to the budget app
  inbound_rule {
    protocol         = "tcp"
    port_range       = var.app_port
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  # HTTP port - for nginx proxy
  inbound_rule {
    protocol         = "tcp"
    port_range       = "80"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  # HTTPS port - for nginx proxy (future SSL)
  inbound_rule {
    protocol         = "tcp"
    port_range       = "443"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  # Status server port - for deployment diagnostics
  inbound_rule {
    protocol         = "tcp"
    port_range       = "9999"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  # Allow all outbound traffic
  outbound_rule {
    protocol              = "tcp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "udp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  # Note: No SSH port 22 since we don't use SSH for deployment
  # Note: Firewall resources don't use tags, they use droplet_ids
}