# Cost Impact Monitor

Real-time cost monitoring for all ConfigHub deployments with trigger-based pre/post deployment analysis.

## Example Use

The cost impact monitor has analyzed your Kind cluster resources:

### **Current Monthly Cost: $50.87**

### **Resource Breakdown:**
- **backend-api**: 5 replicas = $19.99/month
- **test-app**: 5 replicas = $21.34/month (⚠️ DRIFTED - should be 2)
- **complex-app**: 1 replica = $6.54/month (⚠️ DRIFTED - should be 3)
- **frontend-web**: 1 replica = $3.00/month

### **Drift Cost Impact:**
1. **test-app over-scaled**: +$12.80/month waste (running 5 instead of 2)
2. **complex-app under-scaled**: -$13.07/month (running 1 instead of 3)
3. **ConfigMap misconfigured**: Debug logging increases costs

### **ConfigHub Corrections Needed:**
```bash
# Fix test-app drift (save $12.80/month)
cub unit update deployment-test-app --patch --data '{"spec":{"replicas":2}}'

# Fix complex-app drift (ensure HA)
cub unit update deployment-complex-app --patch --data '{"spec":{"replicas":3}}'

# Fix ConfigMap drift
cub unit update configmap-app-config --patch --data '{"data":{"log_level":"info"}}'
```

### **Additional Savings Opportunities:**
- Reduce backend-api from 5→3 replicas: **Save $8.00/month**
- Right-size test-app after fixing drift: **Save $8.54/month**

### **Key Insights:**
1. **Total potential savings**: $21.34/month (42% reduction)
2. **Drift is costing**: $12.80/month in unnecessary replicas
3. **ConfigHub corrections** would automatically fix these issues

This demonstrates how the cost-impact-monitor provides **pre-deployment cost analysis** and helps maintain cost efficiency by detecting and correcting drift through ConfigHub unit updates!

## Overview

The Cost Impact Monitor is a **ConfigHub-deployed** DevOps app that continuously monitors all ConfigHub spaces for cost impacts. It provides:

- **Pre-deployment cost analysis**: Analyze cost impact before units are applied
- **Post-deployment verification**: Track actual vs predicted costs
- **Cross-space monitoring**: Monitor costs across all ConfigHub spaces
- **Trigger-based hooks**: Automatic cost warnings and risk assessments
- **Self-monitoring**: The monitor itself is deployed via ConfigHub

## Architecture

```
ConfigHub → Cost Monitor → Dashboard (:8083)
    ↓           ↓              ↓
  Units     Triggers      Real-time
            (Pre/Post)     Analysis
```

## Key Features

### 1. ConfigHub Self-Deployment
The monitor deploys itself through ConfigHub units, following the canonical pattern:
- Creates unique project prefix
- Sets up environment hierarchy (base → dev → staging → prod)
- Uses push-upgrade for promotions

### 2. Trigger System
- **Pre-Apply Hooks**: Warn about high-cost deployments before they happen
- **Post-Apply Hooks**: Track prediction accuracy and learn from actual usage
- **Change Detection**: Polls ConfigHub every 30 seconds for unit changes

### 3. Cost Analysis
- Analyzes all ConfigHub units for resource requirements
- Calculates monthly cost estimates
- Tracks pending changes and their cost impact
- Uses Claude AI for intelligent risk assessment

### 4. Web Dashboard
- Real-time cost visualization at `http://localhost:8083`
- Shows pending changes with risk levels
- Tracks deployment history and prediction accuracy
- Displays cost trends across all spaces

![Cost Monitoring Dashboard](cost%20monitoring%20dashboard.png)

## Installation

### Prerequisites
- Kubernetes cluster (Kind, EKS, GKE, etc.)
- ConfigHub CLI (`cub`) authenticated
- Go 1.21+ (for building)
- Claude API key (optional, for AI features)

### Deploy via ConfigHub

```bash
# 1. Create ConfigHub structure
bin/install-base      # Creates units in ConfigHub
bin/install-envs      # Creates env hierarchy

# 2. Deploy to Kubernetes
bin/apply-all dev     # Deploy to dev environment

# 3. Access dashboard
kubectl port-forward -n cost-monitoring svc/cost-impact-monitor 8083:8083
open http://localhost:8083
```

## How It Works

### Monitoring Flow

