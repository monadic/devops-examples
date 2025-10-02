# DevOps Examples - DevOps as Apps Platform

Production-ready DevOps automation applications built using the ConfigHub SDK. These are persistent Kubernetes applications, not ephemeral workflows.

## 🚀 Overview

This repository demonstrates the **DevOps as Apps** pattern - building DevOps automation as persistent, event-driven Kubernetes applications instead of ephemeral workflows (like agentic devops workflow tools).

1. Drift Detector

  - Event-driven Kubernetes configuration drift detection
  - Uses ConfigHub Sets, Filters, and bulk operations
  - Auto-corrects drift by updating existing units (not creating new "-fix" units)
  - Real-time dashboard on :8080
  - Claude AI integration for drift analysis
  - Full ConfigHub deployment pattern with push-upgrade

  2. Cost Optimizer

  - AI-powered cost optimization with Claude
  - NEW: OpenCost integration for real cloud cost data (vs estimates)
  - Web dashboard on :8081 with Claude API history viewer
  - Metrics-server integration for real resource usage
  - Uses Sets for grouping recommendations
  - Push-upgrade for promoting optimizations across environments

  3. Cost Impact Monitor

  - Pre-deployment cost analysis before units are applied
  - Monitors all ConfigHub spaces for cost impact
  - Trigger-based hooks (pre/post deployment)
  - Web dashboard on :8083
  - Self-deploys through ConfigHub
  - Complements Cost Optimizer (monitor = pre-deployment, optimizer = post-deployment)


### Key Principles

✅ **Persistent Applications** - Long-running apps, not one-shot workflows
✅ **Event-Driven** - Kubernetes informers, not polling
✅ **ConfigHub Native** - Uses Sets, Filters, and push-upgrade patterns
✅ **AI-Powered** - Claude enabled by default for intelligent analysis
✅ **Production Ready** - Health checks, metrics, proper error handling

Examples now include **Claude AI by default** with:
- **Automatic setup** - Prompts for API key if not provided
- **Debug logging** - See all prompts and responses
- **Easy disable** - `ENABLE_CLAUDE=false ./run.sh`
- **Fallback mode** - Works without Claude using basic analysis

```bash
# Quick setup
cp .env.example .env
# Add your CLAUDE_API_KEY to .env
cd any-example/
./run.sh  # Handles everything automatically
```


## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- ConfigHub account ([sign up](https://confighub.com))
- Kubernetes cluster (or [Kind](https://kind.sigs.k8s.io/) for testing)
- Claude API key ([get one](https://console.anthropic.com/settings/keys)) - **Required by default**
- **ConfigHub Worker** - Required for ConfigHub ↔ Kubernetes bridge (see [WORKER-SETUP.md](WORKER-SETUP.md))

### Installation

1. **Clone and authenticate:**
   ```bash
   git clone https://github.com/monadic/devops-examples.git
   cd devops-examples
   cub auth login
   ```

2. **Setup ConfigHub Worker (REQUIRED):**
   ```bash
   ./setup-worker.sh      # Creates worker and targets
   cub worker run devops-worker  # Run in separate terminal
   ```

3. **Try the drift detector:**
   ```bash
   cd drift-detector
   ./bin/install-base     # Set up ConfigHub resources (global-app pattern)
   ./bin/install-envs     # Create environment hierarchy
   ./drift-detector demo  # Run interactive demo
   ```

3. **Test with real cluster:**
   ```bash
   ./bin/create-cluster     # Create Kind cluster
   ./bin/deploy-test        # Deploy test workloads
   ./drift-detector         # Run drift detection
   ```

## 🧪 Testing Protocol

All apps follow a standardized 2-step testing protocol:

### Step 1: Local Tests (Fast)
```bash
cd app-name
go test -v           # Unit tests with mocks
./app-name demo      # Demo mode
```

### Step 2: Integration Tests (Real Services)
```bash
export CUB_TOKEN="$(cub auth get-token)"
./bin/create-cluster                    # Create Kind cluster
go test -tags=integration -v            # Test with real APIs
./app-name                              # Run against cluster
```


## 📚 Documentation

- [DevOps as Apps Architecture](https://github.com/monadic/devops-as-apps-project)
- [ConfigHub SDK](https://github.com/monadic/devops-sdk)
- [ConfigHub Docs](https://docs.confighub.com)

## 📄 License

Proprietary - ConfigHub, Inc.
