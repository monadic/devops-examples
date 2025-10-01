# CRITICAL: ConfigHub Worker Image Bug

**Date**: 2025-10-01
**Severity**: CRITICAL - All new workers fail to start
**Status**: Broken image pushed to `:latest` tag today between 15:20-17:50 UTC

---

## Problem

**All new ConfigHub workers fail with 404 error on startup**, preventing the entire ConfigHub → Worker → Kubernetes deployment workflow.

## Root Cause

**Broken worker image pushed to `:latest` tag today.**

### Working Image (before ~15:20 UTC)
```
ghcr.io/confighubai/confighub-worker@sha256:8b96908dd57fd906ce9168aa9c283b3285029fc86adcf7a2079d622650ca02b5
```

### Broken Image (current :latest)
```
ghcr.io/confighubai/confighub-worker@sha256:c4338b51eaa8896207b02c28dca18bb280e548dc58e82190da453881994d46dc
```

## Error

Worker pod crashes immediately with:
```
2025/10/01 17:50:32 [ERROR] Failed to get bridge worker slug: server returned status 404: 404 Not Found
2025/10/01 17:50:32 Error starting worker: failed to get bridge worker slug: server returned status 404: 404 Not Found
```

**However**: Worker record EXISTS in ConfigHub API and can be queried successfully via CLI.

## Reproduction (30 seconds)

```bash
# 1. Create worker
cub worker create test-worker --space any-space

# 2. Install to Kubernetes
cub worker install test-worker --namespace confighub --space any-space --export > worker.yaml
kubectl apply -f worker.yaml

# 3. Check logs - will show 404 error
kubectl logs -n confighub -l app=test-worker
```

## Evidence

### Timeline from Our Deployment
```
15:19 UTC → Worker created → Image: 8b96908d... → ✅ WORKS
15:20-17:49 UTC → [NEW IMAGE PUSHED TO :latest]
17:50 UTC → Worker created → Image: c4338b51... → ❌ FAILS (404)
17:57 UTC → Worker created → Image: c4338b51... → ❌ FAILS (404)
```

### Image Hashes Verified
```bash
# Working worker pod (created 15:19)
$ kubectl get pod devops-test-worker-xxx -n confighub -o jsonpath='{.status.containerStatuses[0].imageID}'
ghcr.io/confighubai/confighub-worker@sha256:8b96908dd57fd906ce9168aa9c283b3285029fc86adcf7a2079d622650ca02b5

# Broken worker pod (created 17:50)
$ kubectl get pod cost-optimizer-worker-xxx -n confighub -o jsonpath='{.status.containerStatuses[0].imageID}'
ghcr.io/confighubai/confighub-worker@sha256:c4338b51eaa8896207b02c28dca18bb280e548dc58e82190da453881994d46dc
```

### Worker Exists in API
```bash
$ cub worker get cost-optimizer-worker --space sunrise-cub-cost-optimizer-base --json
{
  "BridgeWorkerID": "82bde808-ce33-4e54-9646-0ff2d0ddf97c",
  "Slug": "cost-optimizer-worker",
  "Condition": "Disconnected",
  "LastSeenAt": "0001-01-01T00:00:00Z"  ← Never connected due to 404 error
}
```

CLI can fetch worker metadata, but worker pod itself gets 404 when fetching the same data during initialization.

## Impact

- ❌ All new workers fail to start (100% failure rate)
- ❌ Cannot deploy units via `cub unit apply`
- ❌ Cannot set targets for units
- ❌ Complete ConfigHub deployment workflow broken for anyone creating new workers after ~15:20 UTC today
- ✅ Existing workers (created before 15:20 UTC) continue to work normally

## Workaround

Pin deployments to working image hash:

```bash
# Generate manifest
cub worker install my-worker --namespace confighub --space my-space --export > worker.yaml

# Edit worker.yaml - change this line:
# FROM: image: ghcr.io/confighubai/confighub-worker:latest
# TO:   image: ghcr.io/confighubai/confighub-worker@sha256:8b96908dd57fd906ce9168aa9c283b3285029fc86adcf7a2079d622650ca02b5

# Apply
kubectl apply -f worker.yaml
```

## Action Required

### Immediate (to unblock users)
1. Revert `:latest` tag to working image: `sha256:8b96908dd57fd906ce9168aa9c283b3285029fc86adcf7a2079d622650ca02b5`

### For Fix
2. Debug why worker code in `c4338b51...` fails to fetch its own slug from API
3. The worker is calling an API endpoint to "get bridge worker slug" that returns 404
4. Same worker can be successfully queried via CLI using `cub worker get`
5. This suggests an API endpoint mismatch or authentication issue in the worker code

### Before Next Deploy
6. Test new worker images in staging before pushing to `:latest`
7. Verify worker can successfully start and connect to ConfigHub

## Additional Details

### Environment
- **Kubernetes**: Kind cluster (local testing)
- **ConfigHub API**: https://hub.confighub.com/api
- **Namespace**: confighub
- **Affected Spaces**: All (tested multiple)

### Configuration Comparison
Both working and broken workers use:
- Same Kubernetes cluster
- Same ConfigHub API endpoint
- Same authentication secret (confighub-worker-env)
- Same RBAC (confighub-worker ServiceAccount)
- Same deployment configuration

**Only difference**: Container image hash

### Test Case
We even created a NEW worker in the SAME space as a working worker, and it failed:
```bash
# Working space: cozy-cub-drift-detector-base (created today)
# Working worker: devops-test-worker (created 15:19) → ✅ Works
# New test worker: test-worker-2 (created 17:57 in SAME space) → ❌ Fails
```

This proves it's not a space-specific issue, configuration issue, or environment issue. It's purely the worker image code.

## Live Example

**ConfigHub space with units but broken worker:**
https://hub.confighub.com/space/743a8f42-8c32-4771-ac65-cdbe75252397

Space contains:
- 3 units (nginx-web, redis-cache, postgres-db) - STATUS=NoLive
- 1 worker (cost-optimizer-worker) - Condition=Disconnected
- 0 targets (worker can't connect to create them)

## Contact

- **GitHub**: https://github.com/monadic/devops-examples
- **Example Directory**: cost-optimizer/
- **Organization ID**: afab8926-c115-4419-9857-f135580a0244
