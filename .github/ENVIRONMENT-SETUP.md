# GitHub Environment Setup for Auto-Termination

PR deployments are destroyed by the **Terminate PR Deployment** workflow, which uses the `termination-delay` environment wait timer after successful deploys.

## Setup Instructions

### 1. Create Repository Environment

1. Go to your GitHub repository
2. Navigate to **Settings** → **Environments**
3. Click **New environment**
4. Name it: `termination-delay`

### 2. Configure Environment Wait Timer

1. In the `termination-delay` environment settings:
2. Click **Add protection rule**
3. Enable **Wait timer**
4. Set wait time to your desired delay (e.g. **5 minutes** for testing, **30 minutes** for production-like runs)
5. Click **Save protection rules**

This environment setting is the single source of truth for how long PR deployments live after a successful deploy.

### 3. Environment Configuration

```yaml
Environment Name: termination-delay
Protection Rules:
  ✅ Wait timer: your chosen delay (e.g. 5 or 30 minutes)
  ❌ Required reviewers: (leave unchecked)
  ❌ Prevent self-review: (leave unchecked)
  ❌ Restrict pushes: (leave unchecked)
```

## How It Works

After a successful PR deploy completes:

```
Deploy completes → Terminate PR Deployment starts → termination-delay wait (no runner)
                                                  → Terraform destroy (~30 sec)
                                                  → Notify posts PR comment + Slack
```

After a failed PR deploy:

```
Deploy fails → Terminate PR Deployment starts immediately (no wait)
            → Terraform destroy if partial resources exist
            → Notify posts cleanup result
```

Manual cleanup: run **Terminate PR Deployment** with `skip_environment_wait: true` (default).

## Verification

After setup, on a successful PR deploy you should see:

1. CI completes (~few minutes)
2. Deploy completes (~5–15 minutes)
3. Deploy PR run posts deploy success (Slack + PR comment)
4. Terminate workflow shows waiting on `termination-delay`
5. After the wait timer, terminate destroys resources and posts termination result (Slack + PR comment)

## Cost Comparison

| Approach | Runner time during wait | Notes |
|----------|-------------------------|-------|
| Sleep in workflow | Full delay billed | Wasteful |
| Environment wait timer | ~0 during wait | Recommended |

The environment wait timer avoids billing runner minutes during the delay.
