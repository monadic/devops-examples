# ConfigHub Worker 404 Error - Bug Report

**Date**: 2025-10-01
**Reporter**: Cost-Optimizer Example Implementation
**Severity**: High - Blocks worker-based deployment workflow

## Summary

ConfigHub workers fail to start with "404 Not Found" error when trying to get bridge worker slug. This prevents the complete ConfigHub → Worker → Kubernetes deployment workflow from functioning.

## Environment

- **ConfigHub API**: https://hub.confighub.com/api
- **CLI Version**: Latest (as of 2025-10-01)
- **Kubernetes**: Kind cluster (local)
- **Space**: sunrise-cub-cost-optimizer-base (ID: 743a8f42-8c32-4771-ac65-cdbe75252397)

## Steps to Reproduce

1. Create a ConfigHub space:
```bash
cub space create sunrise-cub-cost-optimizer-base \
  --label app=cost-optimizer \
  --label type=analysis
```

2. Create a worker in that space:
```bash
cub worker create cost-optimizer-worker --space sunrise-cub-cost-optimizer-base
# Output: Successfully created bridgeworker cost-optimizer-worker (82bde808-ce33-4e54-9646-0ff2d0ddf97c)
```

3. Install worker to Kubernetes:
```bash
cub worker install cost-optimizer-worker \
  --namespace confighub \
  --space sunrise-cub-cost-optimizer-base \
  --export > worker-manifest.yaml

kubectl apply -f worker-manifest.yaml
```

4. Check worker logs:
```bash
kubectl logs -n confighub -l app=cost-optimizer-worker
```

## Expected Behavior

Worker should:
1. Start successfully
2. Connect to ConfigHub API
3. Register as "Ready" condition
4. Create a target (e.g., `k8s-cost-optimizer-worker`)
5. Be available for `cub unit apply` operations

## Actual Behavior

Worker crashes with 404 error:

```
2025-10-01T17:50:31Z	INFO	Using dispatcher pattern for multi-worker support with unit-level serialization
2025-10-01T17:50:31Z	INFO	Registered worker	{"toolchainType": "Kubernetes/YAML", "providerType": "Kubernetes"}
2025-10-01T17:50:31Z	INFO	Registered bridge worker	{"workerType": "kubernetes", "toolchainType": "Kubernetes/YAML", "providerType": "Kubernetes"}
2025-10-01T17:50:31Z	INFO	Registered function worker	{"toolchainType": "Kubernetes/YAML"}
2025-10-01T17:50:31Z	INFO	Registered function worker	{"workerType": "kubernetes", "toolchainType": "Kubernetes/YAML"}
2025/10/01 17:50:31 Starting worker with ID: 82bde808-ce33-4e54-9646-0ff2d0ddf97c
2025/10/01 17:50:31 Starting worker with Token: ch_cjvr7...
2025/10/01 17:50:32 [ERROR] Failed to get bridge worker slug: server returned status 404: 404 Not Found
2025/10/01 17:50:32 Error starting worker: failed to get bridge worker slug: server returned status 404: 404 Not Found
2025-10-01T17:50:32Z	INFO	Failed to start worker	{"error": "failed to get bridge worker slug: server returned status 404: 404 Not Found"}
2025-10-01T17:50:32Z	ERROR	failed to execute command	{"error": "failed to get bridge worker slug: server returned status 404: 404 Not Found"}
main.main
	/go/src/app/public/cmd/cub-worker/main.go:503
runtime.main
	/usr/local/go/src/runtime/proc.go:283
```

Pod enters CrashLoopBackOff state.

## Worker Status in ConfigHub

```bash
$ cub worker list --space sunrise-cub-cost-optimizer-base

NAME                     CONDITION       SPACE                              LAST-SEEN
cost-optimizer-worker    Disconnected    sunrise-cub-cost-optimizer-base    0001-01-01 00:00:00
```

## Target Status

```bash
$ cub target list --space sunrise-cub-cost-optimizer-base

NAME    WORKER    PROVIDERTYPE    PARAMETERS    SPACE
# Empty - no targets created because worker failed to connect
```

## Impact

This bug prevents:
1. ✅ Units can be created in ConfigHub
2. ❌ Workers cannot connect and register
3. ❌ Targets are not created
4. ❌ `cub unit apply` workflow is blocked
5. ❌ ConfigHub → Worker → Kubernetes deployment pattern is broken

## Workaround

**Manual kubectl apply** (bypasses ConfigHub):
```bash
# Units exist in ConfigHub but show STATUS=NoLive
kubectl apply -f confighub/workloads/nginx-deployment.yaml
kubectl apply -f confighub/workloads/redis-deployment.yaml
kubectl apply -f confighub/workloads/postgres-deployment.yaml
```

