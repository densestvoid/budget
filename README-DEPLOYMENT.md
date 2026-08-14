# Budget App Deployment

Automated deployment to DigitalOcean App Platform via GitHub Actions.

## Quick start

1. Configure [GitHub Actions secrets and variables](.github/SETUP.md)
2. Create the `termination-delay` environment for PR auto-termination
3. Set branch protection to require **CI / Run Go Checks** only
4. Open a PR — CI runs, then deploy runs automatically and comments with the app URL
5. Merge to `main` — CI runs, then production deploys automatically

## Manual deployment

### PR environment

Actions → **Deploy PR** → Run workflow

| Input | Required | Description |
|-------|----------|-------------|
| `pr_number` | Yes | Pull request number |
| `ref` | No | Git ref to build from |
| `force_cleanup` | No | Destroy existing PR resources before deploy |

Manual deploy skips CI.

### Production

Actions → **Deploy Prod** → Run workflow

| Input | Required | Description |
|-------|----------|-------------|
| `ref` | No | Branch or tag (default: `main`) |
| `domain_name` | No | Workflow input; defaults to `PRODUCTION_DOMAIN` repository variable |

Manual production deploy skips CI.

### Terminate PR deployment

Actions → **Terminate PR Deployment** → Run workflow

| Input | Required | Description |
|-------|----------|-------------|
| `pr_number` | Yes | Pull request number |
| `skip_environment_wait` | No | Immediate cleanup (default: true) |

## Environments

| Environment | Terraform dir | DO project | Auto-terminate | Domain |
|-------------|---------------|------------|----------------|--------|
| PR | `terraform/pr` | `budget-develop` | Yes (`termination-delay` env) | `{deployment_id}.{domain_name}` when `PRODUCTION_DOMAIN` set |
| Production | `terraform/production` | `budget-prod` | No | Optional (`PRODUCTION_DOMAIN` or workflow `domain_name` input) |

PR deploy runs only when the PR head branch is not `main`. Production deploy runs on push to `main`.

## Custom domain

Set repository variable `PRODUCTION_DOMAIN` (DO DNS zone). Workflows pass it to Terraform as `domain_name`. PR URLs use `{deployment_id}.{domain_name}`.

Override production hostname per run via the Deploy Prod workflow `domain_name` input.

The DNS zone must exist in **DO Networking → Domains** before deploy. Terraform attaches the hostname and App Platform creates records in that zone.

## Pipeline overview

```
PR/main change → CI (go-checks)
              → deploy-pr.yml or deploy-production.yml
                   → deploy-reusable.yml (build, Terraform, deploy-result artifact)
                   → notify job (PR comment + Slack, or Slack only for production)
              → terminate-pr-deployment.yml (PR only, workflow_run after deploy)
                   → notify job (termination Slack + PR comment)
```

`deploy-reusable.yml` handles build caching, GHCR push, and Terraform apply. Notify and terminate are sibling or follow-on jobs in the same workflow runs (terminate still uses `workflow_run` to start after deploy).

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
| PR | ~$0.01 per run | Auto-terminates after `termination-delay` wait timer |
| Production | ~$25+/month | Persistent DB + app |

## Setup reference

Full secret, variable, and environment configuration: [.github/SETUP.md](.github/SETUP.md)
