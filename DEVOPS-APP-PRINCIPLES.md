# DevOps App Core Principles

These are the fundamental principles that every DevOps app in our architecture must follow. They're derived from real implementation experience and the global-app canonical patterns.

## Recent ConfigHub Capabilities (2024-11)

### Functions Framework
- **Functions**: ConfigHub now has a functions framework for operations on units
- **Commands**: `cub function list`, `cub function explain`, `cub function do`
- **Categories**: Inspection (read-only), Modification (mutating), Validation
- **Use case**: Programmatic manipulation of configurations without local files

### ChangeSets
- **Purpose**: Group related changes together (different from Sets)
- **Automatic filtering**: Functions can operate on ChangeSet members
- **Bulk operations**: Delete, update ChangeSets in bulk

### Helm Integration
- **Native Helm support**: `cub helm install`, `cub helm upgrade`
- **Chart management**: Store Helm charts as ConfigHub units
- **Values management**: ConfigHub handles Helm values merging

### Package System
- **Package commands**: Create and load configuration packages
- **Unit packaging**: Bundle related units for distribution

## PRINCIPLE #0: Each Component is a Config Unit, App is the Collection

**The Most Important Principle**: Every Kubernetes component (deployment, service, configmap, secret, etc.) is a **separate ConfigHub Unit**. The app itself is the **collection** of these units, grouped via Labels and Filters.

### The Mental Model
```
App = Collection of Units with same labels
    ├── Unit: namespace.yaml    (label: app=myapp)
    ├── Unit: rbac.yaml         (label: app=myapp)
    ├── Unit: configmap.yaml    (label: app=myapp)
    ├── Unit: service.yaml      (label: app=myapp)
    └── Unit: deployment.yaml   (label: app=myapp)
```

### Why This Matters
- **Granular Control**: Update individual components without touching others
- **Bulk Operations**: Apply changes to all units in the Set at once
- **Atomic Deployments**: All units succeed or fail together
- **Version Tracking**: Each unit has its own revision history
- **Flexible Composition**: Mix and match units across environments

### Implementation Pattern (Following Global-App)
```bash
# In bin/install-base
PROJECT=$(cub space new-prefix)

# Each Kubernetes component becomes a separate Unit with labels
cub unit create namespace k8s/namespace.yaml \
  --space $PROJECT-base \
  --label app=myapp --label type=infra

cub unit create deployment k8s/deployment.yaml \
  --space $PROJECT-base \
  --label app=myapp --label type=app

cub unit create service k8s/service.yaml \
  --space $PROJECT-base \
  --label app=myapp --label type=app

# Create filter to target all app components
cub filter create myapp-all Unit \
  --where "Labels.app = 'myapp'" \
  --space $PROJECT

# Create filter for just workloads
cub filter create myapp-workloads Unit \
  --where "Labels.app = 'myapp' AND Labels.type = 'app'" \
  --space $PROJECT
```

### In Code
```go
// Get all units for this app via filter
units, _ := app.Cub.ListUnits(sdk.ListUnitsParams{
    SpaceID: spaceID,
    Where:   "Labels.app = 'myapp'",
})

// Update just the deployment unit
app.Cub.UpdateUnit(spaceID, deploymentUnitID, sdk.UpdateUnitRequest{
    Data: newDeploymentYAML,
})

// Bulk update all app workloads
app.Cub.BulkPatchUnits(sdk.BulkPatchParams{
    SpaceID: spaceID,
    Where:   "Labels.app = 'myapp' AND Labels.type = 'app'",
    Patch:   imageUpdatePatch,
})

// Apply all app units
app.Cub.BulkApplyUnits(sdk.BulkApplyParams{
    SpaceID: spaceID,
    Where:   "Labels.app = 'myapp'",
})
```

### Real Example: Drift Detector App
```
drift-detector = Units with label 'app=drift-detector'
    ├── namespace.yaml     (labels: app=drift-detector, type=infra)
    ├── rbac.yaml         (labels: app=drift-detector, type=infra)
    ├── configmap.yaml    (labels: app=drift-detector, type=config)
    ├── service.yaml      (labels: app=drift-detector, type=app)
    └── deployment.yaml   (labels: app=drift-detector, type=app)
```

