# Flux-Based Drift Detector

This version leverages Flux's built-in drift detection capabilities instead of implementing our own.

## How It Works

1. **Flux does the detection**: Flux continuously reconciles and detects drift
2. **We read Flux status**: Our app queries Flux CRDs for drift conditions
3. **Claude analyzes**: When drift is found, Claude suggests remediation
4. **ConfigHub integration**: Creates fixes in ConfigHub spaces

## Why This Approach?

### Advantages
- **Sophisticated detection**: Flux has battle-tested drift detection
- **Efficient**: Uses Kubernetes watch APIs, not polling
- **Comprehensive**: Detects all types of drift (RBAC, CRDs, etc.)
- **Less code**: We don't maintain detection logic

### Challenges
- **Requires Flux**: Cluster must have Flux installed
- **Git intermediary**: Flux watches Git, not ConfigHub directly
- **Additional complexity**: Need to bridge ConfigHub → Git → Flux

## Flux Drift Detection Features We Use

1. **Kustomization status conditions**
   - `DriftDetected` - Direct drift detection
   - `ReconciliationFailed` - Possible drift
   - `HealthCheckFailed` - Resource unhealthy

2. **HelmRelease status**
   - `TestFailed` - Helm tests detect drift
   - `UpgradeFailed` - Can't apply due to drift

3. **Inventory tracking**
   - Flux maintains inventory of all resources
   - Detects external modifications

## ConfigHub Integration Options

### Option 1: Git Bridge (Current)
```
ConfigHub → Git Sync Service → Git → Flux → Kubernetes
```

### Option 2: Direct Integration (Future)
```
ConfigHub → Flux Source Controller → Kubernetes
```
Would require custom Flux source controller for ConfigHub.

## Running

```bash
# Prerequisites: Flux must be installed in cluster
flux check

# Set environment
export FLUX_NAMESPACE=flux-system
export KUBECONFIG=/path/to/kubeconfig

# Run
go run main.go
```

## Example Output

```
2025/09/21 16:30:00 Flux-based drift detector started
2025/09/21 16:30:00 Checking Flux resources for drift...
2025/09/21 16:30:00 === FLUX DRIFT REPORT ===
2025/09/21 16:30:00 DRIFT: Kustomization/my-app in flux-system
2025/09/21 16:30:00   Message: Deployment 'backend' has been modified outside of Flux
2025/09/21 16:30:00   Source: flux-system/my-app
2025/09/21 16:30:00   Claude suggests: Force reconcile recommended
```

## Comparison with Direct Approach

See [DRIFT-DETECTION-COMPARISON.md](../DRIFT-DETECTION-COMPARISON.md) for detailed comparison.