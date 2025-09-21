# DevOps Examples

Practical DevOps automation applications using ConfigHub and Claude AI.

## Applications

### 1. Drift Detector
Detects configuration drift between ConfigHub (desired state) and Kubernetes (actual state).

- **How it works**: Compares ConfigHub units with live Kubernetes resources
- **Detection method**: Direct API comparison (not using Flux/Argo)
- **AI Integration**: Claude analyzes drift and suggests fixes

### 2. Cost Optimizer (Coming Soon)
Analyzes resource usage and suggests cost optimizations.

## Architecture

Each app follows the same simple pattern:
1. Polls for state (ConfigHub + Kubernetes)
2. Analyzes with Claude
3. Creates fixes in new ConfigHub spaces
4. Can be applied manually or automatically

## Getting Started

```bash
# Build drift-detector
cd drift-detector
go build
./drift-detector

# Or run in Kubernetes
kubectl apply -f drift-detector/k8s/
```