1. **Discovery**: Finds all ConfigHub spaces on startup
2. **Analysis**: Analyzes each space for current and projected costs
3. **Triggers**: Processes unit changes with pre/post hooks
4. **Dashboard**: Updates web UI with real-time data

### Cost Calculation

For each ConfigHub unit:
- Parses resource requirements from unit data
- Estimates monthly cost based on AWS pricing
- Tracks cost delta for pending changes
- Assesses risk level based on cost impact

### Trigger Processing

```go
// Pre-deployment trigger
func OnPreApply(unit *Unit) {
    if costDelta > $100 {
        CreateCostWarning(unit)
        UpdateDashboard()
    }
}

// Post-deployment trigger
func OnPostApply(unit *Unit) {
    actual := MeasureActualUsage(unit)
    UpdatePredictionModel(actual)
}
```

## Example Scenarios

### 📊 **[Platform Team Saves $2,400/month on Prometheus Upgrade](SCENARIO-HELM-FLUX.md)**
A detailed walkthrough of how a platform team using ConfigHub + Flux for GitOps prevented a massive cost overrun during a critical security upgrade. This real-world scenario shows the Cost Impact Monitor in action with Helm charts and demonstrates the value of pre-deployment cost analysis.

## Use Cases

### 1. Helm Chart Update Cost Preview
```bash
# Update Helm chart in ConfigHub
cub helm upgrade prometheus --version 55.0.0

# Cost monitor automatically detects and analyzes
# Dashboard shows: "⚠️ Prometheus upgrade: +$650/month"
```

### 2. Multi-Environment Cost Tracking
- Monitor costs across dev/staging/prod
- See cost impact before promoting changes
- Track cost trends over time

### 3. Automated Cost Warnings
- Automatic warnings for high-cost deployments
- Risk assessment with Claude AI
- Stored as ConfigHub units for audit trail

## Environment Variables

- `CUB_TOKEN`: ConfigHub API token (required)
- `CUB_API_URL`: ConfigHub API endpoint
- `CLAUDE_API_KEY`: Claude API key for AI features
- `AUTO_APPLY_OPTIMIZATIONS`: Enable automatic cost optimizations

## Dashboard Features

### Main Metrics
- Total monthly cost across all spaces
- Projected cost including pending changes
- Number of high-risk changes
- Prediction accuracy rate

### Pending Changes View
- All units awaiting deployment
- Cost delta for each change
- Risk level assessment
- Claude AI recommendations

### Space Monitoring
- Cost per ConfigHub space
- Cost trends (increasing/decreasing/stable)
- Number of pending changes per space

## Integration with Cost Optimizer

The Cost Impact Monitor complements the Cost Optimizer:
- **Monitor**: Tracks ConfigHub deployment costs
- **Optimizer**: Analyzes Kubernetes resource usage

Together they provide complete cost intelligence:
1. Monitor warns about high-cost ConfigHub changes
2. Optimizer suggests right-sizing after deployment
3. Both use ConfigHub for configuration management

## Competitive Advantages

### vs Traditional GitOps (Flux/Argo)
- **Preview without deployment**: Analyze costs before applying
- **Trigger-based analysis**: Automatic pre/post deployment hooks
- **Cross-space visibility**: Monitor all environments at once
- **Self-contained**: No external dependencies

### vs Cased.com Workflows
- **Persistent monitoring**: Continuous, not ephemeral
- **Event-driven**: React immediately to changes
- **Predictive**: Analyze before deployment
- **Learning system**: Improves predictions over time

## Development

### Building
```bash
go build -o cost-impact-monitor .
```

### Running Locally
```bash
export CUB_TOKEN="your-token"
export CLAUDE_API_KEY="your-key"
./cost-impact-monitor
```

### Testing
```bash
go test -v
```

## Troubleshooting

### Monitor not detecting changes
- Check ConfigHub authentication: `cub auth status`
- Verify spaces are accessible: `cub space list`
- Check logs: `kubectl logs -n cost-monitoring deploy/cost-impact-monitor`

### Dashboard not loading
- Verify port forward is active
- Check service is running: `kubectl get svc -n cost-monitoring`
- Check pod status: `kubectl get pods -n cost-monitoring`

## Future Enhancements

- [ ] Webhook support for instant triggers
- [ ] Cost budget alerts
- [ ] Multi-cloud pricing support
- [ ] Historical cost reports
- [ ] Integration with FinOps tools

## Contributing

This is part of the DevOps-as-Apps project. See the main repository for contribution guidelines.