# ConfigHub Worker Setup Guide

## 🚨 IMPORTANT: Worker Required for ConfigHub ↔ Kubernetes Integration

All DevOps examples require a **ConfigHub worker** to bridge between ConfigHub and your Kubernetes cluster. Without this worker, units created in ConfigHub won't be deployed to Kubernetes.

## ⚠️ CRITICAL: Always Use `--include-secret`

When installing ConfigHub workers, **you MUST use the `--include-secret` flag** to generate proper authentication credentials for each worker.

**Without `--include-secret`:**
- Workers will reuse existing secrets with WRONG credentials
- Workers fail with: `[ERROR] Failed to get bridge worker slug: server returned status 404`
- Cannot connect or deploy units

**With `--include-secret`:**
- Each worker gets its own unique `CONFIGHUB_WORKER_SECRET`
- Workers authenticate successfully
- Full deployment workflow works

## Quick Start

Each example has a `bin/setup-worker` script that handles worker creation with proper credentials:

```bash
# For drift-detector
cd drift-detector
bin/setup-worker

# For cost-optimizer
cd cost-optimizer
bin/setup-worker

# For cost-impact-monitor
cd cost-impact-monitor
bin/setup-worker
```

These scripts will:
- ✅ Create a ConfigHub worker in the project space
- ✅ Generate worker deployment with **unique credentials** (`--include-secret`)
- ✅ Deploy worker as a Kubernetes pod in `confighub` namespace
- ✅ Automatically create targets for unit deployment

### Verify Worker Status
```bash
# Check worker is connected
cub worker list --space <your-space>
# Should show: Condition=Ready

# Check targets were created
cub target list --space <your-space>
# Should show: k8s-<worker-name> target

# Check worker pod
kubectl get pods -n confighub
# Should show: <worker-name>-xxx Running
```

## Architecture Overview

```
┌─────────────┐     ┌──────────────┐     ┌────────────┐
│  ConfigHub  │────▶│    Worker    │────▶│ Kubernetes │
│    Units    │     │   (Bridge)   │     │  Cluster   │
└─────────────┘     └──────────────┘     └────────────┘
      ↑                                          ↑
      │                                          │
┌─────────────┐                        ┌────────────┐
│   DevOps    │                        │   Actual   │
│    Apps     │◀──────────────────────▶│ Resources  │
└─────────────┘     (Monitoring)       └────────────┘
```

## Manual Worker Setup

If the script doesn't work, here are the manual steps:

### Step 1: Create a Worker
```bash
cub worker create my-worker --space my-space
```

### Step 2: Install Worker to Kubernetes (WITH --include-secret!)
```bash
# CRITICAL: Use --include-secret to generate unique credentials
cub worker install my-worker \
  --namespace confighub \
  --space my-space \
  --include-secret \
  --export > worker.yaml

kubectl apply -f worker.yaml
```

### Step 3: Verify Worker Connection
```bash
# Wait for worker to connect
sleep 10

# Check worker status
cub worker list --space my-space
# Should show: Condition=Ready

# Check targets were auto-created
cub target list --space my-space
# Should show: k8s-my-worker

# Check worker pod
kubectl logs -n confighub -l app=my-worker --tail=10
# Should show: "Successfully connected to event stream"
```

### Step 4: Set Targets for Units
```bash
# Set target for specific units
cub unit set-target unit-name k8s-my-worker --space my-space

# Or set target for all units in a space
cub unit set-target k8s-my-worker --where "Space.Slug = 'my-space'" --space my-space
```

### Step 5: Apply Units
```bash
cub unit apply unit-name --space my-space

# Check deployment
kubectl get all -n default
```

## Troubleshooting

### Worker Shows 404 Error (MOST COMMON ISSUE)
```
[ERROR] Failed to get bridge worker slug: server returned status 404: 404 Not Found
```

**Root Cause:** Worker is using wrong authentication credentials (missing `--include-secret`).

**Solution:**
1. Delete the worker: `kubectl delete deployment <worker-name> -n confighub`
2. Recreate with `--include-secret`:
   ```bash
   cub worker install <worker-name> --space <space> --include-secret --export > worker.yaml
   kubectl apply -f worker.yaml
   ```
3. Verify: `cub worker list --space <space>` should show `Condition=Ready`

### Worker Condition Shows "Disconnected"
```bash
$ cub worker list --space my-space
NAME       CONDITION       SPACE        LAST-SEEN
my-worker  Disconnected    my-space     0001-01-01 00:00:00
```

**Causes:**
1. Missing `--include-secret` (most common)
2. Worker pod crashed - check logs: `kubectl logs -n confighub -l app=my-worker`
3. Network issues - check worker can reach ConfigHub API

### No Targets Created
If `cub target list` shows empty:
1. Worker must be connected first (Condition=Ready)
2. Targets are auto-created when worker connects
3. If worker is Ready but no targets, restart worker pod

### Units Not Deploying
If units aren't deploying to Kubernetes:
1. Check unit has a target: `cub unit get <unit-name> --space <space>`
2. Set target if missing: `cub unit set-target <unit-name> <target-name> --space <space>`
3. Verify worker is running: `cub worker list --space <space>`
4. Check worker logs: `kubectl logs -n confighub -l app=<worker-name>`
5. Apply manually: `cub unit apply <unit-name> --space <space>`

## Why Is a Worker Needed?

ConfigHub is a **control plane** that stores desired configuration. To actually deploy resources to Kubernetes, you need a **worker** that:

1. **Polls ConfigHub** for units to deploy
2. **Translates** ConfigHub units to Kubernetes resources
3. **Applies** resources to the cluster
4. **Reports** status back to ConfigHub

Without a worker:
- Units exist only in ConfigHub (configuration storage)
- No actual Kubernetes resources are created
- Drift-detector can't compare desired vs actual state

## Integration with DevOps Apps

### drift-detector
- Reads desired state from ConfigHub units
- Reads actual state from Kubernetes
- Compares and detects drift
- **Requires**: Worker to deploy ConfigHub units first

### cost-optimizer
- Analyzes Kubernetes resource usage
- Stores recommendations in ConfigHub
- Can apply optimizations via ConfigHub
- **Requires**: Worker to apply optimization changes

## Best Practices

1. **Always run the worker** when testing DevOps examples
2. **Use separate workers** for different environments (dev, staging, prod)
3. **Monitor worker health** - check `cub worker list` regularly
4. **Use namespaces** to isolate different applications
5. **Set appropriate timeouts** in target configuration

## Next Steps

Once your worker is running:

1. **Test drift-detector**:
   ```bash
   cd drift-detector
   ./drift-detector --space drift-test-demo --namespace drift-test
   ```

2. **Test cost-optimizer**:
   ```bash
   cd cost-optimizer
   ./cost-optimizer
   ```

3. **View in ConfigHub UI**:
   ```bash
   open https://hub.confighub.com/spaces
   ```

Remember: The worker is the critical bridge between ConfigHub's desired state and Kubernetes' actual state!