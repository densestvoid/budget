# GitHub Actions Setup Guide

To enable automated deployment with GitHub Actions, you need to configure repository secrets.

## Required Secrets

Go to your GitHub repository → Settings → Secrets and variables → Actions → New repository secret

### 1. `DO_TOKEN`
- **Description**: DigitalOcean API token
- **How to get**: 
  1. Go to [DigitalOcean Control Panel](https://cloud.digitalocean.com/account/api/tokens)
  2. Click "Generate New Token"
  3. Give it a name like "GitHub Actions Budget App"
  4. Select "Write" scope
  5. Copy the generated token

### 2. `SSH_PRIVATE_KEY`
- **Description**: SSH private key content
- **How to get**:
  ```bash
  # Generate new SSH key pair (if you don't have one)
  ssh-keygen -t rsa -b 4096 -f ~/.ssh/budget_app_key -N ""
  
  # Copy private key content
  cat ~/.ssh/budget_app_key
  ```
  Copy the entire content including `-----BEGIN OPENSSH PRIVATE KEY-----` and `-----END OPENSSH PRIVATE KEY-----`

### 3. `SSH_PUBLIC_KEY`
- **Description**: SSH public key content
- **How to get**:
  ```bash
  # Copy public key content
  cat ~/.ssh/budget_app_key.pub
  ```
  Copy the entire line (it should start with `ssh-rsa`)

## Deployment Triggers

The GitHub Action will automatically deploy when you:
- Push to `main`, `develop`, or any `feature/*`, `hotfix/*` branch
- The deployment will auto-terminate after 30 minutes to save costs

## Deployment Information

After deployment, check the GitHub Actions tab to see:
- Application URL
- Deployment status
- Cost information
- Auto-termination timer

## Security Notes

- Secrets are encrypted and only accessible to GitHub Actions
- SSH keys are generated specifically for deployment
- DigitalOcean token should have minimal required permissions
- Auto-termination ensures resources don't run indefinitely

## Cost Control

- Each deployment costs ~$0.0055 (30 minutes at $4/month rate)
- Perfect for testing, demos, and CI/CD pipelines
- No long-running costs unless you disable auto-termination