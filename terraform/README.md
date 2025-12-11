# Budget App Terraform Configuration

This directory contains Terraform configurations for deploying the Budget App to DigitalOcean. The configuration follows Terraform best practices by separating PR and production deployments into distinct directories.

## Structure

```
terraform/
├── modules/
│   └── budget-app/          # Reusable module for app deployment
│       ├── main.tf          # Core app resources (VPC, migrations, app)
│       ├── variables.tf     # Module input variables
│       └── outputs.tf      # Module outputs
├── pr/                      # PR deployment configuration
│   ├── main.tf              # PR-specific resources (creates new DB)
│   ├── variables.tf         # PR input variables
│   ├── outputs.tf           # PR outputs
│   └── terraform.tfvars.example
└── production/              # Production deployment configuration
    ├── main.tf              # Production-specific resources (uses existing DB)
    ├── variables.tf         # Production input variables
    ├── outputs.tf           # Production outputs
    └── terraform.tfvars.example
```

## Module: budget-app

The reusable module that handles:
- VPC creation for private networking
- Database health checks
- Migration app deployment (schema migrations **always execute**)
- Main application deployment
- Project resource assignment

## PR Deployments (`terraform/pr/`)

Creates a complete new deployment for each PR:
- **Creates new database** cluster, database, and user
- **Creates database schema** and grants permissions
- **Runs schema migrations** (always executed via migration app)
- Uses `budget-develop` project
- No domain configuration (optional)
- Backend state: `pr/{deployment_id}.tfstate`

### Usage

```bash
cd terraform/pr
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values
terraform init
terraform plan
terraform apply
```

## Production Deployments (`terraform/production/`)

Deploys to production using long-living, pre-allocated resources:
- **Uses existing database** (pre-allocated, never recreated)
- **Runs schema migrations** (always executed via migration app)
- **Uses pre-allocated domain** (DNS records preconfigured, not managed by Terraform)
- Uses `budget-prod` project (all resources assigned to this project)
- Backend state: `production/production.tfstate`

### Key Features

1. **Database**: References existing database cluster, database, and user
   - Database is **never recreated** or modified by Terraform
   - Long-living and stable
   
2. **Domain**: References pre-allocated domain
   - DNS records are **preconfigured outside Terraform**
   - Terraform only references the domain for informational purposes
   
3. **Migrations**: Always execute on every deployment
   - Schema migrations run via the migration app before the main app
   - This ensures database schema is always up-to-date

### Usage

```bash
cd terraform/production
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with existing resource names
terraform init
terraform plan
terraform apply
```

## Key Differences

| Feature | PR Deployments | Production Deployments |
|---------|---------------|----------------------|
| Database | Creates new | Uses existing |
| Domain | Optional | Pre-allocated (DNS preconfigured) |
| Schema Setup | Creates schema | Not needed (exists) |
| Migrations | Always runs | Always runs |
| Project | `budget-develop` | `budget-prod` |
| State Path | `pr/{deployment_id}` | `production/production` |

## Schema Migrations

**Schema migrations always execute** on every deployment (both PR and production):
- Runs via the `budget_migrations` app before the main app deployment
- Uses the `PRE_DEPLOY` job kind in DigitalOcean App Platform
- Ensures database schema is always up-to-date with the application code

## Variables

### PR Deployments

- `deployment_id`: Unique identifier (e.g., `pr-123-feature-branch`)
- `github_repo`: GitHub repository
- `docker_image_tag`: Docker image tag to deploy
- `region`: DigitalOcean region

### Production Deployments

- `existing_database_cluster_name`: Name of existing database cluster
- `existing_database_name`: Name of existing database
- `existing_database_user_name`: Name of existing database user
- `domain_name`: Pre-allocated domain name
- `github_repo`: GitHub repository
- `docker_image_tag`: Docker image tag to deploy
- `region`: DigitalOcean region

## Outputs

Both configurations output:
- `app_url`: Application URL
- `app_id`: DigitalOcean App Platform app ID
- `migration_app_id`: Migration app ID
- `database_connection_string`: Database connection string (sensitive)
- `deployment_id`: Deployment identifier

