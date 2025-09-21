# Cost Optimizer

A Kubernetes cost optimization application that analyzes resource usage and recommends cost-saving opportunities.

## How It Works

This is a **persistent DevOps application** (like global-app), not a workflow:

1. **Continuous Monitoring**: Runs every hour to analyze costs
2. **Resource Collection**: Gathers data on all deployments, statefulsets, and storage
3. **Metrics Analysis**: Compares actual usage (from metrics-server) with requested resources
4. **Claude AI Analysis**: Uses Claude to identify optimization opportunities
5. **Recommendations**: Provides specific actions to reduce costs
6. **ConfigHub Integration**: Can create optimization spaces for gradual rollout

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Kubernetes  │────▶│   Cost      │────▶│   Claude    │
│   Cluster   │     │ Optimizer   │     │     API     │
└─────────────┘     └──────┬──────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  ConfigHub  │
                    │   (Fixes)   │
                    └─────────────┘
```

## Features

### Resource Analysis
- **CPU**: Compares requested vs actual usage
- **Memory**: Identifies over-provisioned memory
- **Storage**: Finds unused or oversized volumes
- **Replicas**: Suggests optimal replica counts

### Cost Calculation
- Simple pricing model ($25/vCPU, $3/GB RAM, $0.10/GB storage)
- Configurable cloud provider pricing
- Monthly cost projections

### Optimization Types
1. **Rightsizing**: Reduce resource requests to match actual usage
2. **Scaling**: Optimize replica counts
3. **Removal**: Identify idle resources
4. **Reserved Instances**: Suggest commitment discounts

## Running Locally

```bash
# Set environment variables
export KUBECONFIG=/path/to/kubeconfig
export CLAUDE_API_KEY=your-claude-key
export NAMESPACE=qa  # Namespace to analyze

# Build and run
go build
./cost-optimizer
```

## Sample Output

```
2025/09/21 17:00:00 Cost optimizer started
2025/09/21 17:00:00 Analyzing costs...
2025/09/21 17:00:00 Collected data for 15 resources
2025/09/21 17:00:00 Current monthly cost: $1,234.56
2025/09/21 17:00:01 === COST OPTIMIZATION REPORT ===
2025/09/21 17:00:01 Current Monthly Cost: $1,234.56
2025/09/21 17:00:01 Potential Savings: $456.78 (37.0%)
2025/09/21 17:00:01 Found 5 optimization opportunities:
2025/09/21 17:00:01   backend (rightsize):
2025/09/21 17:00:01     Savings: $125.00/month
2025/09/21 17:00:01     Risk: low
2025/09/21 17:00:01     Action: Reduce CPU from 1000m to 400m based on usage
2025/09/21 17:00:01   frontend (scale):
2025/09/21 17:00:01     Savings: $85.50/month
2025/09/21 17:00:01     Risk: medium
2025/09/21 17:00:01     Action: Reduce replicas from 5 to 3
```

## Deployment to Kubernetes

```bash
# Create namespace and secrets
kubectl create namespace devops-apps
kubectl create secret generic cost-optimizer-secrets \
  --from-literal=cub-token=$CUB_TOKEN \
  --from-literal=claude-api-key=$CLAUDE_API_KEY \
  -n devops-apps

# Deploy
kubectl apply -f k8s/deployment.yaml

# Check logs
kubectl logs -f deployment/cost-optimizer -n devops-apps
```

## Configuration

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `NAMESPACE` | Namespace to analyze | `default` |
| `CUB_SPACE` | ConfigHub space for optimizations | `acorn-bear-qa` |
| `CLAUDE_API_KEY` | Claude API key for AI analysis | Required |
| `AUTO_OPTIMIZE` | Automatically apply optimizations | `false` |

## How This Differs from Workflows

### This Cost Optimizer (DevOps App)
- **Persistent**: Runs continuously as a Deployment
- **Stateful**: Tracks costs over time
- **Learning**: Can improve recommendations based on history
- **Integrated**: Direct Kubernetes API access
- **Versioned**: Can be rolled back like any app

### Workflow Approach (e.g., Cased)
- **Triggered**: Runs on schedule or event
- **Stateless**: No memory between runs
- **Simple**: Pre-built analysis
- **Limited**: Can't access metrics directly
- **Not versioned**: Workflow config, not an app

## Integration with ConfigHub

When optimizations are found, the app can:

1. Create a new ConfigHub space (e.g., `qa-cost-opt-123456`)
2. Apply recommended changes to units in that space
3. Test in isolation before promoting
4. Gradually roll out using ConfigHub's promotion model

Example:
```
acorn-bear-qa (current)
    └── acorn-bear-qa-cost-opt-123456 (optimization)
            ├── backend (reduced CPU/memory)
            ├── frontend (reduced replicas)
            └── postgres (optimized storage)
```

## Claude Integration

Claude analyzes the resource data and provides:
- Intelligent recommendations beyond simple heuristics
- Risk assessment for each optimization
- Explanation of why each change makes sense
- Consideration of workload patterns

The Claude prompt includes:
- Current resource allocations
- Actual usage metrics
- Cost calculations
- Optimization goals

## Future Enhancements

1. **Historical Analysis**: Track usage patterns over time
2. **Predictive Scaling**: Anticipate load changes
3. **Multi-Cloud**: Support AWS, GCP, Azure pricing
4. **Spot Instances**: Recommend spot/preemptible instances
5. **Network Costs**: Analyze data transfer costs
6. **Automated Rollout**: Gradual application of optimizations
7. **Prometheus Integration**: Richer metrics
8. **Slack Notifications**: Alert on savings opportunities

## Why This Architecture?

This demonstrates that cost optimization is not a one-time script or workflow, but a **continuous application** that:
- Monitors constantly
- Learns from patterns
- Integrates deeply with infrastructure
- Can be managed like any other application

Just like global-app serves business logic, cost-optimizer serves DevOps optimization logic.