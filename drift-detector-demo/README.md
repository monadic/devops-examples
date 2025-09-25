# Drift Detector Demo - Global App Pattern

This demo follows the canonical global-app pattern from ConfigHub examples, properly managing units and drift detection.

## Architecture

```
project-base/        # Base configuration (source of truth)
  ├── backend-api    # 3 replicas (base)
  ├── frontend-app   # 2 replicas (base)
  ├── cache-service  # 1 replica (base)
  └── database-service # 1 replica (base)

project-dev/         # Dev environment (upstream: base)
  └── [cloned units with upstream relationships]

project-staging/     # Staging (upstream: dev)
  └── [cloned units with upstream relationships]

project-prod/        # Production (upstream: staging)
  └── [cloned units with upstream relationships]
```

## Setup Instructions

### 1. Authenticate with ConfigHub

```bash
cub auth login
export CUB_TOKEN=<your-token>
```

### 2. Initialize the Project

```bash
cd drift-detector-demo

# Create base configuration and spaces
bin/install-base

# Create environment hierarchy
bin/install-envs
```

### 3. Apply to Kubernetes

```bash
# Apply dev environment
bin/apply-all dev

# Or apply staging
bin/apply-all staging
```

### 4. View the Dashboard

Open http://localhost:8082 to see:
- All units from ConfigHub
- Current Kubernetes state
- Any drift detected
- ConfigHub commands to fix drift

### 5. Test Drift Detection

```bash
# Induce drift using ConfigHub
bin/induce-drift dev

# This changes backend-api from 3 to 5 replicas
# The dashboard will show this drift
```

### 6. Fix Drift

Use the push-upgrade pattern to restore base configuration:

```bash
# Push-upgrade from base to dev
cub unit update backend-api --space <project>-dev --patch --upgrade
```

## Key Commands

View the unit hierarchy:
```bash
project=$(cat .cub-project)
cub unit tree --node=space --filter $project/app --space '*'
```

List units in an environment:
```bash
cub unit list --space $project-dev
```

Apply changes:
```bash
cub unit apply backend-api --space $project-dev --yes
```

## Global App Pattern Features

1. **Unique project prefix**: Generated with `cub space new-prefix`
2. **Environment hierarchy**: base → dev → staging → prod
3. **Upstream relationships**: Units inherit from parent environments
4. **Filters**: Target units by type (app, infra)
5. **Push-upgrade**: Propagate changes through environments
6. **ConfigHub as source of truth**: All changes through ConfigHub, never kubectl

## Files Structure

```
drift-detector-demo/
├── bin/
│   ├── install-base      # Create base configuration
│   ├── install-envs      # Create environment hierarchy
│   ├── new-env          # Create a new environment
│   ├── new-app-env      # Clone app units to environment
│   ├── new-infra        # Clone infra units
│   ├── apply-all        # Apply units to Kubernetes
│   └── induce-drift     # Create test drift
├── baseconfig/
│   ├── backend-api.yaml
│   ├── frontend-app.yaml
│   ├── cache-service.yaml
│   ├── database-service.yaml
│   └── ns-base.yaml
└── .cub-project         # Stores the unique project prefix
```

## Important Notes

- **Never use kubectl to modify resources** - always use ConfigHub
- The `.cub-project` file stores your unique project prefix
- All corrections should use `cub unit update` commands
- The drift detector monitors the difference between ConfigHub (desired) and Kubernetes (actual)

## Dashboard Access

The live dashboard at http://localhost:8082 shows:
- ConfigHub spaces and units
- Kubernetes deployments
- Drift detection results
- Correction commands (using ConfigHub only)
- Health check capability