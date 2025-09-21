# Drift Detector

A Kubernetes application that detects configuration drift between ConfigHub (desired state) and Kubernetes (actual state).

## How It Works

### Drift Detection Method

This detector uses **direct API comparison**, not Flux or ArgoCD:

1. **Desired State**: Fetches configuration units from ConfigHub via API
2. **Actual State**: Queries Kubernetes API for current resource state
3. **Comparison**: Compares key fields (replicas, images, resources, ports)
4. **AI Analysis**: Optionally uses Claude to provide deeper analysis
5. **Reporting**: Logs drift items and proposed fixes

### Why Not Flux/ArgoCD?

- **Flux/ArgoCD** are GitOps tools that watch Git repos
- **Our approach** watches ConfigHub spaces directly
- **Benefit**: No Git intermediary, direct ConfigHub → Kubernetes comparison
- **Trade-off**: We implement the comparison logic ourselves

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  ConfigHub  │────▶│   Detector  │────▶│ Kubernetes  │
│    Units    │     │   (Go App)  │     │   Cluster   │
└─────────────┘     └──────┬──────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │   Claude    │
                    │  (Optional) │
                    └─────────────┘
```

## Running Locally

```bash
# Set environment variables
export KUBECONFIG=/path/to/kubeconfig
export CUB_TOKEN=your-confighub-token
export CLAUDE_API_KEY=your-claude-key  # Optional

# Build and run
go build
./drift-detector
```

## Running in Kubernetes

```bash
# Create namespace and secrets
kubectl create namespace devops-apps
kubectl create secret generic drift-detector-secrets \
  --from-literal=cub-token=$CUB_TOKEN \
  --from-literal=claude-api-key=$CLAUDE_API_KEY \
  -n devops-apps

# Deploy
kubectl apply -f k8s/deployment.yaml
```

## Configuration

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `NAMESPACE` | Kubernetes namespace to monitor | `qa` |
| `CUB_SPACE` | ConfigHub space to use as desired state | `acorn-bear-qa` |
| `CUB_API_URL` | ConfigHub API endpoint | `https://hub.confighub.com/api/v1` |
| `CUB_TOKEN` | ConfigHub API token | Required |
| `CLAUDE_API_KEY` | Claude API key for AI analysis | Optional |
| `AUTO_FIX` | Create fixes automatically | `false` |

## What It Detects

Currently detects drift in:
- Deployment replica counts
- Container images
- Resource requests/limits
- Service ports
- ConfigMap data

## Example Output

```
2025/09/21 15:50:35 Checking for drift...
2025/09/21 15:50:35 Found 2 units in ConfigHub space acorn-bear-qa
2025/09/21 15:50:35 Found 8 resources in Kubernetes namespace qa
2025/09/21 15:50:35 Detected 1 drift items
2025/09/21 15:50:35 === DRIFT REPORT ===
2025/09/21 15:50:35 Summary: Configuration drift detected
2025/09/21 15:50:35 Drift Items: 1
2025/09/21 15:50:35   - deployment/backend.replicas: expected=2, actual=1
```

## Future Enhancements

1. **Real ConfigHub API Integration**: Currently using mock data
2. **More Resource Types**: StatefulSets, DaemonSets, Ingresses
3. **Automatic Remediation**: Create and apply fixes via ConfigHub
4. **Metrics Export**: Prometheus metrics for drift tracking
5. **Webhook Mode**: Trigger on ConfigHub changes instead of polling

## Comparison with Cased

| Aspect | This Drift Detector | Cased Approach |
|--------|-------------------|----------------|
| **Architecture** | Persistent Go application | Ephemeral workflow |
| **State** | Maintains history | Stateless |
| **Customization** | Full control (it's our code) | Limited to their agents |
| **AI Integration** | Direct Claude API calls | Through their platform |
| **Deployment** | Standard K8s deployment | Workflow configuration |

## Why This Approach?

This is a **DevOps App**, not a workflow:
- Runs continuously (not triggered)
- Can maintain state and learn
- Deployed and managed like any application
- Full control over logic and integrations
- Can be versioned, rolled back, monitored

This demonstrates the fundamental difference between ConfigHub's app-based approach and workflow-based automation.