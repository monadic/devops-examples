# DevOps Examples

Production-ready DevOps automation applications built with ConfigHub. These are persistent Kubernetes applications following the **DevOps as Apps** pattern - continuous, event-driven apps instead of ephemeral workflows.

## 📦 Available Examples

### 1. [Drift Detector](./drift-detector)
- Event-driven Kubernetes configuration drift detection
- Uses ConfigHub Sets, Filters, and bulk operations
- Auto-corrects drift by updating existing units (not creating new "-fix" units)
- Real-time dashboard on :8080
- Claude AI integration for drift analysis
- Full ConfigHub deployment pattern with push-upgrade

### 2. [Cost Optimizer](./cost-optimizer)
- AI-powered cost optimization with Claude
- NEW: OpenCost integration for real cloud cost data (vs estimates)
- Web dashboard on :8081 with Claude API history viewer
- Metrics-server integration for real resource usage
- Uses Sets for grouping recommendations
- Push-upgrade for promoting optimizations across environments

### 3. [Cost Impact Monitor](./cost-impact-monitor)
- Pre-deployment cost analysis before units are applied
- Monitors all ConfigHub spaces for cost impact
- Trigger-based hooks (pre/post deployment)
- Web dashboard on :8083
- Self-deploys through ConfigHub
- Complements Cost Optimizer (monitor = pre-deployment, optimizer = post-deployment)

## 🚀 Quick Start

Each example has complete setup instructions in its own README:

```bash
# Example: Drift Detector
cd drift-detector
cat README.md  # Full setup guide

# Or use the QUICKSTART.md for fast setup
cat QUICKSTART.md
```

## 📋 Prerequisites

- **ConfigHub account** - [Sign up](https://confighub.com)
- **ConfigHub CLI** - `brew install confighubai/tap/cub`
- **Kubernetes cluster** - Kind, Minikube, or cloud provider
- **Claude API key** - [Get one](https://console.anthropic.com/settings/keys)
- **Go 1.21+** - For building from source

## 📚 Learn More

- **[DevOps as Apps Architecture](https://github.com/monadic/devops-as-apps-project)** - Full explanation of the pattern
- **[Canonical Patterns](https://github.com/monadic/devops-as-apps-project/blob/main/CANONICAL-PATTERNS-SUMMARY.md)** - ConfigHub best practices
- **[ConfigHub SDK](https://github.com/monadic/devops-sdk)** - Reusable library used by all examples

## 🏗️ Common Pattern

All examples follow the same structure:
- Deploy via ConfigHub (not kubectl)
- Environment hierarchy: base → dev → staging → prod
- Event-driven with Kubernetes informers
- Claude AI integration
- Push-upgrade for promotions
- ConfigHub Sets and Filters for bulk operations

See each example's README for detailed architecture and deployment instructions.

## 📄 License

Proprietary - ConfigHub, Inc.
