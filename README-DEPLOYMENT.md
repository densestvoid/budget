# Budget App Deployment

Automated deployment to DigitalOcean App Platform via GitHub Actions.

## Quick start

1. Configure [GitHub Actions secrets and variables](.github/SETUP.md)
2. Create the `termination-delay` environment for PR auto-termination
3. Open a PR — deployment runs automatically and comments with the app URL
4. Merge to `main` — production deploys automatically

## Manual deployment

### PR environment

Actions → **Deploy Budget App to DigitalOcean** → Run workflow

| Input | Required | Description |
|-------|----------|-------------|
| `pr_number` | Yes | Pull request number |
| `ref` | No | Git ref to build from |

### Production

Actions → **Deploy to Production** → Run workflow

| Input | Required | Description |
|-------|----------|-------------|
| `ref` | No | Branch or tag (default: `main`) |
| `domain_name` | No | Custom domain (default: `PRODUCTION_DOMAIN` variable) |

## Environments

| Environment | Terraform dir | DO project | Auto-terminate | Domain |
|-------------|---------------|------------|----------------|--------|
| PR | `terraform/pr` | `budget-develop` | Yes (30 min) | App Platform default URL |
| Production | `terraform/production` | `budget-prod` | No | Optional (`PRODUCTION_DOMAIN` or workflow input) |

## Custom domain (production)

Production does not manage DNS in Terraform. To use a custom domain:

1. Add the domain in DigitalOcean
2. Configure DNS records outside Terraform (pointing to App Platform)
3. Set repository variable `PRODUCTION_DOMAIN` or pass `domain_name` when running the production workflow manually

Terraform only references the domain for outputs when configured.

## Pipeline overview

All deploy workflows call `.github/workflows/deploy-reusable.yml`, which handles:

- Build caching (Go binary, Docker buildx)
- External Go checks (`densestvoid/workflows`)
- GHCR image push
- Terraform apply against AWS S3 backend (`densestvoid-terraform` bucket)
- Health check against `/health`
- Slack notifications
- PR comments with deployment URL and termination time

## Local Terraform

```bash
# PR deployment
cd terraform/pr
cp terraform.tfvars.example terraform.tfvars
terraform init
terraform plan

# Production
cd terraform/production
cp terraform.tfvars.example terraform.tfvars
terraform init
terraform plan
```

See [terraform/README.md](terraform/README.md) for variable details.

## Cost

| Environment | Typical cost | Notes |
|-------------|--------------|-------|
| PR | ~$0.01 per run | Auto-terminates after 30 minutes |
| Production | ~$25+/month | Persistent DB + app |

## Setup reference

Full secret, variable, and environment configuration: [.github/SETUP.md](.github/SETUP.md)
