# Helm Integration Design for DevOps SDK

Based on actual ConfigHub implementation review, here's the corrected Helm integration design.

## Current ConfigHub Helm Implementation (Reality)

ConfigHub has built-in Helm support via CLI commands:
- `cub helm install` - Render and install Helm charts as ConfigHub units
- `cub helm upgrade` - Update existing units with new chart versions

### How It Actually Works

1. **Chart Rendering**: Uses Helm SDK to render templates locally
2. **Unit Creation**: Creates ConfigHub units with rendered YAML
3. **Smart Splitting**: Separates CRDs from regular resources
4. **Namespace Handling**: Prepends namespace definition to resources
5. **Linking**: Links main unit to CRDs unit for relationships

### Key Implementation Details from Source

```go
// From helm_install.go
- Renders chart using Helm SDK (helm.sh/helm/v3)
- Creates 1-2 units:
  - {release-name}: Main resources + namespace
  - {release-name}-crds: CRDs (if any exist)
- Uses labels: HelmChart={chart}, HelmRelease={release}
- Supports --values, --set, --version flags
- Handles both crds/ directory and templated CRDs
```

## SDK Integration Approach

### Option 1: Direct CLI Wrapper (Simplest)
```go
// SDK wraps existing cub helm commands
type HelmHelper struct {
    cub *ConfigHubClient
}

func (h *HelmHelper) InstallChart(release, chart string, opts HelmOptions) error {
    cmd := exec.Command("cub", "helm", "install",
        "--space", h.cub.SpaceID.String(),
        "--namespace", opts.Namespace,
        release, chart,
        "--version", opts.Version)

    for _, v := range opts.Values {
        cmd.Args = append(cmd.Args, "--set", v)
    }

    return cmd.Run()
}

func (h *HelmHelper) UpgradeChart(release, chart string, opts HelmOptions) error {
    cmd := exec.Command("cub", "helm", "upgrade",
        "--space", h.cub.SpaceID.String(),
        release, chart,
        "--version", opts.Version)

    if opts.UpdateCRDs {
        cmd.Args = append(cmd.Args, "--update-crds")
    }

    return cmd.Run()
}
```

### Option 2: Native SDK Implementation (More Control)
```go
// SDK reimplements Helm rendering + ConfigHub unit creation
type HelmManager struct {
    cub *ConfigHubClient
}

func (h *HelmManager) InstallChart(ctx context.Context, req HelmInstallRequest) error {
    // 1. Render chart using Helm SDK
    renderedYAML, err := h.renderChart(req)
    if err != nil {
        return err
    }

    // 2. Split into CRDs and resources
    crds, resources := k8skit.SplitResources(renderedYAML)

    // 3. Create CRDs unit if needed
    if len(crds) > 0 {
        _, err = h.cub.CreateUnit(CreateUnitRequest{
            SpaceID: req.SpaceID,
            Slug:    req.ReleaseName + "-crds",
            Data:    crds,
            Labels: map[string]string{
                "HelmChart":   req.ChartName,
                "HelmRelease": req.ReleaseName,
            },
        })
        if err != nil {
            return err
        }
    }

    // 4. Create main resources unit with namespace
    namespace := h.generateNamespace(req.Namespace)
    fullYAML := namespace + "---\n" + resources

    _, err = h.cub.CreateUnit(CreateUnitRequest{
        SpaceID: req.SpaceID,
        Slug:    req.ReleaseName,
        Data:    fullYAML,
        Labels: map[string]string{
            "HelmChart":   req.ChartName,
            "HelmRelease": req.ReleaseName,
        },
    })

    return err
}

func (h *HelmManager) renderChart(req HelmInstallRequest) (string, error) {
    // Use helm.sh/helm/v3 SDK like ConfigHub does
    settings := cli.New()
    actionConfig := new(action.Configuration)

    // Locate and load chart
    cp, err := action.ChartPathOptions{
        Version: req.Version,
        RepoURL: req.RepoURL,
    }.LocateChart(req.Chart, settings)

    chart, err := loader.Load(cp)

    // Build values
    values := h.mergeValues(req.ValuesFiles, req.SetValues)

    // Render templates
    engine := engine.Engine{}
    return engine.Render(chart, values)
}
```

## DevOps App Integration Examples

