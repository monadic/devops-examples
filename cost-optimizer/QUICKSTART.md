# Cost Optimizer Quickstart

This guide shows the complete workflow of deploying the AI-powered cost optimizer through ConfigHub to Kubernetes.

## Prerequisites

```bash
# Install and authenticate with ConfigHub CLI
cub auth login

# Upgrade to latest version
cub upgrade

# Create a Kind cluster for testing
kind create cluster --name devops-test
kubectl cluster-info
```

## Step 1: Create ConfigHub Structure

This creates spaces, filters, and base units in ConfigHub:

```bash
cd cost-optimizer
bin/install-base
```

**You should see:**
```
🧹 Cleaning up old resources before setup...
  ✅ Cleanup complete
Generated project name: cozy-cub-cost-optimizer
🚀 Setting up ConfigHub spaces and units for cost-optimizer...
Creating space: cozy-cub-cost-optimizer
Creating filters space: cozy-cub-cost-optimizer-filters
Creating filters...
Creating base space: cozy-cub-cost-optimizer-base
Loading base configurations as units...
Creating sets for grouped operations...
✅ Base setup complete!
```

**Verify the structure:**
```bash
PROJECT=$(cat .cub-project)
cub unit list --space $PROJECT-base
```

**You should see:**
```
NAME                         SPACE                           STATUS
namespace                    cozy-cub-cost-optimizer-base    NoLive
cost-optimizer-rbac          cozy-cub-cost-optimizer-base    NoLive
cost-optimizer-deployment    cozy-cub-cost-optimizer-base    NoLive
cost-optimizer-service       cozy-cub-cost-optimizer-base    NoLive
metrics-server              cozy-cub-cost-optimizer-base    NoLive
```

Note: STATUS = "NoLive" means the units exist in ConfigHub but haven't been deployed yet.

## Step 2: Set Up ConfigHub Worker

A worker is required to execute `cub unit apply` commands and deploy to Kubernetes.

**What is a worker?** A ConfigHub worker is a bridge service that runs in your Kubernetes cluster and executes apply operations. When you run `cub unit apply`, the worker receives the command and deploys the configuration to Kubernetes.

```bash
bin/setup-worker
```

**You should see:**
```
🔧 Setting up ConfigHub worker for Kubernetes deployment
================================================

Creating confighub namespace in cluster...
namespace/confighub created

Creating ConfigHub worker...
Successfully created bridgeworker devops-test-worker (600900ba-6d95-4e39-8fe2-1fb778ed8b88)

Installing worker to Kubernetes cluster...
deployment.apps/devops-test-worker created
secret/confighub-worker-env created

Waiting for worker to connect...
✅ Worker connected!

NAME                  CONDITION    LAST-SEEN
devops-test-worker    Ready        2025-10-01 15:20:17
```

**Verify worker is running:**
```bash
kubectl get pods -n confighub
PROJECT=$(cat .cub-project)
cub worker list --space $PROJECT-base
```

**You should see:**
```
NAME                                  READY   STATUS    RESTARTS   AGE
devops-test-worker-85df78d66c-trgrl   1/1     Running   0          30s

NAME                  CONDITION    SPACE
devops-test-worker    Ready        cozy-cub-cost-optimizer-base
```

## Step 3: Set Targets and Apply Units

Now we'll assign the Kubernetes target to units and deploy them:

```bash
bin/apply-base
```

**You should see:**
```
🎯 Setting targets for all units...
Bulk set-target operation completed:
  Success: 5 unit(s)
  Context: target k8s-devops-test-worker

📦 Applying units to Kubernetes...
Applying namespace...
Successfully completed Apply on unit namespace

Applying metrics-server...
Successfully completed Apply on unit metrics-server

Applying cost-optimizer-rbac...
Successfully completed Apply on unit cost-optimizer-rbac

Applying cost-optimizer-deployment...
Successfully completed Apply on unit cost-optimizer-deployment

Applying cost-optimizer-service...
Successfully completed Apply on unit cost-optimizer-service

✅ All units applied successfully!
```

**Verify deployment:**
```bash
PROJECT=$(cat .cub-project)
cub unit list --space $PROJECT-base
kubectl get all -n cost-optimizer
```

