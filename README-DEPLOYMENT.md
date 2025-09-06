# Quick Deployment Guide

This branch contains a complete Terraform configuration for deploying the Budget App to DigitalOcean.

## Quick Start

1. **Setup SSH keys:**
   ```bash
   ./scripts/setup-ssh.sh
   ```

2. **Configure deployment:**
   ```bash
   cp terraform/terraform.tfvars.example terraform/terraform.tfvars
   # Edit terraform/terraform.tfvars with your DigitalOcean API token
   ```

3. **Deploy everything:**
   ```bash
   ./scripts/deploy.sh
   ```

That's it! Your Budget App will be deployed to DigitalOcean with:
- Application server (Ubuntu droplet)
- Managed PostgreSQL database
- Load balancer with health checks
- VPC for secure networking
- Firewall rules
- Optional domain configuration

See [DEPLOYMENT.md](DEPLOYMENT.md) for detailed documentation.

## Cost Estimation
- Basic setup: ~$33/month (droplet + database + load balancer)
- Standard setup: ~$54/month (larger instances)

## Architecture
```
Internet → Load Balancer → Application Droplet → Managed Database
                             ↓
                        VPC (Private Network)
```