### 1. Drift Detector with Helm Charts
```go
func (d *DriftDetector) detectHelmDrift() []DriftItem {
    // Find all Helm releases
    units, _ := d.app.Cub.ListUnits(ListUnitsParams{
        Where: "Labels.HelmRelease != ''",
    })

    for _, unit := range units {
        release := unit.Labels["HelmRelease"]
        chart := unit.Labels["HelmChart"]

        // Check if newer chart version available
        latestVersion := d.getLatestChartVersion(chart)
        currentVersion := unit.Labels["HelmChartVersion"]

        if latestVersion != currentVersion {
            driftItems = append(driftItems, DriftItem{
                Type:        "helm-version",
                Release:     release,
                Current:     currentVersion,
                Expected:    latestVersion,
                Correction:  fmt.Sprintf("cub helm upgrade %s %s --version %s",
                           release, chart, latestVersion),
            })
        }
    }

    return driftItems
}
```

### 2. Cost Optimizer with Helm Values
```go
func (o *Optimizer) optimizeHelmRelease(release string, analysis *WasteAnalysis) error {
    // Generate optimized values
    optimizedValues := map[string]string{
        "resources.requests.cpu":    fmt.Sprintf("%dm", analysis.OptimizedCPU),
        "resources.requests.memory": fmt.Sprintf("%dMi", analysis.OptimizedMemory),
        "replicas":                  fmt.Sprintf("%d", analysis.OptimizedReplicas),
    }

    // Apply via helm upgrade with new values
    return o.app.HelmHelper.UpgradeChart(release, "", HelmOptions{
        SetValues: optimizedValues,
    })
}
```

### 3. Compliance Checker for Helm Charts
```go
func (c *ComplianceChecker) checkHelmCompliance() []ComplianceIssue {
    var issues []ComplianceIssue

    // Check all Helm releases
    units, _ := c.app.Cub.ListUnits(ListUnitsParams{
        Where: "Labels.HelmRelease != ''",
    })

    for _, unit := range units {
        // Validate chart is from approved repo
        chart := unit.Labels["HelmChart"]
        if !c.isApprovedChart(chart) {
            issues = append(issues, ComplianceIssue{
                Type:     "unapproved-chart",
                Release:  unit.Labels["HelmRelease"],
                Chart:    chart,
                Fix:      "Use chart from approved repository",
            })
        }

        // Check for required labels
        if unit.Labels["environment"] == "" {
            issues = append(issues, ComplianceIssue{
                Type:    "missing-environment-label",
                Release: unit.Labels["HelmRelease"],
                Fix:     fmt.Sprintf("cub unit update %s --label environment=prod", unit.Slug),
            })
        }
    }

    return issues
}
```

## Implementation Recommendations

### Phase 1: CLI Wrapper (Quick Win)
1. Add `HelmHelper` to SDK using exec.Command
2. Wrap `cub helm install` and `cub helm upgrade`
3. Parse command output for results
4. **Benefit**: Working immediately, minimal code

### Phase 2: Native Implementation (Full Control)
1. Import helm.sh/helm/v3 SDK packages
2. Reimplement chart rendering logic
3. Direct API calls to create/update units
4. **Benefit**: Better error handling, no CLI dependency

### Phase 3: Advanced Features
1. **Helm Repository Management**: Track approved charts
2. **Values Validation**: CEL expressions for Helm values
3. **Diff Generation**: Show what will change before upgrade
4. **Rollback Support**: Track previous versions in ConfigHub

## Key Advantages vs Vanilla Helm

1. **Audit Trail**: All Helm operations tracked in ConfigHub
2. **Environment Hierarchy**: Helm values inherit via upstream/downstream
3. **Drift Detection**: Know when clusters diverge from charts
4. **Bulk Operations**: Upgrade all releases with one command
5. **GitOps Ready**: ConfigHub can trigger Git commits

## Testing Approach

```bash
# Test Helm installation
cub helm install test-nginx bitnami/nginx --version 15.5.2

# Verify units created
cub unit list --filter "HelmRelease=test-nginx"

# Test upgrade
cub helm upgrade test-nginx bitnami/nginx --version 15.6.0

# Check drift detection
./drift-detector --check-helm

# Apply optimizations
./cost-optimizer --optimize-helm test-nginx
```

## Migration Path for Existing Helm Users

```bash
# Import existing Helm release
helm get values my-release > values.yaml
helm get manifest my-release > manifest.yaml

# Create ConfigHub unit from existing release
cub unit create my-release --data @manifest.yaml \
  --label HelmRelease=my-release \
  --label HelmChart=bitnami/nginx

# Future updates via ConfigHub
cub helm upgrade my-release bitnami/nginx --values values.yaml
```

## Conclusion

ConfigHub's Helm integration is simpler than initially thought:
- No complex Helm controllers or operators
- Just render charts and store as units
- Labels track Helm metadata
- SDK can wrap CLI or reimplement natively

This approach gives us Helm benefits (packaging, templating) while maintaining ConfigHub as the source of truth for all configurations.