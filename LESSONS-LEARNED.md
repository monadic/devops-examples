# Lessons Learned - DevOps as Apps Implementation

## Date: 2025-09-26

### Key Discoveries from Today's Session

## 1. ConfigHub Worker is MANDATORY
**Discovery**: ConfigHub units cannot be applied to Kubernetes without a running worker.

**Why This Matters**:
- Worker is the bridge between ConfigHub (control plane) and Kubernetes (data plane)
- Without worker: Units exist in ConfigHub but never deploy to clusters
- With worker: Real-time sync between desired and actual state

**Implementation**:
```bash
# Create worker in correct space
cub worker create devops-worker --space drift-test-demo

# Run worker (keep running in separate terminal)
cub worker run devops-worker

# Create target linking to cluster
cub target create drift-test-target \
  '{"KubeContext":"kind-devops-test","KubeNamespace":"drift-test"}' \
  devops-worker --space drift-test-demo
```

## 2. Global-App Pattern is the Standard
**Discovery**: All examples must follow the global-app canonical patterns.

**Key Patterns**:
1. **Unique prefix generation**: `cub space new-prefix`
2. **Space hierarchy**: base → dev → staging → prod
3. **Filters for targeting**: WHERE clauses for bulk operations
4. **Upstream/downstream**: Inheritance via UpstreamUnitID
5. **Push-upgrade**: Propagation with BulkPatchUnits(Upgrade: true)

**Example Structure**:
```bash
project=$(cub space new-prefix)  # e.g., "fluffy-paws"
cub space create $project --label project=$project
cub space create $project-base --label base=true
cub space create $project-filters --label type=filters
```

## 3. Unit Data Must Be Valid Kubernetes YAML
**Discovery**: ConfigHub units must contain properly formatted Kubernetes manifests.

**Common Issues**:
- Corrupted base64 encoding
- Missing required Kubernetes fields
- Invalid YAML structure

**Solution**:
```bash
# Update unit with valid YAML from file
cat manifest.yaml | cub unit update unit-name --space space-name --from-stdin

# Or create new unit
cub unit create unit-name manifest.yaml --space space-name \
  --label type=app --label tier=critical
```

## 4. Targets Link Units to Clusters
**Discovery**: Each unit needs a target to know where to deploy.

**Requirements**:
1. Worker must be running in the same space
2. Target must specify KubeContext and namespace
3. Units must have TargetID set

**Setting Targets**:
```bash
# Create target
cub target create target-name '{"KubeContext":"..","KubeNamespace":".."}' worker-name

# Assign to unit
cub unit set-target unit-name target-name --space space-name
```

## 5. Authentication Flow
**Discovery**: Multiple auth tokens needed for full functionality.

**Required Credentials**:
1. **ConfigHub**: `cub auth login` → generates token
2. **Claude AI**: API key from console.anthropic.com
3. **Kubernetes**: kubeconfig for cluster access

**Environment Setup**:
```bash
export CUB_TOKEN="$(cub auth get-token)"
export CLAUDE_API_KEY="sk-ant-..."
export KUBECONFIG="$HOME/.kube/config"
```

## 6. Event-Driven > Polling
**Discovery**: Kubernetes informers provide instant updates vs periodic polling.

**Benefits**:
- Immediate reaction to changes
- Lower resource usage
- No missed events
- Better for persistent apps

**Implementation**:
```go
app.RunWithInformers(func() error {
    // React to changes immediately
    return detectAndFixDrift()
})
```

## 7. Demo Mode is Essential
**Discovery**: Demo mode allows testing without real infrastructure.

**Benefits**:
- Quick validation of logic
- No cloud costs
- Reproducible scenarios
- Good for CI/CD

**Pattern**:
```go
if len(os.Args) > 1 && os.Args[1] == "demo" {
    return RunDemo()
}
```

## Action Items Completed

✅ Created `setup-worker.sh` script for easy worker setup
✅ Created `WORKER-SETUP.md` comprehensive documentation
✅ Updated README with worker requirements
✅ Created proper YAML manifests in `/manifests/`
✅ Fixed ConfigHub units with valid Kubernetes YAML
✅ Established worker bridge for ConfigHub ↔ Kubernetes

## Best Practices Moving Forward

1. **Always start with worker**: No point testing without the bridge
2. **Use global-app patterns**: Consistency across all examples
3. **Validate YAML first**: Test manifests with `kubectl apply --dry-run`
4. **Document prerequisites**: Especially authentication requirements
5. **Provide demo mode**: Allow testing without full infrastructure
6. **Use Sets and Filters**: For bulk operations across environments
7. **Implement health checks**: For production readiness

## What Worked Well

✅ ConfigHub authentication via `cub auth login`
✅ Claude AI integration for intelligent analysis
✅ Cost dashboard visualization at :8081
✅ Event-driven architecture with informers
✅ Demo mode for quick testing

## What Needed Fixing

❌ Missing worker setup → Created setup script
❌ Invalid unit YAML → Fixed with proper manifests
❌ No target configuration → Added target creation
❌ Unclear prerequisites → Updated documentation

## Future Improvements

1. **Automated worker management**: Start/stop with apps
2. **Multi-cluster support**: Workers per environment
3. **GitOps integration**: ConfigHub → Git → Flux/Argo
4. **Metrics collection**: Prometheus/Grafana dashboards
5. **Policy enforcement**: OPA integration