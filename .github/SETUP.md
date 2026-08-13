# GitHub Actions Setup

This repository deploys the Budget App to DigitalOcean App Platform using GitHub Actions, Terraform, and a reusable workflow pipeline.

## Repository secrets

Configure these under **Settings → Secrets and variables → Actions → Secrets**:

| Secret | Used by | Description |
|--------|---------|-------------|
| `DO_TOKEN` | PR + production | DigitalOcean API token with write access |
| `TERRAFORM_AWS_S3_ACCESS_KEY` | PR + production | AWS access key for Terraform state (S3) |
| `TERRAFORM_AWS_S3_ACCESS_KEY_SECRET` | PR + production | AWS secret key for Terraform state |
| `TERRAFORM_AWS_S3_REGION` | PR + production | AWS region for the state bucket (e.g. `us-east-1`) |
| `SLACK_WEBHOOK_PR` | PR deployments | Slack incoming webhook for PR deploy notifications |
| `SLACK_WEBHOOK_PRODUCTION` | Production | Slack incoming webhook for production notifications |

### Getting a DigitalOcean token

1. Open [DigitalOcean API tokens](https://cloud.digitalocean.com/account/api/tokens)
2. Create a token with **Write** scope
3. Save it as `DO_TOKEN`

## Repository variables

Configure under **Settings → Secrets and variables → Actions → Variables**:

| Variable | Used by | Description |
|----------|---------|-------------|
| `PRODUCTION_DOMAIN` | Production (optional) | Custom domain pre-allocated in DigitalOcean (DNS managed outside Terraform) |
| `TERMINATION_DELAY_MINUTES` | PR auto-termination (optional) | Override for minutes before PR deployments are destroyed. If unset, the workflow reads the `termination-delay` environment wait timer automatically (fallback: `30`). |

You can override `PRODUCTION_DOMAIN` per run via the production workflow's `domain_name` input. Override `TERMINATION_DELAY_MINUTES` per run via the PR deploy workflow's `termination_delay_minutes` input.

## GitHub environment (PR auto-termination)

PR deployments auto-terminate after the `termination-delay` environment wait timer. Create a repository environment:

1. **Settings → Environments → New environment**
2. Name: `termination-delay`
3. Add protection rule: **Wait timer** (e.g. 5 minutes for testing, 30 for production-like runs)

The deploy workflow reads this wait timer via the GitHub API for PR comments and auto-termination scheduling.

See [ENVIRONMENT-SETUP.md](ENVIRONMENT-SETUP.md) for details.

## DigitalOcean prerequisites

- Project `budget-develop` (PR deployments)
- Project `budget-prod` (production)
- S3 bucket `densestvoid-terraform` for Terraform state
- GHCR package access for `ghcr.io/<org>/budget/budget-app`
- Optional: custom domain added in DigitalOcean (referenced via `PRODUCTION_DOMAIN`)

## Workflows

| Workflow | File | Triggers |
|----------|------|----------|
| Deploy Budget App to DigitalOcean | `deploy.yml` | PR open/sync/reopen; **manual** (`workflow_dispatch`) |
| Deploy to Production | `deploy-production.yml` | Push to `main`; **manual** (`workflow_dispatch`) |
| Auto-Terminate Deployment | `auto-terminate.yml` | Triggered by PR deploy workflow; **manual** (`workflow_dispatch`) |
| Deploy Budget App (Reusable) | `deploy-reusable.yml` | Called by the workflows above (not run directly) |

### Automatic triggers

- **PR**: Opening or updating a pull request runs a PR deployment (`pr-<number>` resources in `budget-develop`).
- **Production**: Merging to `main` runs a production deployment (`budget-prod`).

### Manual triggers

Workflows with `workflow_dispatch` must be run from a branch that contains the workflow file. In the Actions UI, use **Run workflow** and select the branch (e.g. your PR head branch) before starting the run.

**Deploy a PR** (Actions → *Deploy Budget App to DigitalOcean* → Run workflow):

- `pr_number` — PR number to deploy (required)
- `ref` — optional branch or tag to build from (defaults to the workflow's selected branch)
- `termination_delay_minutes` — optional delay override (must match `termination-delay` environment wait timer and `TERMINATION_DELAY_MINUTES`)
- `force_cleanup` — set to **true** to destroy existing PR resources before deploying (use when a prior deployment was never terminated)

**Terminate a PR deployment immediately** (Actions → *Auto-Terminate Deployment* → Run workflow):

- Select the branch that contains the workflow (e.g. your PR branch)
- `pr_number`, `deployment_id` (e.g. `pr-9`), `app_id`, `app_url` — from the deployment PR comment
- `skip_environment_wait` — set to **true** for immediate cleanup (skips the wait timer)

**Deploy production** (Actions → *Deploy to Production* → Run workflow):

- `ref` — branch or tag to deploy (default: `main`)
- `domain_name` — optional custom domain (falls back to `PRODUCTION_DOMAIN`)

## What each deployment does

1. Detect whether Go and Docker builds are needed (cache + GHCR image check by content hash)
2. Run Go checks when source changed
3. Build and push Docker image to GHCR when needed
4. Run Terraform (`terraform/pr` or `terraform/production`)
5. Run database migrations via a DigitalOcean pre-deploy job
6. Deploy the application to App Platform
7. Post a PR comment and Slack notification
8. PR only: schedule auto-termination after `TERMINATION_DELAY_MINUTES` (default 30)

## Architecture

```
Internet → DigitalOcean App Platform (HTTPS)
              ├── Migration job (PRE_DEPLOY)
              └── Web service
              └── Managed PostgreSQL (private VPC)
```

PR deployments are ephemeral (default ~30 minutes; configurable via `TERMINATION_DELAY_MINUTES`). Production uses long-lived database resources with `prevent_destroy`.

## Cost notes

- PR: managed DB + App Platform, auto-terminated after `TERMINATION_DELAY_MINUTES` (default 30)
- Production: persistent DB (`db-s-1vcpu-1gb`) + App Platform (`basic-xxs`)

## Security

- Secrets are only available inside GitHub Actions
- Database runs on a private VPC endpoint
- No SSH or droplet access — fully managed App Platform
- PR deployments are destroyed automatically

## More documentation

- [ENVIRONMENT-SETUP.md](ENVIRONMENT-SETUP.md) — auto-termination environment
- [README-DEPLOYMENT.md](../README-DEPLOYMENT.md) — deployment overview
- [terraform/README.md](../terraform/README.md) — Terraform layout and variables
