# Cost Impact Monitor Quickstart

This guide shows the complete workflow of deploying the cost impact monitor through ConfigHub to Kubernetes.

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
cd cost-impact-monitor
bin/install-base
```

**You should see:**
```
🧹 Cleaning up old resources before setup...
  ✅ Cleanup complete
Generated project name: cozy-cub-cost-monitor
🚀 Setting up ConfigHub spaces and units for cost-impact-monitor...
Creating space: cozy-cub-cost-monitor
Creating filters space: cozy-cub-cost-monitor-filters
Creating filters...
Creating base space: cozy-cub-cost-monitor-base
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
NAME                            SPACE                           STATUS
namespace                       cozy-cub-cost-monitor-base      NoLive
cost-impact-monitor-rbac        cozy-cub-cost-monitor-base      NoLive
cost-impact-monitor-deployment  cozy-cub-cost-monitor-base      NoLive
cost-impact-monitor-service     cozy-cub-cost-monitor-base      NoLive
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
devops-test-worker    Ready        cozy-cub-cost-monitor-base
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
  Success: 4 unit(s)
  Context: target k8s-devops-test-worker

📦 Applying units to Kubernetes...
Applying namespace...
Successfully completed Apply on unit namespace

Applying cost-impact-monitor-rbac...
Successfully completed Apply on unit cost-impact-monitor-rbac

Applying cost-impact-monitor-deployment...
Successfully completed Apply on unit cost-impact-monitor-deployment

Applying cost-impact-monitor-service...
Successfully completed Apply on unit cost-impact-monitor-service

✅ All units applied successfully!
```

**Verify deployment:**
```bash
PROJECT=$(cat .cub-project)
cub unit list --space $PROJECT-base
kubectl get all -n cost-monitoring
```

**You should see:**
```
# ConfigHub shows units are deployed
NAME                            STATUS    TARGET
namespace                       Ready     k8s-devops-test-worker
cost-impact-monitor-rbac        Ready     k8s-devops-test-worker
cost-impact-monitor-deployment  Ready     k8s-devops-test-worker
cost-impact-monitor-service     Ready     k8s-devops-test-worker

# Kubernetes shows running resources
NAME                                      READY   STATUS    RESTARTS   AGE
pod/cost-impact-monitor-858b69844-t66nl   1/1     Running   0          1m

NAME                          TYPE        CLUSTER-IP     PORT(S)
service/cost-impact-monitor   ClusterIP   10.96.59.255   8083/TCP

NAME                                  READY   UP-TO-DATE   AVAILABLE
deployment.apps/cost-impact-monitor   1/1     1            1
```

## Step 4: Access the Dashboard

The cost impact monitor includes a web dashboard for viewing cost analysis:

```bash
# Port-forward to access the dashboard
kubectl port-forward -n cost-monitoring svc/cost-impact-monitor 8083:8083

# Open in browser
open http://localhost:8083
```

**You should see:**
- Real-time cost monitoring across all ConfigHub spaces
- Pending changes with cost impact
- Pre-deployment cost warnings
- Post-deployment verification

## Step 5: Test Cost Monitoring

Create a test deployment to see the monitor in action:

```bash
# Create a test unit in ConfigHub
cub unit create test-deployment k8s/test-deployment.yaml \
  --space $PROJECT-base \
  --label tier=critical

# The monitor will detect the new unit and show cost impact
# Check the dashboard at http://localhost:8083
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
   └─> Monitor tracks cost impact

5. Access dashboard
   └─> View cost monitoring and predictions
```

### Why This Pattern?

✅ **Single Source of Truth**: ConfigHub manages all configuration
✅ **Audit Trail**: Every change tracked in ConfigHub
✅ **Pre-deployment Analysis**: See cost impact before applying
✅ **Multi-Space Monitoring**: Track costs across all environments
✅ **No kubectl**: All changes through ConfigHub (GitOps ready)
✅ **Trigger-based**: Automatic cost warnings

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
✅ Found project: cozy-cub-cost-monitor
✅ Found 4 units in ConfigHub
✅ Worker is Ready
✅ Targets set on 4 units
✅ 4 units in Ready state
✅ Namespace cost-monitoring exists
✅ Found 1/1 pods running
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
kubectl get svc -n cost-monitoring

# Check if pod is running
kubectl get pods -n cost-monitoring

# Check pod logs
kubectl logs -n cost-monitoring deployment/cost-impact-monitor
```

### Monitor not detecting changes

```bash
# Check ConfigHub authentication
cub auth status

# Verify spaces are accessible
cub space list

# Check monitor logs
kubectl logs -n cost-monitoring deployment/cost-impact-monitor
```

## Cleanup

```bash
# Delete from Kubernetes
kubectl delete namespace cost-monitoring
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
- See SCENARIO-HELM-FLUX.md for real-world use case
- Learn about push-upgrade pattern for environment promotion
- Integrate with cost-optimizer for complete cost intelligence
