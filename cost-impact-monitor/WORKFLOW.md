# ConfigHub → Kubernetes Workflow

This document explains the workflow for deploying configurations through ConfigHub to Kubernetes.

## Architecture Overview

```
┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│   ConfigHub │         │   Worker    │         │ Kubernetes  │
│   (Units)   │────────▶│  (Bridge)   │────────▶│  (Deploy)   │
└─────────────┘         └─────────────┘         └─────────────┘
      ▲                                                │
      │                                                │
      │                    Live State                  │
      └────────────────────────────────────────────────┘
```

## Key Components

### 1. ConfigHub Units
- **What**: Configuration definitions stored in ConfigHub
- **Status**: NoLive → Ready → Live
- **Location**: `cozy-cub-drift-detector-base` space
- **View**: `cub unit list --space <space>`

### 2. ConfigHub Worker
- **What**: Bridge service running in Kubernetes
- **Purpose**: Executes `cub unit apply` commands
- **Location**: `confighub` namespace
- **View**: `kubectl get pods -n confighub`

### 3. Targets
- **What**: Link between units and Kubernetes clusters
- **Purpose**: Tell units where to deploy
- **Set**: `cub unit set-target <target> --unit <unit>`
- **View**: `cub target list --space <space>`

### 4. Apply Operations
- **What**: Deploy units to Kubernetes
- **Command**: `cub unit apply <unit> --space <space>`
- **Effect**: Worker creates/updates Kubernetes resources
- **Result**: Unit status changes to "Ready"

## Workflow Steps

### Step 1: Create Units in ConfigHub
```bash
bin/install-base
```

**What happens:**
- Creates ConfigHub space (e.g., `cozy-cub-drift-detector-base`)
- Loads YAML files as ConfigHub units
- Units are in "NoLive" status (exist but not deployed)

**Verify:**
```bash
cub unit list --space <space>
# STATUS should be "NoLive"
```

### Step 2: Set Up Worker
```bash
bin/setup-worker
```

**What happens:**
- Creates ConfigHub worker entity
- Deploys worker pod to Kubernetes
- Worker connects to ConfigHub API
- Worker ready to execute applies

**Verify:**
```bash
cub worker list --space <space>
# CONDITION should be "Ready"

kubectl get pods -n confighub
# STATUS should be "Running"
```

### Step 3: Assign Targets
```bash
cub unit set-target k8s-devops-test-worker \
  --where "Space.Slug = '<space>'" \
  --space <space>
```

**What happens:**
- Links units to the worker's target
- Units now know where to deploy
- STATUS remains "NoLive" (not deployed yet)

**Verify:**
```bash
cub unit list --space <space>
# TARGET column should show "k8s-devops-test-worker"
```

### Step 4: Apply Units
```bash
cub unit apply <unit-name> --space <space>
```

**What happens:**
1. ConfigHub sends apply request to worker
2. Worker reads unit data (YAML)
3. Worker applies YAML to Kubernetes
4. Worker reports back to ConfigHub
5. ConfigHub updates unit status to "Ready"
6. Live state tracked in ConfigHub

**Verify:**
```bash
# ConfigHub side
cub unit list --space <space>
# STATUS should be "Ready"

# Kubernetes side
kubectl get all -n <namespace>
# Resources should exist and be running
```

### Step 5: Update and Re-apply
```bash
# Update unit in ConfigHub
cub unit update <unit-name> <file.yaml> \
  --space <space> \
  --change-desc "Description of change"

# Re-apply to Kubernetes
cub unit apply <unit-name> --space <space>
```

**What happens:**
1. Unit data updated in ConfigHub
2. Apply triggers worker to update Kubernetes
3. Kubernetes resources updated
4. Live state reflects changes

## Unit Status Lifecycle

```
NoLive ──────▶ Ready ──────▶ Degraded
  │              │              │
  │              │              │
  └──────────────┴──────────────┘
         (re-apply fixes)
```

- **NoLive**: Unit exists but not deployed
- **Ready**: Unit deployed and healthy
- **Degraded**: Unit deployed but has issues
- **Progressing**: Unit is being updated

## Common Operations

### View All Units
```bash
cub unit list --space <space>
```

### View Unit Data (YAML)
```bash
cub unit get <unit-name> --space <space> --json | jq -r '.Data' | base64 -d
```

### View Live State
```bash
cub unit get <unit-name> --space <space> --json | jq -r '.LiveState' | base64 -d
```

### View Worker Status
```bash
cub worker list --space <space>
kubectl get pods -n confighub
kubectl logs -n confighub deployment/<worker-name>
```

### View Target Information
```bash
cub target list --space <space>
```

### Apply All Units
```bash
for unit in $(cub unit list --space <space> --json | jq -r '.[].Slug'); do
  cub unit apply $unit --space <space> --wait=false
done
```

## Troubleshooting

### Unit Stuck in "NoLive"
**Possible causes:**
1. No target assigned
2. Worker not running
3. Apply not executed

**Solutions:**
```bash
# Check target
cub unit list --space <space>  # Look for TARGET column

# Set target if missing
cub unit set-target <target> --unit <unit> --space <space>

# Check worker
cub worker list --space <space>
kubectl get pods -n confighub

# Apply unit
cub unit apply <unit> --space <space>
```

### Unit Shows "Degraded"
**Meaning:** Deployed but has issues (e.g., pod not starting)

**Solutions:**
```bash
# Check Kubernetes resources
kubectl get all -n <namespace>
kubectl describe pod <pod-name> -n <namespace>

# Check ConfigHub live state
cub unit get <unit> --space <space> --json | jq '.LiveState'

# Fix and re-apply
cub unit update <unit> <fixed-file.yaml> --space <space>
cub unit apply <unit> --space <space>
```

### Worker Not Ready
**Possible causes:**
1. Secret missing
2. Network issues
3. Authentication failed

**Solutions:**
```bash
# Check worker pod
kubectl get pods -n confighub
kubectl logs -n confighub deployment/<worker-name>

# Check secret exists
kubectl get secret confighub-worker-env -n confighub

# Recreate worker
bin/setup-worker
```

## Best Practices

1. **Always use ConfigHub for changes**
   - Don't use `kubectl apply` directly
   - Update units in ConfigHub, then apply

2. **Use change descriptions**
   - Add `--change-desc` to track changes
   - Makes audit trail clear

3. **Check status before moving on**
   - Wait for "Ready" before next step
   - Use `--wait=true` (default) for synchronous applies

4. **Use filters for bulk operations**
   - Group related units with labels
   - Use filters to apply to many units at once

5. **Keep environments separate**
   - Use different spaces for dev/staging/prod
   - Use upstream/downstream for promotion

## Reference

- **QUICKSTART.md**: Step-by-step setup guide
- **README.md**: Full architecture and features
- **bin/test-workflow**: Validation script
- **ConfigHub docs**: https://docs.confighub.com
