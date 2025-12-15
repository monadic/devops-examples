# config-lineage

Query configuration inheritance across your fleet.

## The Problem

You have 500 deployments with inheritance: base → prod → prod-eu.

When something breaks, you need to know:
- "Why does prod-eu have replicas=5?" (which layer set it?)
- "If I change base, what's affected downstream?"

Git has files. Kustomize has overlays. Neither tracks inheritance at runtime.

## Usage

```bash
# Show inheritance chain
./config-lineage show prod-eu/trade-service

# Blame a specific value
./config-lineage blame prod-eu/trade-service spec.replicas

# Show downstream impact
./config-lineage impact base/trade-service

# Diff two units
./config-lineage diff base/trade-service prod-eu/trade-service
```

## Commands

### show

Display the inheritance chain for a unit:

```bash
$ ./config-lineage show prod-eu/trade-service

Lineage for prod-eu/trade-service:

prod-eu/trade-service (this unit)
    replicas: 5 (overrides 3 from prod/trade-service)
  └── prod/trade-service (upstream)
      replicas: 3 (overrides 1 from base/trade-service)
    └── base/trade-service (upstream)
        replicas: 1
```

### blame

Show where a specific value came from:

```bash
$ ./config-lineage blame prod-eu/trade-service spec.replicas

spec.replicas = 5
  Set in: prod-eu/trade-service
  Previous: 3

Lineage:
  → prod-eu/trade-service: 5
    prod/trade-service: 3
    base/trade-service: 1
```

### impact

Show what downstream units would be affected by changes:

```bash
$ ./config-lineage impact base/trade-service

Downstream units affected by changes to base/trade-service:

  - dev/trade-service
  - staging/trade-service
  - prod/trade-service
  - prod-us/trade-service
  - prod-eu/trade-service
  - prod-asia/trade-service

Total: 6 downstream units
```

### diff

Compare two units:

```bash
$ ./config-lineage diff base/trade-service prod-eu/trade-service

Differences from base/trade-service to prod-eu/trade-service:

  spec.replicas: 1 → 5
  metadata.labels.region: <nil> → eu
```

## Prerequisites

```bash
export CUB_TOKEN=$(cub auth get-token)
```

## Build

```bash
go build -o config-lineage .
```

## How It Works

ConfigHub units have explicit upstream/downstream relationships. When you create a unit with `--upstream-space` and `--upstream-unit`, ConfigHub tracks this relationship.

config-lineage walks these relationships to:
1. Build the inheritance chain (show, blame)
2. Find all downstream dependents (impact)
3. Compare effective values (diff)

This is read-only - it queries ConfigHub but doesn't modify anything.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `CUB_TOKEN` | ConfigHub API token | required |
| `CUB_API_URL` | ConfigHub API URL | https://hub.confighub.com/api |
