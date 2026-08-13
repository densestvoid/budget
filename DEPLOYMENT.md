# Budget App - DigitalOcean Deployment

This guide describes the current App Platform deployment architecture. For GitHub Actions setup, see [.github/SETUP.md](.github/SETUP.md).

## Architecture

The deployment uses **DigitalOcean App Platform** with **managed PostgreSQL**:

- **VPC** — private networking between app and database (free)
- **Managed PostgreSQL** — `db-s-1vcpu-1gb` cluster on a private endpoint
- **App Platform** — migration pre-deploy job + web service (`basic-xxs`)
- **Automatic HTTPS** — provided by App Platform
- **GHCR** — container images built in CI and pushed to GitHub Container Registry
- **Terraform state** — stored in AWS S3 (`densestvoid-terraform` bucket)

PR deployments auto-terminate after the `termination-delay` environment wait timer. Production is persistent.

## GitHub Actions (recommended)

### Setup

1. Add repository secrets (see [.github/SETUP.md](.github/SETUP.md))
2. Set `PRODUCTION_DOMAIN` variable if using a custom production domain
3. Create the `termination-delay` GitHub environment
4. Require **CI / Run Go Checks** in branch protection (not deploy workflows)

### Automatic deployment

| Event | Workflows | Result |
|-------|-----------|--------|
| PR updated (non-`main` head) | `ci.yml` → `deploy.yml` | Ephemeral `pr-<number>` environment |
| Push to `main` | `ci.yml` → `deploy-production.yml` | Production deployment |

Notifications and PR teardown run via `workflow_run` listeners (`notify-deployment.yml`, `terminate-pr-deployment.yml`).

### Manual deployment

Both workflows support **Run workflow** from the Actions tab. See [README-DEPLOYMENT.md](README-DEPLOYMENT.md).

## Manual Terraform deployment

### PR

```bash
cd terraform/pr
cp terraform.tfvars.example terraform.tfvars
# Edit: do_token, deployment_id, docker_image_tag
terraform init -backend-config="key=pr/pr-123.tfstate"
terraform apply
```

### Production

```bash
cd terraform/production
cp terraform.tfvars.example terraform.tfvars
# Edit: do_token, docker_image_tag, domain_name (optional)
terraform init
terraform apply
```

## Database migrations

Migrations run automatically on every deployment via a DigitalOcean `PRE_DEPLOY` job before the web service starts. The migration container runs `./budget migrate`.

## Health checks

The deploy workflow checks `GET /health` after deployment. A `200` or `204` response is treated as healthy.

## Security

- Database accessible only via VPC private networking
- No SSH access — fully managed platform
- PR resources destroyed after 30 minutes
- Production database has `prevent_destroy` in Terraform

## Troubleshooting

| Issue | Check |
|-------|-------|
| Terraform init fails | `TERRAFORM_AWS_S3_*` secrets and bucket access |
| Docker image not found | GHCR auth, `packages: write` permission |
| Migration fails | DigitalOcean App Platform migration job logs |
| Auto-termination not running | `termination-delay` environment + wait timer |
| Wrong PR comment status | Recent workflow run logs for deploy job outputs |

## Related docs

- [.github/SETUP.md](.github/SETUP.md) — secrets, variables, workflows
- [README-DEPLOYMENT.md](README-DEPLOYMENT.md) — deployment overview
- [terraform/README.md](terraform/README.md) — Terraform structure
