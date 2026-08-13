# GitHub Environment Setup for Auto-Termination

To enable the modern workflow_dispatch auto-termination with wait timers, you need to create a repository environment.

## 🔧 Setup Instructions

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

The deploy workflow reads this wait timer via the GitHub API for PR comments and auto-termination scheduling. This environment setting is the single source of truth for when PR deployments are destroyed.

### 3. Environment Configuration

```yaml
Environment Name: termination-delay
Protection Rules:
  ✅ Wait timer: your chosen delay (e.g. 5 or 30 minutes)
  ❌ Required reviewers: (leave unchecked)
  ❌ Prevent self-review: (leave unchecked)
  ❌ Restrict pushes: (leave unchecked)
```

## 🚀 How It Works

### Without Environment (Current Issue):
```
Deploy Workflow (5 min) → Trigger Auto-Terminate → Auto-Terminate Runs Immediately
                                                   ↓
                                              Sleep 30 minutes (wastes runner)
```

### With Environment Wait Timer (Correct):
```
Deploy Workflow (5 min) → Trigger Auto-Terminate → Environment Wait (NO RUNNER)
                                                   ↓
                                              Auto-Terminate Runs (~30 sec)
```

## ⚡ Benefits

- **Zero runner waste**: No sleep/wait during the delay
- **Automatic execution**: Workflow starts automatically after wait timer
- **Cost effective**: Only pays for actual execution time (~30 seconds)
- **GitHub native**: Built-in environment protection feature
- **Reliable**: GitHub manages the timing, not custom code

## 🔍 Verification

After setup, you should see:
1. Deploy workflow completes in ~5 minutes
2. Auto-terminate workflow shows "Waiting for environment approval" 
3. After the configured wait timer, auto-terminate workflow runs automatically
4. Total runner time: ~5.5 minutes instead of 35+ minutes

## 📊 Cost Comparison

| Approach | Runner Time | Cost per Deployment |
|----------|------------|-------------------|
| **Sleep-based** | 35+ minutes | ~$0.28 |
| **Environment Timer** | 5.5 minutes | ~$0.044 |
| **Savings** | 85% less | **84% cheaper** |

The environment wait timer is the key to eliminating runner waste while maintaining precise timing!
