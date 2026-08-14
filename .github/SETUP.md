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
| `SLACK_WEBHOOK_PR` | PR deploy + terminate | Slack incoming webhook for PR notifications |
| `SLACK_WEBHOOK_PROD` | Production | Slack incoming webhook for production notifications |

### Getting a DigitalOcean token

1. Open [DigitalOcean API tokens](https://cloud.digitalocean.com/account/api/tokens)
2. Create a token with **Write** scope
3. Save it as `DO_TOKEN`

## Repository variables

Configure under **Settings → Secrets and variables → Actions → Variables**:

| Variable | Used by | Description |
|----------|---------|-------------|
| `PRODUCTION_DOMAIN` | PR + production | DO DNS zone (e.g. `budget.example.com`). Passed to Terraform as `domain_name`. Zone must exist in **Networking → Domains**. PR URLs use `{deployment_id}.{domain_name}`. |
| `TERMINATION_DELAY_MINUTES` | PR deploy (optional) | Overrides `termination-delay` environment wait timer when computing scheduled termination UTC in notifications |

You can override per run via the production workflow's `domain_name` input (falls back to `PRODUCTION_DOMAIN`).

## Branch protection

Require **CI / Run Go Checks** only. Do not require deploy workflows as status checks — deploy runs asynchronously after CI passes.

## GitHub environment (PR auto-termination)

PR deployments auto-terminate after the `termination-delay` environment wait timer. Create a repository environment:

1. **Settings → Environments → New environment**
2. Name: `termination-delay`
3. Add protection rule: **Wait timer** (e.g. 5 minutes for testing, 30 for production-like runs)

The terminate workflow uses this environment when scheduling auto-termination after a successful PR deploy.

See [ENVIRONMENT-SETUP.md](ENVIRONMENT-SETUP.md) for details.

## DigitalOcean prerequisites

- Project `budget-develop` (PR deployments)
- Project `budget-prod` (production)
- S3 bucket `densestvoid-terraform` for Terraform state
- GHCR package access for `ghcr.io/<org>/budget/budget-app`
- DNS zone for `PRODUCTION_DOMAIN` in **Networking → Domains** (create manually before first deploy with custom domain)
- Optional: `PRODUCTION_DOMAIN` repository variable set to that zone name

## Workflows

| Workflow | File | Triggers |
|----------|------|----------|
| CI | `ci.yml` | PR open/sync/reopen; push to `main` |
| Deploy PR | `deploy-pr.yml` | After CI success on PR branch (non-`main`); **manual** |
| Deploy Prod | `deploy-production.yml` | After CI success on push to `main`; **manual** |
| Terminate PR Deployment | `terminate-pr-deployment.yml` | After PR deploy completes; **manual** |
| Notify Deployment | `notify-deployment.yml` | After deploy or terminate completes (`workflow_run`) |
| Deploy Budget App (Reusable) | `deploy-reusable.yml` | Called by deploy workflows (not run directly) |

### Automatic triggers

- **PR**: Push to a PR branch runs CI. When CI passes, PR deploy runs automatically **only if the branch has deployable changes relative to `main`** (Go sources, `go.mod`/`go.sum`, embedded migrations, Docker files, or Terraform). Docs-, workflow-, and config-only PRs skip deploy and post a skip notification (PR comment + Slack). Manual `workflow_dispatch` always deploys.
- **Production**: Push to `main` runs CI. When CI passes, production deploy runs automatically.
- **PR teardown**: When a PR deploy completes (success or failure), terminate runs — scheduled wait after success, immediate cleanup after failure. Skipped deploys do not trigger termination.
- **Notifications**: Notify runs after every deploy and terminate completion (PR comment + Slack or Slack only).

**Deployable paths** (branch diff vs `main`): `**/*.go`, `go.mod`, `go.sum`, `data/migrations/**`, `Dockerfile*`, `.dockerignore`, `terraform/**`.

**Excluded** (no PR environment): `.github/**`, `**/*.md`, `.cursor/**`, local dev files (`Taskfile.yml`, `docker-compose.yml`, `.air.toml`), `config.yaml`, `env.example`, `.golangci.yml`, `scripts/**`, `.do/**`, `LICENSE`.

`workflow_run` listener workflows must exist on the default branch to fire.

### Manual triggers

Workflows with `workflow_dispatch` must be run from a branch that contains the workflow file.

**Deploy a PR** (Actions → *Deploy PR* → Run workflow):

- `pr_number` — PR number to deploy (required)
- `ref` — optional git ref to build from
- `force_cleanup` — destroy existing PR resources before deploying

Manual PR deploy does **not** run CI — use only for redeploy/debug.

**Terminate or cleanup a PR deployment** (Actions → *Terminate PR Deployment* → Run workflow):

- `pr_number` — PR number (required)
- `skip_environment_wait` — default **true** for immediate cleanup

Terminate loads `pr/{id}.tfstate` from S3 and tears down whatever is recorded there via `terraform/pr-destroy` (providers only, no resource definitions). Terraform requires `apply` here, not `destroy`: `destroy` only removes resources still present in config, but pr-destroy has none — state entries are orphans removed by `apply`.

**Deploy production** (Actions → *Deploy Prod* → Run workflow):

- `ref` — branch or tag to deploy (default: `main`)
- `domain_name` — optional workflow input; falls back to `PRODUCTION_DOMAIN`

Manual production deploy does **not** run CI.

## What each deployment does

1. **CI** (`ci.yml`): Go checks (vet, lint, static analysis, security, vulnerabilities)
2. **Deploy** (`deploy-reusable.yml`): detect build requirements, build/push Docker image when needed, Terraform apply, health check, publish `deploy-result` artifact
3. **Notify** (`notify-deployment.yml`): PR comment and/or Slack on deploy success or failure
4. **Terminate** (`terminate-pr-deployment.yml`): destroy PR resources after success (with wait timer) or failure (immediate), plus manual cleanup
5. **Notify** (again): PR comment and Slack when terminate/cleanup completes

## Architecture

```
Internet → DigitalOcean App Platform (HTTPS)
              ├── Migration job (PRE_DEPLOY)
              └── Web service
              └── Managed PostgreSQL (private VPC)
```

PR deployments are ephemeral (duration set by the `termination-delay` environment wait timer). Production uses long-lived database resources with `prevent_destroy`.

## Cost notes

- PR: managed DB + App Platform, auto-terminated after the `termination-delay` environment wait timer
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
