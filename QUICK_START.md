# Quick Start Guide - DevOps as Apps

Get everything running in 5 minutes!

## Prerequisites Check

```bash
# Run this first to check you have everything
./check-prerequisites.sh
```

If anything is missing:
- **Go**: `brew install go` (needs 1.21+)
- **kubectl**: `brew install kubectl`
- **kind**: `brew install kind`
- **ConfigHub CLI**: Get from https://confighub.com

## Step 1: Environment Setup

```bash
# ConfigHub authentication (REQUIRED)
export CUB_TOKEN="your-token-here"
cub auth whoami  # Should show your username

# Claude API (OPTIONAL but recommended)
export CLAUDE_API_KEY="sk-ant-..."

# Create test Kubernetes cluster
kind create cluster --name devops-test
kubectl create namespace drift-test
```

## Step 2: Build Everything

```bash
# One command to build all apps
./build-all.sh

# You should see:
# ✅ drift-detector built
# ✅ cost-optimizer built
# ✅ live-dashboard built
```

## Step 3: Start the Dashboard

```bash
# Start the unified dashboard
cd cost-impact-monitor
./live-dashboard &

# Open in browser
open http://localhost:8082
```

## Step 4: Verify Everything Works

```bash
# Run health check
curl http://localhost:8082/api/health | jq '.health_score'
# Should see: 95-100

# Check dashboard
open http://localhost:8082
# Should see real-time monitoring data
```

## Step 5: Test Drift Detection

```bash
# Create some drift using ConfigHub
cub unit update backend-api-unit --space drift-test-demo \
  --patch --data '{"spec":{"replicas":5}}'

# Watch the dashboard - drift should appear!
```

## What You Should See

### Dashboard (http://localhost:8082)
- **Cluster Info**: Your Kind cluster details
- **ConfigHub Info**: Connected spaces and units
- **Cost Metrics**: Current costs and drift impact
- **Resources**: List of deployments with status
- **Corrections**: ConfigHub commands to fix drift
- **Health Check Button**: Click to run comprehensive health check

### Health Check Results
When you click "Run Health Check":
- ConfigHub: ✅ HEALTHY
- Kubernetes: ✅ HEALTHY
- Deployments: ✅ HEALTHY
- Drift Detection: ⚠️ DRIFTED (if drift exists)
- APIs: ✅ ONLINE

## Common Issues & Fixes

### "Cannot connect to ConfigHub"
```bash
# Check your token
echo $CUB_TOKEN
cub auth whoami

# Re-authenticate if needed
cub auth login
```

### "Dashboard not loading"
```bash
# Check if running
ps aux | grep live-dashboard

# Restart if needed
pkill -f live-dashboard
cd cost-impact-monitor && ./live-dashboard &
```

### "No drift detected"
```bash
# Verify units exist
cub unit list --space drift-test-demo

# Check deployments
kubectl get deployments -n drift-test
```

## Next Steps

1. **Explore the Apps**:
   - Cost Optimizer: `cd cost-optimizer && ./cost-optimizer`
   - Drift Detector: `cd drift-detector && ./drift-detector`

2. **Run Compliance Tests**:
   ```bash
   ./test-app-compliance-quick.sh drift-detector
   ```

3. **Check Health Status**:
   ```bash
   ./devops-app-health-check.sh
   ```

## Clean Up

```bash
# Stop all apps
./stop-all.sh

# Delete Kind cluster
kind delete cluster --name devops-test

# Clean build artifacts
./clean-all.sh
```

## Get Help

- **Logs**: Check `logs/` directory in each app
- **Debug**: Set `DEBUG=true` before running
- **Issues**: https://github.com/monadic/devops-examples/issues

## 🎉 Success!

You now have:
- ✅ Live dashboard monitoring your infrastructure
- ✅ Drift detection running continuously
- ✅ Cost analysis with AI recommendations
- ✅ Health checks available on demand
- ✅ All using ConfigHub as source of truth (no kubectl!)

Visit **http://localhost:8082** to see everything in action!