Each component can be:
- Updated independently (patch just the deployment)
- Versioned separately (configmap v1, deployment v2)
- Applied in order (namespace first, deployment last)
- Filtered by labels (update all workloads)
- Promoted as a group (the entire Set)

## PRINCIPLE #1: ConfigHub Worker is Mandatory

Workers are the bridge between ConfigHub (control plane) and Kubernetes (data plane). Without a worker, units exist in ConfigHub but never deploy.

### Setup Pattern
```bash
cub worker create devops-worker --space $PROJECT
cub worker run devops-worker  # Keep running!
```

## PRINCIPLE #2: Global-App Patterns

Follow the canonical patterns from global-app:
- Unique prefix generation: `cub space new-prefix`
- Space hierarchy: base → dev → staging → prod
- Filters for targeting
- Upstream/downstream relationships

## PRINCIPLE #3: Valid Kubernetes YAML

ConfigHub units must contain properly formatted Kubernetes manifests with all required fields.

## PRINCIPLE #4: Targets Link Units to Clusters

Each unit needs a target to know where to deploy:
```bash
cub target create k8s-target \
  '{"KubeContext":"kind-cluster","KubeNamespace":"default"}' \
  devops-worker --space $PROJECT
```

## PRINCIPLE #5: Complete Auth Flow

Three authentication layers:
1. ConfigHub: `cub auth login`
2. Claude AI: `CLAUDE_API_KEY`
3. Kubernetes: `KUBECONFIG`

## PRINCIPLE #6: Event-Driven Architecture

Use Kubernetes informers for immediate reaction:
```go
app.RunWithInformers(func() error {
    // React to changes immediately
    return processChanges()
})
```

Never use polling loops with `time.Sleep`.

## PRINCIPLE #7: Demo Mode

Every app must have a demo mode for testing without infrastructure:
```go
if len(os.Args) > 1 && os.Args[1] == "demo" {
    return RunDemo()
}
```

## PRINCIPLE #8: Cleanup-First

Always clean up old resources before creating new ones:
```bash
#!/bin/bash
# CRITICAL: Clean up old resources first
if [ -e ".cub-project" ]; then
    OLD_PROJECT=$(cat .cub-project)
    cub space delete $OLD_PROJECT 2>/dev/null || true
fi
# Now create new resources...
```

## PRINCIPLE #9: ConfigHub-Only Commands

**NEVER** use `kubectl` for modifications. All changes go through ConfigHub:
- ✅ `cub unit update backend --patch`
- ❌ `kubectl scale deployment backend`

## PRINCIPLE #10: Self-Deployment

Every DevOps app deploys itself through ConfigHub:
```bash
bin/install-base    # Create ConfigHub structure
bin/install-envs    # Set up environments
bin/apply-all dev   # Deploy via ConfigHub
```

## Testing for Principles

Use the compliance checker to verify:
```bash
./test-app-compliance-quick.sh my-app/
```

Key checks:
- App organized as Set (PRINCIPLE #0)
- No kubectl in code
- Uses bulk operations
- Has demo mode
- Cleanup-first in scripts
- Event-driven architecture

## Why These Principles Matter

1. **Sets Enable Bulk Operations**: Manage entire apps, not individual configs
2. **ConfigHub as Truth**: Single source prevents drift
3. **Event-Driven Beats Polling**: Instant reaction, lower resource usage
4. **Self-Deployment**: Apps manage their own lifecycle
5. **Demo Mode**: Test without infrastructure costs

## The Global-App Pattern

The global-app example demonstrates all principles working together:
- App defined as a Set
- Filters for targeting subsets
- Bulk operations for efficiency
- Environment hierarchy with upstream
- Push-upgrade for propagation

Study `/Users/alexisrichardson/github-repos/confighub-examples/global-app/` for the canonical implementation.