# DevOps Examples

Kubernetes applications that use ConfigHub for fleet-wide operations.

## The Problem

Argo and Flux deploy configurations. They don't help you:

1. **Query a large fleet** - "Which of my 500 deployments run image X?"
2. **Make fleet-wide changes** - "Patch all of them to image Y"
3. **Build an API for this** - Programmatic access, not file editing

These tools reconcile Git → Cluster. They don't provide an operational layer for querying and mutating configurations at scale.

## What ConfigHub Adds

| Need | GitOps Tools | ConfigHub |
|------|--------------|-----------|
| Query fleet | Grep across repos | `cub unit list --where "Data CONTAINS 'image:v1'"` |
| Bulk change | Edit files, commit, PR, wait | `cub run set-image --space '*-prod-*'` |
| API access | Build your own | ConfigHub API + SDK |
| See all states | Argo UI (per-app) | Desired vs Live vs Drift (fleet-wide) |

### Multiple Views of State

| View | What it is | Where it lives |
|------|------------|----------------|
| **Desired** | What you declared | ConfigHub unit |
| **Live** | What's actually in the cluster | Queried via BridgeWorker |
| **Drift** | Are they equal? | Computed |

```bash
# See what you declared
cub unit get-data trade-service --space prod-eu

# See what's actually running
cub unit livestate trade-service --space prod-eu

# See the difference
cub unit diff trade-service --space prod-eu
```

Argo shows sync status per-application. ConfigHub shows Desired/Live/Drift across the entire fleet.

**The model:** Git is the source. CI syncs Git → ConfigHub. ConfigHub provides the query/mutation layer.

```
Git (source) → CI syncs → ConfigHub (query + mutate) → applies → Kubernetes
                               ↑
                     These examples use ConfigHub here
                               │
                               └── sync back to Git (PR) when needed
```

## Examples

### Operational Today

#### [drift-detector](./drift-detector)
Detects when Kubernetes runtime state differs from ConfigHub units.

**Problem it solves:** "Something changed my deployment, but I don't know what or when."

**How it uses ConfigHub:** Queries all units, compares to kubectl output, reports drift, optionally auto-corrects.

#### [cost-optimizer](./cost-optimizer)
Analyzes resource usage and suggests right-sizing.

**Problem it solves:** "My clusters are over-provisioned but I don't know where to cut."

**How it uses ConfigHub:** Queries units for resource requests/limits, correlates with metrics-server data, suggests patches.

#### [cost-impact-monitor](./cost-impact-monitor)
Pre-deployment cost analysis.

**Problem it solves:** "I want to know the cost impact before I deploy."

**How it uses ConfigHub:** Hooks into unit apply events, calculates cost delta, reports before deployment completes.

### Planned

#### [cve-responder](./cve-responder) (SPEC only)
Automated CVE response across infrastructure.

**Problem it solves:** "A critical CVE dropped. I need to find and patch all affected deployments in minutes, not hours."

**How it uses ConfigHub:**
1. Query: `cub unit list --where "Data CONTAINS 'vulnerable-image:1.0'"` (seconds)
2. Patch: `cub run set-image --image 'patched:1.1' --where "..."` (bulk update)
3. Apply: `cub unit apply --where "..."` (immediate, no PR wait)
4. Sync: Open PR to Git with the changes (eventual consistency)

See [cve-responder/SPEC.md](./cve-responder/SPEC.md) for details.

#### [config-lineage](./config-lineage) (SPEC only)
Configuration inheritance visualization.

**Problem it solves:** "Why does prod-eu have replicas=5? Where did that value come from?"

**How it uses ConfigHub:**
- Walk upstream chain: prod-eu → prod → base
- Show which layer set each value
- Impact analysis: "What downstream units are affected if I change base?"

See [config-lineage/SPEC.md](./config-lineage/SPEC.md) for details.

## For Argo/Flux Users

ConfigHub doesn't replace your GitOps workflow. It adds what's missing:

1. **Git stays the source** - your manifests stay in Git
2. **CI syncs to ConfigHub** - same CI pipeline, additional sync target
3. **Query and mutate via ConfigHub** - fleet-wide visibility and changes
4. **Sync back to Git** - PRs keep Git consistent after operational changes

Argo/Flux continue to deploy. ConfigHub provides the operational API they lack.

## Quick Start

Each example has setup instructions in its directory. Prerequisites:

```bash
# ConfigHub CLI
curl -fsSL https://hub.confighub.com/cub/install.sh | bash
cub auth login

# Kubernetes (local)
kind create cluster

# Verify setup
curl -fsSL https://raw.githubusercontent.com/monadic/devops-sdk/main/test-confighub-k8s | bash
```

## Project Structure

```
devops-examples/
├── drift-detector/       # Runtime drift detection
├── cost-optimizer/       # Resource right-sizing
├── cost-impact-monitor/  # Pre-deploy cost analysis
├── cve-responder/        # CVE response automation (SPEC)
└── config-lineage/       # Inheritance visualization (SPEC)
```

All examples use the [devops-sdk](https://github.com/monadic/devops-sdk) for ConfigHub operations.

## Documentation Standards

Commands in README files are validated before commit:

```bash
# Validate cub commands in docs
curl -fsSL https://raw.githubusercontent.com/monadic/devops-sdk/main/cub-command-analyzer.sh | bash -s -- .
```

## License

Proprietary - ConfigHub, Inc.
