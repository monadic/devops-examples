# ConfigHub Worker Setup Guide

## 🚨 IMPORTANT: Worker Required for ConfigHub ↔ Kubernetes Integration

All DevOps examples require a **ConfigHub worker** to bridge between ConfigHub and your Kubernetes cluster. Without this worker, units created in ConfigHub won't be deployed to Kubernetes.

## Quick Start

### 1. Run the Setup Script
```bash
cd /Users/alexisrichardson/github-repos/devops-examples
./setup-worker.sh
```

This script will:
- ✅ Create a ConfigHub worker
- ✅ Configure targets for your Kubernetes cluster
- ✅ Link units to targets for deployment

### 2. Start the Worker (Required!)
In a **separate terminal**, run:
```bash
cub worker run devops-worker
```

Keep this running while using the DevOps examples.

### 3. Apply Units to Kubernetes
```bash
# For drift-detector
cub unit apply --all --space drift-test-demo

# Check deployment status
kubectl get deployments -n drift-test
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
cub worker create devops-worker --space default
```

### Step 2: Create Targets
```bash
# For drift-test namespace
cub target create kind-drift-test \
  '{"KubeContext":"kind-devops-test","KubeNamespace":"drift-test","WaitTimeout":"2m0s"}' \
  devops-worker \
  --space drift-test-demo

# For default namespace (cost-optimizer)
cub target create kind-default \
  '{"KubeContext":"kind-devops-test","KubeNamespace":"default","WaitTimeout":"2m0s"}' \
  devops-worker \
  --space default
```

### Step 3: Update Units with Target
```bash
# Get target ID
TARGET_ID=$(cub target get kind-drift-test --space drift-test-demo --json | jq -r '.TargetID')

# Update each unit
cub unit update backend-api-unit --space drift-test-demo \
  --patch --data '{"TargetID":"'$TARGET_ID'"}'
```

### Step 4: Run the Worker
```bash
cub worker run devops-worker
```

## Troubleshooting

### Worker Not Found
If you see "BridgeWorker not found", ensure:
1. Worker was created in the correct space
2. You're using the correct worker name
3. You're authenticated: `cub auth get-token`

### Target Creation Failed
If target creation fails:
1. Make sure the worker is running first
2. Check kubectl context: `kubectl config current-context`
3. Verify namespace exists: `kubectl get ns`

### Units Not Deploying
If units aren't deploying to Kubernetes:
1. Check unit has a target: `cub unit get <unit-name> --space <space>`
2. Verify worker is running and connected
3. Check worker logs for errors
4. Apply manually: `cub unit apply <unit-name> --space <space>`

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