This deploys to Kubernetes but:
- ConfigHub units remain STATUS=NoLive
- No audit trail in ConfigHub
- Can't use `cub unit update` + apply workflow
- Defeats purpose of ConfigHub-native deployment

## Comparison: Working vs Broken Worker

### Working Worker (drift-detector)
Created ~150 minutes ago, still working:

```bash
$ cub worker list --space cozy-cub-drift-detector-base

NAME                  CONDITION    SPACE                           LAST-SEEN
devops-test-worker    Ready        cozy-cub-drift-detector-base    2025-10-01 17:50:34

$ cub target list --space cozy-cub-drift-detector-base

NAME                      WORKER                PROVIDERTYPE    PARAMETERS                SPACE
k8s-devops-test-worker    devops-test-worker    Kubernetes      {"WaitTimeout":"2m0s"}    cozy-cub-drift-detector-base
```

### Broken Worker (cost-optimizer)
Created just now, immediately fails:

```bash
$ cub worker list --space sunrise-cub-cost-optimizer-base

NAME                     CONDITION       SPACE                              LAST-SEEN
cost-optimizer-worker    Disconnected    sunrise-cub-cost-optimizer-base    0001-01-01 00:00:00

$ cub target list --space sunrise-cub-cost-optimizer-base

NAME    WORKER    PROVIDERTYPE    PARAMETERS    SPACE
# Empty
```

## Key Findings

### Both Workers Use Same Infrastructure
✅ **Same Kubernetes cluster**: Both deployed to Kind cluster, namespace `confighub`
✅ **Same ConfigHub API**: Both use secret `confighub-worker-env` with same credentials
✅ **Same API URL**: CONFIGHUB_URL empty (uses default API endpoint)

### Worker Record EXISTS in API
The `cub` CLI can successfully fetch the broken worker:

```bash
$ cub worker get cost-optimizer-worker --space sunrise-cub-cost-optimizer-base --json
{
  "BridgeWorkerID": "82bde808-ce33-4e54-9646-0ff2d0ddf97c",
  "Slug": "cost-optimizer-worker",
  "Condition": "Disconnected",
  "LastSeenAt": "0001-01-01T00:00:00Z"
}
```

**BUT** the worker pod itself gets 404 when trying to fetch the same data during startup!

### API Endpoint Inconsistency
- ✅ CLI endpoint works: Can fetch worker by slug/ID
- ❌ Worker pod endpoint fails: Returns 404 for same worker

This suggests the worker pod is using a different API endpoint that doesn't properly return the worker metadata, even though the worker record exists in the database.

## Hypothesis

Possible causes:
1. **API endpoint mismatch**: Worker pod uses different endpoint than CLI
2. **Worker authentication issue**: Worker secret not properly authorized to fetch its own metadata
3. **Timing race condition**: Worker pod starts before API fully commits worker record
4. **API version incompatibility**: Newer worker pods incompatible with current API
5. **Organization/space scoping issue**: Worker can't access its own metadata across space boundaries

## Diagnostic Information

**Worker IDs attempted**:
- First attempt: 307b4740-51fe-4d47-96a2-dcc59dfa7293 (failed)
- Second attempt: 82bde808-ce33-4e54-9646-0ff2d0ddf97c (failed)

**Space ID**: 743a8f42-8c32-4771-ac65-cdbe75252397

**Organization ID**: afab8926-c115-4419-9857-f135580a0244

**API Endpoint**: https://hub.confighub.com/api

## Reproducibility

**Frequency**: 100% (2 out of 2 worker creation attempts failed)

**Other affected spaces**:
Unknown - only tested with `sunrise-cub-cost-optimizer-base`

## Current Status

Units are visible in ConfigHub but unusable for deployment:

**View in ConfigHub Web UI**:
https://hub.confighub.com/space/743a8f42-8c32-4771-ac65-cdbe75252397

**Units present**:
- nginx-web (5 replicas, STATUS=NoLive)
- redis-cache (3 replicas, STATUS=NoLive)
- postgres-db (2 replicas, STATUS=NoLive)

**CLI verification**:
```bash
$ cub unit list --space sunrise-cub-cost-optimizer-base

NAME           SPACE                              STATUS
nginx-web      sunrise-cub-cost-optimizer-base    NoLive
postgres-db    sunrise-cub-cost-optimizer-base    NoLive
redis-cache    sunrise-cub-cost-optimizer-base    NoLive
```

## Request for ConfigHub Team

Please investigate:
1. Why is the worker API returning 404 for "get bridge worker slug"?
2. Why does the older `devops-test-worker` still work but new workers fail?
3. Is there a change needed in worker deployment process?
4. Are there any API version compatibility issues?

## Contact

- **Project**: https://github.com/monadic/devops-examples
- **Issue**: Cost Optimizer Example - Worker Connection Failure
- **Files**: `/cost-optimizer/` directory
