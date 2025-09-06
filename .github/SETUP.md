# GitHub Actions Setup Guide

To enable automated deployment with GitHub Actions, you only need to configure one repository secret!

## Required Secret

Go to your GitHub repository → Settings → Secrets and variables → Actions → New repository secret

### `DO_TOKEN` (Only secret needed!)
- **Description**: DigitalOcean API token
- **How to get**: 
  1. Go to [DigitalOcean Control Panel](https://cloud.digitalocean.com/account/api/tokens)
  2. Click "Generate New Token"
  3. Give it a name like "GitHub Actions Budget App"
  4. Select "Write" scope
  5. Copy the generated token

## No SSH Keys Required! 🎉

This deployment uses **cloud-init** instead of SSH, which means:
- ✅ **More secure** - No SSH keys to manage
- ✅ **Simpler setup** - Only one secret needed
- ✅ **Better isolation** - No external SSH access to servers
- ✅ **Automatic deployment** - Everything happens via cloud-init script

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
- No SSH access to servers - more secure than traditional deployments
- DigitalOcean token should have minimal required permissions
- Auto-termination ensures resources don't run indefinitely
- Servers have no external SSH access - only accessible via application port

## Cost Control

- Each deployment costs ~$0.0055 (30 minutes at $4/month rate)
- Perfect for testing, demos, and CI/CD pipelines
- No long-running costs unless you disable auto-termination