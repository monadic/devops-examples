# Drift Detector Quickstart

This guide shows the complete workflow of deploying configurations through ConfigHub to Kubernetes.

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
cd drift-detector
bin/install-base
```

**You should see:**
```
🧹 Cleaning up old resources before setup...
  ✅ Cleanup complete
Generated project name: cozy-cub-drift-detector
🚀 Setting up ConfigHub spaces and units for drift-detector...
Creating space: cozy-cub-drift-detector
Creating filters space: cozy-cub-drift-detector-filters
Creating filters...
Creating base space: cozy-cub-drift-detector-base
Loading base configurations as units...
Creating sets for grouped operations...
✅ Base setup complete!
```

**Verify the structure:**
```bash
cub unit list --space cozy-cub-drift-detector-base
```

**You should see:**
```
NAME                         SPACE                           TARGET    STATUS
namespace                    cozy-cub-drift-detector-base              NoLive
drift-detector-rbac          cozy-cub-drift-detector-base              NoLive
drift-detector-deployment    cozy-cub-drift-detector-base              NoLive
drift-detector-service       cozy-cub-drift-detector-base              NoLive
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
cub worker list --space cozy-cub-drift-detector-base
```

**You should see:**
```
NAME                                  READY   STATUS    RESTARTS   AGE
devops-test-worker-85df78d66c-trgrl   1/1     Running   0          30s

NAME                  CONDITION    SPACE
devops-test-worker    Ready        cozy-cub-drift-detector-base
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

Applying drift-detector-rbac...
Successfully completed Apply on unit drift-detector-rbac

Applying drift-detector-deployment...
Successfully completed Apply on unit drift-detector-deployment

Applying drift-detector-service...
Successfully completed Apply on unit drift-detector-service

✅ All units applied successfully!
```

**Verify deployment:**
```bash
cub unit list --space cozy-cub-drift-detector-base
kubectl get all -n devops-apps
```

**You should see:**
```
# ConfigHub shows units are deployed
NAME                         STATUS    TARGET
namespace                    Ready     k8s-devops-test-worker
drift-detector-rbac          Ready     k8s-devops-test-worker
drift-detector-deployment    Ready     k8s-devops-test-worker
drift-detector-service       Ready     k8s-devops-test-worker

# Kubernetes shows running resources
NAME                                 READY   STATUS    RESTARTS   AGE
pod/drift-detector-858b69844-t66nl   1/1     Running   0          1m

NAME                     TYPE        CLUSTER-IP     PORT(S)
service/drift-detector   ClusterIP   10.96.59.255   8080/TCP

NAME                             READY   UP-TO-DATE   AVAILABLE
deployment.apps/drift-detector   1/1     1            1
```

## Step 4: Fix and Re-apply Configurations

This demonstrates the ConfigHub workflow for fixing configurations:

**Scenario:** The deployment is using a bad image. Let's fix it.

```bash
# Update the unit in ConfigHub with correct image
cat > /tmp/fixed-deployment.yaml <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: drift-detector
  namespace: devops-apps
spec:
  replicas: 1
  selector:
    matchLabels:
      app: drift-detector
  template:
    metadata:
      labels:
        app: drift-detector
    spec:
      containers:
      - name: drift-detector
        image: nginx:alpine  # Fixed image
        ports:
        - containerPort: 80
EOF

# Update unit in ConfigHub
cub unit update drift-detector-deployment /tmp/fixed-deployment.yaml \
  --space cozy-cub-drift-detector-base \
  --change-desc "Fix: Use nginx:alpine image"

# Re-apply to Kubernetes
cub unit apply drift-detector-deployment \
  --space cozy-cub-drift-detector-base
```

**You should see:**
```
Successfully updated unit drift-detector-deployment
Successfully completed Apply on unit drift-detector-deployment
```

**Verify the fix:**
```bash
kubectl get pods -n devops-apps
kubectl describe pod -n devops-apps <pod-name> | grep Image:
```

**You should see:**
```
NAME                             READY   STATUS    RESTARTS   AGE
drift-detector-858b69844-t66nl   1/1     Running   0          30s

Image: nginx:alpine  # ✅ Fixed!
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

5. Fix and re-apply
   └─> Update unit in ConfigHub
   └─> Re-apply to Kubernetes
   └─> Kubernetes resources updated
```

### Why This Pattern?

✅ **Single Source of Truth**: ConfigHub manages all configuration
✅ **Audit Trail**: Every change tracked in ConfigHub
✅ **Rollback**: Easy rollback via ConfigHub revisions
✅ **Multi-Environment**: Apply same config to dev/staging/prod
✅ **No kubectl**: All changes through ConfigHub (GitOps ready)

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
✅ Found project: cozy-cub-drift-detector
✅ Found 4 units in ConfigHub
✅ Worker is Ready
✅ Targets set on 4 units
✅ 4 units in Ready state
✅ Namespace devops-apps exists
✅ Found 1/1 pods running
✅ Workflow test complete!
```

## Troubleshooting

### Worker not connecting

```bash
# Check worker pod logs
kubectl logs -n confighub deployment/devops-test-worker

# Check worker status
cub worker list --space cozy-cub-drift-detector-base
```

### Units stuck in "NoLive"

```bash
# Check if target is set
cub unit list --space cozy-cub-drift-detector-base

# Set target if missing
cub unit set-target k8s-devops-test-worker \
  --where "Space.Slug = 'cozy-cub-drift-detector-base'" \
  --space cozy-cub-drift-detector-base
```

### Apply command times out

```bash
# Use --wait=false for async apply
cub unit apply <unit-name> --space <space> --wait=false

# Check apply status later
cub unit list --space <space>
```

## Cleanup

```bash
# Delete from Kubernetes
kubectl delete namespace devops-apps
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
- See bin/demo for a simulated workflow demo
- Learn about push-upgrade pattern for environment promotion
