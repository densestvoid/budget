# Budget App - DigitalOcean Deployment Guide

This guide walks you through deploying the Budget App to DigitalOcean using Terraform.

## Architecture Overview

The deployment creates:
- **VPC**: Private network for secure communication
- **Droplet**: Ubuntu 22.04 server running the application
- **Managed PostgreSQL Database**: Highly available database cluster
- **Load Balancer**: For high availability and SSL termination
- **Firewall**: Security rules for the application
- **Domain & DNS** (optional): Custom domain configuration

## Prerequisites

1. **DigitalOcean Account**: Create an account at [digitalocean.com](https://digitalocean.com)
2. **DigitalOcean API Token**: Generate a personal access token from the DigitalOcean control panel
3. **Terraform**: Install from [terraform.io](https://terraform.io)
4. **Docker**: Install from [docker.com](https://docker.com)
5. **SSH Keys**: For server access

## Quick Start

### 1. Setup SSH Keys

```bash
./scripts/setup-ssh.sh
```

This will generate SSH keys if they don't exist and display your public key to add to DigitalOcean.

### 2. Configure Terraform Variables

```bash
cp terraform/terraform.tfvars.example terraform/terraform.tfvars
```

Edit `terraform/terraform.tfvars` with your configuration:

```hcl
# Required: Your DigitalOcean API token
do_token = "your-digitalocean-api-token-here"

# Optional: Customize these values
region       = "nyc1"
droplet_size = "s-1vcpu-1gb"
db_size      = "db-s-1vcpu-1gb"

# SSH configuration (update paths if needed)
ssh_public_key_path  = "~/.ssh/id_rsa.pub"
ssh_private_key_path = "~/.ssh/id_rsa"

# Application configuration
app_port = "8080"

# Optional: Custom domain
# domain_name = "your-domain.com"
```

### 3. Deploy Everything

```bash
./scripts/deploy.sh
```

This single command will:
1. Build the Go application
2. Create a Docker image
3. Deploy infrastructure with Terraform
4. Deploy the application to the server
5. Run database migrations

### 4. Access Your Application

After deployment completes, you'll see output like:
```
Your Budget App is now running at: http://147.182.123.45
```

## Manual Deployment Steps

If you prefer to run steps individually:

### Check Requirements
```bash
./scripts/deploy.sh check
```

### Build Application
```bash
./scripts/deploy.sh build
```

### Deploy Infrastructure Only
```bash
./scripts/deploy.sh infrastructure
```

### Deploy Application Only
```bash
./scripts/deploy.sh deploy
```

### Get Deployment Info
```bash
./scripts/deploy.sh info
```

### Destroy Infrastructure
```bash
./scripts/deploy.sh destroy
```

## Configuration Options

### Droplet Sizes
- `s-1vcpu-1gb`: Basic (1 vCPU, 1GB RAM) - $6/month
- `s-1vcpu-2gb`: Standard (1 vCPU, 2GB RAM) - $12/month
- `s-2vcpu-2gb`: Enhanced (2 vCPUs, 2GB RAM) - $18/month
- `s-2vcpu-4gb`: Performance (2 vCPUs, 4GB RAM) - $24/month

### Database Sizes
- `db-s-1vcpu-1gb`: Basic (1 vCPU, 1GB RAM) - $15/month
- `db-s-1vcpu-2gb`: Standard (1 vCPU, 2GB RAM) - $30/month
- `db-s-2vcpu-4gb`: Enhanced (2 vCPUs, 4GB RAM) - $60/month

### Regions
Available regions: `nyc1`, `nyc3`, `ams2`, `ams3`, `sfo1`, `sfo2`, `sfo3`, `sgp1`, `lon1`, `fra1`, `tor1`, `blr1`, `syd1`

## SSL/HTTPS Setup

### Option 1: Let's Encrypt (Recommended)
If you have a domain name, the load balancer can automatically provision SSL certificates.

### Option 2: Custom Domain
1. Set `domain_name` in `terraform.tfvars`
2. Point your domain's nameservers to DigitalOcean
3. The deployment will create DNS records automatically

## Monitoring and Maintenance

### Application Logs
```bash
# SSH into the server
ssh root@<server-ip>

# View application logs
docker logs budget-app

# View nginx logs
docker logs budget-nginx
```

### Database Access
```bash
# Get database connection details
cd terraform
terraform output database_connection_string
```

### Health Checks
The application includes health check endpoints:
- `http://your-domain.com/health` - Application health
- Load balancer automatically monitors health

### Scaling

#### Vertical Scaling (Resize Resources)
1. Update `droplet_size` or `db_size` in `terraform.tfvars`
2. Run `terraform apply`

#### Horizontal Scaling (Multiple Droplets)
The current setup uses a single droplet. For multiple droplets:
1. Modify the Terraform configuration to use multiple droplets
2. Update the load balancer configuration

## Troubleshooting

### Common Issues

1. **SSH Connection Failed**
   - Ensure your SSH key is added to DigitalOcean
   - Check that the public key path is correct in `terraform.tfvars`

2. **Database Connection Failed**
   - Verify the VPC configuration allows database access
   - Check that the database is fully provisioned (can take 5-10 minutes)

3. **Application Not Starting**
   - Check application logs: `docker logs budget-app`
   - Verify environment variables are set correctly
   - Ensure database migrations ran successfully

4. **Load Balancer Health Check Failed**
   - Verify the application responds to `/health` endpoint
   - Check that the application is listening on the correct port

### Getting Help

1. Check application logs on the server
2. Review Terraform output for any errors
3. Verify all prerequisites are installed
4. Ensure DigitalOcean API token has proper permissions

## Security Considerations

- Database is only accessible from within the VPC
- Firewall rules restrict access to necessary ports only
- Application runs as non-root user
- SSL/TLS encryption for data in transit
- Regular security updates via cloud-init

## Cost Estimation

Monthly costs (USD):
- Basic setup: ~$21/month (1GB droplet + 1GB database)
- Standard setup: ~$42/month (2GB droplet + 2GB database)
- Load balancer: $12/month
- Bandwidth: Usually included in droplet cost

## Backup and Recovery

- Database backups are automatic with DigitalOcean managed databases
- Point-in-time recovery available
- Consider setting up additional application-level backups for critical data

## Next Steps

1. Set up monitoring (Prometheus/Grafana)
2. Configure log aggregation
3. Set up CI/CD pipeline
4. Implement blue-green deployments
5. Add application performance monitoring