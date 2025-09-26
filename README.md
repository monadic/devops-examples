# DevOps Examples - DevOps as Apps Platform

Production-ready DevOps automation applications built using the ConfigHub SDK. These are persistent Kubernetes applications, not ephemeral workflows.

## 🚀 Overview

This repository demonstrates the **DevOps as Apps** pattern - building DevOps automation as persistent, event-driven Kubernetes applications instead of ephemeral workflows (like agentic devops workflow tools).

### Key Principles

✅ **Persistent Applications** - Long-running apps, not one-shot workflows
✅ **Event-Driven** - Kubernetes informers, not polling
✅ **ConfigHub Native** - Uses Sets, Filters, and push-upgrade patterns
✅ **AI-Powered** - Claude enabled by default for intelligent analysis
✅ **Production Ready** - Health checks, metrics, proper error handling

### 🤖 Claude AI Integration (NEW)

All examples now include **Claude AI by default** with:
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

## 📦 Available Apps

**2 Production-Ready Apps** leveraging ConfigHub's full capabilities:

### 🔍 [Drift Detector](./drift-detector)
Continuously monitors for configuration drift and automatically fixes it.
- **Status**: ✅ Production Ready
- **Features**: Sets, Filters, Informers, Claude AI, Push-upgrade
- **Quick Start**:
  ```bash
  cd drift-detector
  ./bin/install
  ./bin/demo
  ```

### 💰 [Cost Optimizer](./cost-optimizer)
AI-powered Kubernetes cost optimization with real-time analysis and recommendations.
- **Status**: ✅ Production Ready
- **Features**: Claude AI analysis, OpenCost integration, Web dashboard (:8081), Auto-apply optimizations
- **Quick Start**:
  ```bash
  cd cost-optimizer
  ./bin/install-base
  ./run.sh  # Handles Claude API key setup
  ```
- **Use Case**: Reduce cloud costs by 30-50% with intelligent recommendations

### 🔐 Security Scanner *(Planned)*
Continuous security scanning and compliance checking.
- **Status**: 📋 Planned
- **Features**: CVE scanning, Policy enforcement, Compliance reports

### ⬆️ Upgrade Manager *(Planned)*
Manages application upgrades across environments.
- **Status**: 📋 Planned
- **Features**: Blue-green, Canary, Rolling updates

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

## 🏗️ Architecture

These apps use:
- **ConfigHub SDK** for configuration management
- **Kubernetes Informers** for event-driven monitoring
- **Claude AI** for intelligent analysis
- **Push-upgrade pattern** for change propagation

### ConfigHub Features Demonstrated

✅ **Version Management** - `cub run set-image-reference` for tracking versions
✅ **Environment Hierarchy** - base → dev → staging → prod with upstream/downstream
✅ **Rollback Capability** - Leverage revision history for safe rollbacks
✅ **Sets & Filters** - Group and target configurations for bulk operations
✅ **Push-Upgrade** - Automatic change propagation across environments
✅ **Self-Deployment** - Apps deploy themselves through ConfigHub units

## 🆚 Why Not Agentic DevOps Workflow Tools?

| Feature | DevOps as Apps | Agentic Workflow Tools |
|---------|---------------|-----------|
| **Model** | Persistent applications | Ephemeral workflows |
| **Architecture** | Event-driven (informers) | Triggered execution |
| **State** | Stateful, can learn | Stateless |
| **Customization** | Full source control | Limited to their DSL |
| **Cost** | Open source + ConfigHub | Per-workflow pricing |

## 📚 Documentation

- [DevOps as Apps Architecture](https://github.com/monadic/devops-as-apps-project)
- [ConfigHub SDK](https://github.com/monadic/devops-sdk)
- [ConfigHub Docs](https://docs.confighub.com)

## 🤝 Contributing

1. Use only real ConfigHub APIs
2. Follow the 2-step testing protocol
3. Use informers, not polling
4. Include comprehensive documentation
5. Add install/test/demo scripts

## 📄 License

Proprietary - ConfigHub, Inc.