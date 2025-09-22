# Cost Optimizer

AI-powered Kubernetes cost optimization using ConfigHub and our enhanced DevOps SDK.

## Overview

The Cost Optimizer is a DevOps application that:
- **Analyzes** Kubernetes resource usage across your cluster
- **Uses Claude AI** to generate intelligent cost optimization recommendations
- **Stores analysis** in ConfigHub for tracking and collaboration
- **Provides a web dashboard** for visualization and monitoring
- **Follows the global-app pattern** for ConfigHub-driven deployment

## Architecture

### Built with Enhanced DevOps SDK
- **Event-driven processing** using `RunWithInformers()`
- **Comprehensive Claude logging** with timestamped request/response tracking
- **Real ConfigHub integration** with space/set/filter management
- **High-level convenience helpers** for common operations

### ConfigHub Integration
- **Unique space naming** using `cub space new-prefix`
- **Sets for grouping** critical cost items
- **Filters for querying** high-cost resources and recommendations
- **Push-upgrade pattern** for promoting optimizations across environments

### AI-Powered Analysis
- **Claude AI integration** for intelligent recommendations
- **Risk assessment** for each suggested change
- **ConfigHub action mapping** for implementation
- **Automatic application** of low-risk optimizations (optional)

## Quick Start

### Demo Mode
```bash
# See the cost optimizer in action with mock data
./cost-optimizer demo
```

### Real Deployment with ConfigHub

1. **Setup ConfigHub credentials**:
```bash
export CUB_TOKEN="your-confighub-token"
export CUB_API_URL="https://api.confighub.com/v1"
```

2. **Create ConfigHub structure**:
```bash
bin/install-base          # Create base configuration
bin/install-envs          # Set up dev → staging → prod hierarchy
```

3. **Deploy to Kubernetes**:
```bash
bin/apply-all dev         # Deploy to dev environment
```

4. **Access the dashboard**:
```bash
kubectl port-forward svc/cost-optimizer-dashboard 8081:8081 -n devops-apps
# Visit: http://localhost:8081
```

## Features

### 🔍 Cost Analysis
- **Resource utilization analysis** across all deployments
- **Monthly cost estimation** based on CPU, memory, storage usage
- **Utilization thresholds** to identify over-provisioned resources
- **Cluster-wide summary** statistics

### 🤖 AI Recommendations
- **Claude AI-powered analysis** with intelligent suggestions
- **Risk-based categorization** (low/medium/high)
- **Priority scoring** for maximum impact
- **Implementation guidance** for each recommendation

### 📊 Web Dashboard
- **Real-time cost visualization** with auto-refresh
- **Interactive recommendations** with savings estimates
- **Resource breakdown** by compute, memory, storage, network
- **Cluster health metrics** and utilization trends

### ⚙️ ConfigHub Management
- **Automatic space creation** with unique prefixes
- **Cost analysis storage** for historical tracking
- **High-priority recommendations** in dedicated Sets
- **Filter-based querying** for targeted operations

## Configuration

### Environment Variables
```bash
# Claude AI (optional - falls back to basic analysis)
CLAUDE_API_KEY="your-claude-api-key"
CLAUDE_DEBUG_LOG="true"              # Enable full request/response logging

# ConfigHub (optional - runs in local mode without)
CUB_TOKEN="your-confighub-token"
CUB_API_URL="https://api.confighub.com/v1"

# Cost Optimizer Settings
AUTO_APPLY_OPTIMIZATIONS="false"     # Auto-apply low-risk changes
NAMESPACE="devops-apps"              # Target namespace
```

### Cost Calculation Settings
The optimizer uses realistic cloud pricing:
- **CPU**: $0.0416 per vCPU hour
- **Memory**: $0.00456 per GB hour
- **Storage**: $0.10 per GB month
- **Network**: $0.09 per GB transfer

## Deployment Patterns

### Development
```bash
# Quick local development
export CLAUDE_DEBUG_LOG=true
./cost-optimizer
```

### ConfigHub-Driven (Recommended)
```bash
# Full ConfigHub deployment following global-app pattern
bin/install-base
bin/install-envs
bin/apply-all dev

# Promote through environments
bin/promote dev staging
bin/apply-all staging

bin/promote staging prod
bin/apply-all prod
```

### Environment Variants
```bash
# Create analysis environment for testing optimizations
bin/install-envs --with-analysis-envs

# This creates optimized variants with reduced resources
# for testing cost savings before applying to production
```

## API Endpoints

### Health & Monitoring
- **Health**: `:8080/health` - Application health status
- **Metrics**: `:8080/metrics` - Prometheus metrics (if enabled)

### Dashboard & Data
- **Dashboard**: `:8081/` - Interactive web dashboard
- **Analysis API**: `:8081/api/analysis` - Full cost analysis JSON
- **Recommendations**: `:8081/api/recommendations` - Just recommendations JSON

## ConfigHub Structure

### Spaces Created
```
{prefix}-cost-optimizer           # Main space
├── {prefix}-cost-optimizer-base     # Base configurations
├── {prefix}-cost-optimizer-dev      # Dev environment
├── {prefix}-cost-optimizer-staging  # Staging environment
├── {prefix}-cost-optimizer-prod     # Production environment
└── {prefix}-cost-optimizer-filters  # Filters for querying
```

### Sets for Organization
- **critical-costs**: High-priority cost items (>$50/month savings)
- **cost-recommendations**: All optimization suggestions
- **cost-analysis-history**: Historical analysis data

### Filters for Querying
- **all**: All project units
- **high-cost**: Resources >$100/month
- **critical-recommendations**: High priority recommendations
- **low-utilization**: Resources <50% CPU and memory utilization

## Example Cost Analysis

```json
{
  "timestamp": "2024-01-15T14:30:00Z",
  "total_monthly_cost": 1245.67,
  "potential_savings": 287.45,
  "savings_percentage": 23.1,
  "recommendations": [
    {
      "resource": "deployment/frontend-web",
      "namespace": "production",
      "type": "rightsize",
      "priority": "high",
      "monthly_savings": 73.65,
      "risk": "low",
      "explanation": "Only using 30% of allocated CPU and memory",
      "confighub_action": "Update deployment unit with new resource limits"
    }
  ],
  "cluster_summary": {
    "total_nodes": 3,
    "total_pods": 24,
    "avg_cpu_utilization": 42.5,
    "avg_memory_utilization": 38.7
  }
}
```

## Integration Examples

### With CI/CD
```yaml
# In your CI pipeline
- name: Cost Analysis
  run: |
    ./cost-optimizer demo > cost-analysis.json
    # Upload to artifact storage or send to Slack
```

### With Monitoring
```yaml
# Prometheus scraping config
- job_name: 'cost-optimizer'
  static_configs:
  - targets: ['cost-optimizer-health.devops-apps.svc.cluster.local:8080']
```

---

**Built with the enhanced DevOps SDK** • **Follows the global-app pattern** • **AI-powered by Claude**