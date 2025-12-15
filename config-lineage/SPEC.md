# config-lineage SPEC

## What It Does

config-lineage queries inheritance relationships across your fleet.

## The Problem

You have 500 deployments across dev/staging/prod and multiple regions. Configuration inherits: base → prod → prod-eu.

When something breaks, you need to answer:
- "Why does prod-eu have replicas=5?" (which layer set it?)
- "If I change base, what's affected downstream?"

### Why This Is Hard

Git has files. Kustomize has overlays. Neither tracks inheritance at runtime.

```
manifests/
  base/trade-service.yaml        # replicas: 1
  overlays/prod/trade-service.yaml    # replicas: 3
  overlays/prod-eu/trade-service.yaml # replicas: 5... but why?
```

You can run `kustomize build` to see the merged output, but not which layer set each value.

### What ConfigHub Provides

ConfigHub units have explicit upstream/downstream relationships:

```
base/trade-service
    ↓ (upstream)
prod/trade-service (replicas: 3)
    ↓ (upstream)
prod-eu/trade-service (replicas: 5, emergency patch)
```

You can query: "Show me the inheritance chain for prod-eu/trade-service"

```bash
cub unit get trade-service --space prod-eu --show-upstream
# Returns: prod-eu/trade-service → prod/trade-service → base/trade-service
```

## How It Works

Git is the source. CI syncs Git → ConfigHub, preserving upstream/downstream relationships.

```
Git (base + overlays)
     │
     │ CI sync (with upstream relationships)
     ▼
ConfigHub (queryable inheritance)
     ↑
config-lineage reads
```

config-lineage is read-only. It queries ConfigHub to show inheritance. It doesn't modify anything.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     config-lineage                          │
├─────────────────────────────────────────────────────────────┤
│  Lineage Query        │  Diff Engine       │  Visualizer    │
│  - Walk upstream      │  - Compare layers  │  - Tree view   │
│  - Find overrides     │  - Show deltas     │  - Web UI      │
│  - Track patches      │  - Blame values    │  - CLI output  │
└─────────────────────────────────────────────────────────────┘
           │                      │                   │
           ▼                      ▼                   ▼
      ConfigHub API          ConfigHub API       Terminal/Browser
           │
           ▼
      Unit relationships (upstream/downstream)
```

## Core Operations

### 1. Query Inheritance Chain

```bash
# CLI
config-lineage show prod-eu/trade-service

# Output:
# prod-eu/trade-service
#   ├── replicas: 5 (set here, overrides prod)
#   ├── image: trade:v2.1 (inherited from prod)
#   └── upstream: prod/trade-service
#         ├── replicas: 3 (set here, overrides base)
#         ├── image: trade:v2.1 (set here)
#         └── upstream: base/trade-service
#               ├── replicas: 1
#               └── image: trade:v2.0
```

### 2. Blame a Value

```bash
# Where did replicas=5 come from?
config-lineage blame prod-eu/trade-service spec.replicas

# Output:
# spec.replicas = 5
#   Set in: prod-eu/trade-service
#   Changed: 2024-01-15 by ops-team (emergency scale-up)
#   Previous: 3 (from prod/trade-service)
```

### 3. Impact Analysis

```bash
# What happens if I change base/trade-service?
config-lineage impact base/trade-service

# Output:
# Downstream units affected by changes to base/trade-service:
#   - dev/trade-service (inherits all)
#   - staging/trade-service (inherits all)
#   - prod/trade-service (overrides: replicas, image)
#   - prod-us/trade-service (inherits from prod)
#   - prod-eu/trade-service (overrides: replicas)
#   - prod-asia/trade-service (inherits from prod)
```

### 4. Find Drift from Base

```bash
# What has prod-eu overridden vs base?
config-lineage diff base/trade-service prod-eu/trade-service

# Output:
# Differences from base/trade-service to prod-eu/trade-service:
#   spec.replicas: 1 → 5 (changed at prod-eu, prod)
#   spec.template.spec.containers[0].image: trade:v2.0 → trade:v2.1 (changed at prod)
#   metadata.labels.region: (none) → "eu" (added at prod-eu)
```

## ConfigHub SDK Usage

```go
// Walk upstream chain
func getLineage(ctx context.Context, client *confighub.Client, space, unit string) ([]LineageNode, error) {
    var chain []LineageNode

    current, err := client.Units().Get(ctx, space, unit)
    if err != nil {
        return nil, err
    }

    chain = append(chain, LineageNode{
        Space: space,
        Unit:  unit,
        Data:  current.Data,
    })

    // Walk upstream
    for current.UpstreamSpace != "" && current.UpstreamUnit != "" {
        upstream, err := client.Units().Get(ctx, current.UpstreamSpace, current.UpstreamUnit)
        if err != nil {
            break
        }
        chain = append(chain, LineageNode{
            Space: current.UpstreamSpace,
            Unit:  current.UpstreamUnit,
            Data:  upstream.Data,
        })
        current = upstream
    }

    return chain, nil
}
```

## Value Proposition (Honest Assessment)

**What config-lineage provides:**
- Visibility: See where configuration values actually come from
- Debugging: Quickly identify which layer introduced a problem
- Impact analysis: Understand what changes affect before making them
- Audit: Track configuration changes through inheritance

**What it does NOT provide:**
- Enforcement: It shows relationships, doesn't prevent bad ones
- Auto-fix: It tells you what's wrong, not how to fix it
- Git history: It tracks ConfigHub state, not Git commits

**When to use it:**
- Debugging "why is prod-eu different from prod-us?"
- Before making base changes that affect many environments
- Auditing configuration after incidents
- Onboarding (understanding existing structure)

**When NOT to use it:**
- Simple setups with no inheritance
- If you don't use ConfigHub's upstream/downstream features

## Comparison to Existing Tools

| Tool | What it shows |
|------|---------------|
| `kubectl get` | Current runtime state (no history, no lineage) |
| `git log` | File history (no inheritance relationships) |
| `kustomize build` | Final merged output (no layer attribution) |
| `helm template` | Rendered templates (no value sources) |
| **config-lineage** | Inheritance chain with layer attribution |

## Implementation Status

Not yet implemented. This SPEC defines the requirements.

## Dependencies

- ConfigHub SDK (Go)
- ConfigHub CLI (`cub`)
- Units must be created with --upstream-space/--upstream-unit flags
