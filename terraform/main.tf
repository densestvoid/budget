terraform {
  required_version = ">= 1.0"
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

# Configure the DigitalOcean Provider
provider "digitalocean" {
  token = var.do_token
}

# Create a VPC
resource "digitalocean_vpc" "budget_vpc" {
  name     = "budget-vpc"
  region   = var.region
  ip_range = "10.10.0.0/16"

  tags = ["budget", "production"]
}

# Create a managed PostgreSQL database (only if use_managed_db is true)
resource "digitalocean_database_cluster" "budget_db" {
  count      = var.use_managed_db ? 1 : 0
  name       = "budget-db"
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

# Create SSH key
resource "digitalocean_ssh_key" "budget_key" {
  name       = "budget-app-key"
  public_key = file(var.ssh_public_key_path)
}

# Create a Droplet for the application
resource "digitalocean_droplet" "budget_app" {
  image    = "ubuntu-22-04-x64"
  name     = "budget-app"
  region   = var.region
  size     = var.droplet_size
  ssh_keys = [digitalocean_ssh_key.budget_key.id]
  vpc_uuid = digitalocean_vpc.budget_vpc.id

  tags = ["budget", "app", "production"]

  user_data = templatefile("${path.module}/cloud-init.yml", {
    database_url = var.use_managed_db ? "postgres://${digitalocean_database_user.budget_user[0].name}:${digitalocean_database_user.budget_user[0].password}@${digitalocean_database_cluster.budget_db[0].private_host}:${digitalocean_database_cluster.budget_db[0].port}/${digitalocean_database_db.budget_database[0].name}?sslmode=require" : "sqlite:///app/data/budget.db"
    app_port     = var.app_port
    auto_terminate_minutes = var.auto_terminate_minutes
    github_repo = var.github_repo
    github_branch = var.github_branch
  })

  connection {
    type        = "ssh"
    user        = "root"
    private_key = file(var.ssh_private_key_path)
    host        = self.ipv4_address
  }

  # Wait for the droplet to be ready
  provisioner "remote-exec" {
    inline = [
      "while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 2; done"
    ]
  }
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
  name = "budget-app-firewall"

  droplet_ids = [digitalocean_droplet.budget_app.id]

  inbound_rule {
    protocol         = "tcp"
    port_range       = "22"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = var.app_port
    source_addresses = ["0.0.0.0/0", "::/0"] # Allow direct access since no load balancer
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "80"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "443"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

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

  tags = ["budget", "firewall", "production"]
}