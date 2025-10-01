# ConfigHub Worker Setup - Important Note

## Critical Setup Requirement

When creating ConfigHub workers, you **MUST** use the `--include-secret` flag to generate proper authentication credentials for each worker.

## The Issue

If you create multiple workers and reuse the same secret, workers will fail with:
```
[ERROR] Failed to get bridge worker slug: server returned status 404: 404 Not Found
```

This happens because each worker needs its own unique `CONFIGHUB_WORKER_SECRET`.

## Correct Usage

```bash
# Create worker in ConfigHub
cub worker create my-worker --space my-space

# Install with --include-secret to generate proper credentials
cub worker install my-worker \
  --namespace confighub \
  --space my-space \
  --include-secret \
  --export > worker.yaml

# Apply to cluster
kubectl apply -f worker.yaml
```

## What `--include-secret` Does

Without `--include-secret`:
- Generates deployment manifest only
- Expects existing `confighub-worker-env` secret
- If secret exists from another worker, new worker will use WRONG credentials
- Worker fails to authenticate and gets 404 errors

With `--include-secret`:
- Generates deployment manifest AND secret manifest
- Secret contains correct `CONFIGHUB_WORKER_SECRET` for THIS specific worker
- Worker authenticates successfully and connects

## Our Experience

Initial attempts without `--include-secret`:
```bash
# First worker (devops-test-worker) - Created secret
cub worker install devops-test-worker --export  # No --include-secret
kubectl apply -f worker.yaml
→ ✅ Works (created new secret)

# Second worker (cost-optimizer-worker) - Reused old secret
cub worker install cost-optimizer-worker --export  # No --include-secret
kubectl apply -f worker.yaml
→ ❌ FAILS with 404 (uses devops-test-worker's credentials!)
```

Fixed with `--include-secret`:
```bash
cub worker install cost-optimizer-worker --include-secret --export
kubectl apply -f worker.yaml
→ ✅ Works (proper credentials in secret)
```

## Verification

After applying with `--include-secret`, you should see:

### Worker Status
```bash
$ cub worker list --space my-space
NAME       CONDITION    SPACE        LAST-SEEN
my-worker  Ready        my-space     2025-10-01 18:31:55
```

### Target Created
```bash
$ cub target list --space my-space
NAME            WORKER      PROVIDERTYPE    PARAMETERS
k8s-my-worker   my-worker   Kubernetes      {"WaitTimeout":"2m0s"}
```

### Worker Logs (Success)
```bash
$ kubectl logs -n confighub -l app=my-worker --tail=5
[INFO] Successfully connected to event stream in 375ms, status: 200 200 OK
[INFO] Starting to read events from stream
```

## Summary

**Always use `--include-secret` when installing ConfigHub workers** unless you're intentionally managing secrets separately (e.g., via external secret management).

This ensures each worker has the correct authentication credentials and can connect to ConfigHub successfully.