**You should see:**
```
# ConfigHub shows units are deployed
NAME                         STATUS    TARGET
namespace                    Ready     k8s-devops-test-worker
metrics-server              Ready     k8s-devops-test-worker
cost-optimizer-rbac          Ready     k8s-devops-test-worker
cost-optimizer-deployment    Ready     k8s-devops-test-worker
cost-optimizer-service       Ready     k8s-devops-test-worker

# Kubernetes shows running resources
NAME                                 READY   STATUS    RESTARTS   AGE
pod/cost-optimizer-858b69844-t66nl   1/1     Running   0          1m
pod/metrics-server-7c4f9b8d-xyz      1/1     Running   0          1m

NAME                     TYPE        CLUSTER-IP     PORT(S)
service/cost-optimizer   ClusterIP   10.96.59.255   8081/TCP

NAME                             READY   UP-TO-DATE   AVAILABLE
deployment.apps/cost-optimizer   1/1     1            1
deployment.apps/metrics-server   1/1     1            1
```

## Step 4: Access the Dashboard

The cost optimizer includes a web dashboard for viewing cost analysis:

```bash
# Port-forward to access the dashboard
kubectl port-forward -n cost-optimizer svc/cost-optimizer 8081:8081

# Open in browser
open http://localhost:8081
```

**You should see:**
- Real-time cost analysis with metrics-server data
- AI-powered recommendations from Claude
- Resource utilization metrics
- Cost savings opportunities
- **🤖 Claude API History** - Real-time API request/response viewer showing:
  - Last 10 Claude API calls with timestamps
  - Truncated prompts and responses
  - Duration and success status
  - Debug toggle via `CLAUDE_DEBUG_LOGGING=true`

## Step 5: Enable OpenCost (Optional)

For real cloud cost data instead of estimates:

```bash
# Configure OpenCost integration
bin/configure-opencost true

# Install OpenCost
bin/install-opencost-base
bin/install-opencost-envs
bin/apply-opencost dev
```

## Key Concepts

### ConfigHub → Kubernetes Workflow

```
1. Create units in ConfigHub (bin/install-base)
   └─> Units in "NoLive" state

2. Set up worker (bin/setup-worker)
   └─> Worker connects to ConfigHub and Kubernetes

3. Assign targets to units
   └─> Units now know where to deploy

4. Apply units (cub unit apply)
   └─> Worker executes deployment
   └─> Units transition to "Ready" state
   └─> Resources created in Kubernetes

5. Access dashboard
   └─> View cost analysis and recommendations
```

### Why This Pattern?

✅ **Single Source of Truth**: ConfigHub manages all configuration
✅ **Audit Trail**: Every change tracked in ConfigHub
✅ **Rollback**: Easy rollback via ConfigHub revisions
✅ **Multi-Environment**: Apply same config to dev/staging/prod
✅ **No kubectl**: All changes through ConfigHub (GitOps ready)
✅ **AI-Powered**: Claude integration for intelligent cost recommendations

## Validate Everything Works

Run the validation script to check all steps:

```bash
bin/test-workflow
```

**You should see:**
```
🧪 Testing ConfigHub → Kubernetes Workflow
==========================================

✅ cub CLI found
✅ kubectl found
✅ Kubernetes cluster accessible
✅ Found project: cozy-cub-cost-optimizer
✅ Found 5 units in ConfigHub
✅ Worker is Ready
✅ Targets set on 5 units
✅ 5 units in Ready state
✅ Namespace cost-optimizer exists
✅ Found 2/2 pods running
✅ Workflow test complete!
```

## Troubleshooting

### Worker not connecting

```bash
# Check worker pod logs
kubectl logs -n confighub deployment/devops-test-worker

# Check worker status
PROJECT=$(cat .cub-project)
cub worker list --space $PROJECT-base
```

### Units stuck in "NoLive"

```bash
# Check if target is set
PROJECT=$(cat .cub-project)
cub unit list --space $PROJECT-base

# Set target if missing
cub unit set-target k8s-devops-test-worker \
  --where "Space.Slug = '$PROJECT-base'" \
  --space $PROJECT-base
```

### Dashboard not accessible

```bash
# Check if service exists
kubectl get svc -n cost-optimizer

# Check if pod is running
kubectl get pods -n cost-optimizer

# Check pod logs
kubectl logs -n cost-optimizer deployment/cost-optimizer
```

## Cleanup

```bash
# Delete from Kubernetes
kubectl delete namespace cost-optimizer
kubectl delete namespace confighub

# Delete from ConfigHub
PROJECT=$(cat .cub-project)
cub space delete $PROJECT-base
cub space delete $PROJECT-filters
cub space delete $PROJECT

# Delete Kind cluster
kind delete cluster --name devops-test
```

## Next Steps

- Read the full README.md for architecture details
- Check out bin/install-envs for multi-environment setup
- See bin/analyze-confighub for cost analysis examples
- Learn about push-upgrade pattern for environment promotion
- Enable OpenCost for real cloud cost data
