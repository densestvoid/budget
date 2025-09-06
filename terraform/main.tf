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

# Create a managed PostgreSQL database
resource "digitalocean_database_cluster" "budget_db" {
  name       = "budget-db"
  engine     = "pg"
  version    = "16"
  size       = var.db_size
  region     = var.region
  node_count = 1

  private_network_uuid = digitalocean_vpc.budget_vpc.id

  tags = ["budget", "database", "production"]
}

# Create a database within the cluster
resource "digitalocean_database_db" "budget_database" {
  cluster_id = digitalocean_database_cluster.budget_db.id
  name       = "budget"
}

# Create a database user
resource "digitalocean_database_user" "budget_user" {
  cluster_id = digitalocean_database_cluster.budget_db.id
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
    database_url = "postgres://${digitalocean_database_user.budget_user.name}:${digitalocean_database_user.budget_user.password}@${digitalocean_database_cluster.budget_db.private_host}:${digitalocean_database_cluster.budget_db.port}/${digitalocean_database_db.budget_database.name}?sslmode=require"
    app_port     = var.app_port
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

# Create a Load Balancer
resource "digitalocean_loadbalancer" "budget_lb" {
  name   = "budget-lb"
  region = var.region

  forwarding_rule {
    entry_protocol  = "http"
    entry_port      = 80
    target_protocol = "http"
    target_port     = var.app_port
  }

  forwarding_rule {
    entry_protocol  = "https"
    entry_port      = 443
    target_protocol = "http"
    target_port     = var.app_port
    tls_passthrough = false
  }

  healthcheck {
    protocol = "http"
    port     = var.app_port
    path     = "/health"
  }

  droplet_ids = [digitalocean_droplet.budget_app.id]
  vpc_uuid    = digitalocean_vpc.budget_vpc.id

  tags = ["budget", "loadbalancer", "production"]
}

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
  value  = digitalocean_loadbalancer.budget_lb.ip
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
    source_addresses = ["10.10.0.0/16"] # Only from VPC
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