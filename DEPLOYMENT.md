# Budget App - Cost-Optimized DigitalOcean Deployment

This guide walks you through deploying the Budget App to DigitalOcean using Terraform with **maximum cost optimization**.

## 💰 Cost-Optimized Architecture

The deployment creates the **cheapest possible** setup:
- **VPC**: Private network for secure communication (FREE)
- **Droplet**: 512MB RAM Ubuntu 22.04 server ($4/month)
- **PostgreSQL Database**: Containerized PostgreSQL (FREE - runs on same droplet)
- **Direct Access**: No load balancer to save costs (saves $12/month)
- **Auto-Termination**: Automatically shuts down after 30 minutes
- **Firewall**: Security rules for the application (FREE)

**Total Cost: ~$4/month** (or $0.13/day, or $0.0055/hour)

## Prerequisites

1. **DigitalOcean Account**: Create an account at [digitalocean.com](https://digitalocean.com)
2. **DigitalOcean API Token**: Generate a personal access token from the DigitalOcean control panel
3. **Terraform**: Install from [terraform.io](https://terraform.io)
4. **Docker**: Install from [docker.com](https://docker.com)
5. **SSH Keys**: For server access

## 🚀 GitHub Actions Deployment (Recommended)

The easiest way to deploy is using GitHub Actions, which automatically deploys on every push:

### 1. Setup Repository Secrets

Add these secrets to your GitHub repository (Settings → Secrets and variables → Actions):

- `DO_TOKEN`: Your DigitalOcean API token
- `SSH_PRIVATE_KEY`: Your SSH private key content
- `SSH_PUBLIC_KEY`: Your SSH public key content

### 2. Push to Repository

Simply push to any branch and GitHub Actions will automatically:
1. Build the application
2. Deploy infrastructure 
3. Deploy the application
4. Set up auto-termination after 30 minutes

## 🛠️ Manual Deployment

### 1. Setup SSH Keys

```bash
./scripts/setup-ssh.sh
```

### 2. Configure Terraform Variables

```bash
cp terraform/terraform.tfvars.example terraform/terraform.tfvars
```

Edit with your **cost-optimized** configuration:

```hcl
# Required: Your DigitalOcean API token
do_token = "your-digitalocean-api-token-here"

# Cost-optimized settings
region       = "nyc1"
droplet_size = "s-1vcpu-512mb-10gb"    # Cheapest at $4/month
use_managed_db = false                  # Use containerized PostgreSQL to save $15/month

# Auto-termination
auto_terminate_minutes = 30             # Auto-shutdown after 30 min
```

### 3. Deploy Everything

```bash
./scripts/deploy.sh
```

### 4. Access Your Application

After deployment completes, you'll see:
```
Your Budget App is now running at: http://147.182.123.45:8080
⚠️ Note: This deployment will auto-terminate after 30 minutes to save costs
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

## 💰 Cost Optimization Features

### Database Options
- **Containerized PostgreSQL (default)**: FREE - PostgreSQL running in Docker container, only accessible from application
- **Managed PostgreSQL**: $15/month - Only enable if you need production-grade features with backups and high availability

### Droplet Sizes (Cost-Optimized)
- `s-1vcpu-512mb-10gb`: **Ultra-cheap** (512MB RAM) - $4/month ⭐ **RECOMMENDED**
- `s-1vcpu-1gb`: Basic (1GB RAM) - $6/month
- `s-1vcpu-2gb`: Standard (2GB RAM) - $12/month

### Auto-Termination
- **Default**: 30 minutes - Saves costs for testing/demos
- **Configurable**: Set `auto_terminate_minutes` to any value
- **Disable**: Set to `0` to disable auto-termination

### Regions (Choose closest to your users)
`nyc1`, `nyc3`, `ams2`, `ams3`, `sfo1`, `sfo2`, `sfo3`, `sgp1`, `lon1`, `fra1`, `tor1`, `blr1`, `syd1`

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

## 💰 Cost Estimation

### Ultra-Cost-Optimized (Default)
- **Droplet**: $4/month (512MB RAM)
- **Database**: FREE (containerized PostgreSQL)
- **Load Balancer**: REMOVED (saves $12/month)
- **Total**: **$4/month** or **$0.13/day**

### With Managed Database (Optional)
- **Droplet**: $4/month
- **Database**: $15/month (managed PostgreSQL)
- **Total**: $19/month

### Auto-Termination Savings
- **30-minute deployment**: ~$0.0055 per deployment
- **Perfect for**: Testing, demos, CI/CD pipelines

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