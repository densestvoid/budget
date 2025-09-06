# 💰 Ultra-Cheap DigitalOcean Deployment

This branch contains a **cost-optimized** Terraform configuration for deploying the Budget App to DigitalOcean for just **$4/month**.

## 🚀 GitHub Actions Deployment (Recommended)

1. **Add repository secrets:**
   - `DO_TOKEN`: Your DigitalOcean API token
   - `SSH_PRIVATE_KEY`: Your SSH private key
   - `SSH_PUBLIC_KEY`: Your SSH public key

2. **Push to any branch** - deployment happens automatically!

3. **Auto-termination after 30 minutes** saves costs

## 🛠️ Manual Deployment

1. **Setup SSH keys:**
   ```bash
   ./scripts/setup-ssh.sh
   ```

2. **Configure deployment:**
   ```bash
   cp terraform/terraform.tfvars.example terraform/terraform.tfvars
   # Edit with your DigitalOcean API token
   ```

3. **Deploy everything:**
   ```bash
   ./scripts/deploy.sh
   ```

## 💰 Ultra-Cost-Optimized Features

- **$4/month total cost** (cheapest possible DigitalOcean setup)
- **SQLite database** (FREE - no managed database costs)
- **No load balancer** (saves $12/month)
- **512MB RAM droplet** (smallest available)
- **Auto-termination** after 30 minutes
- **GitHub Actions integration**

## Architecture
```
Internet → Application Droplet (512MB) → SQLite Database (local)
              ↓
         VPC (Private Network)
```

## Perfect For
- 🧪 **Testing & Demos**
- 🔄 **CI/CD Pipelines** 
- 📚 **Learning & Development**
- 💡 **Proof of Concepts**

See [DEPLOYMENT.md](DEPLOYMENT.md) for detailed documentation.