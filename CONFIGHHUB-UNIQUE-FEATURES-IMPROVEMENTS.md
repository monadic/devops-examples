# ConfigHub Unique Features - SDK & Example Improvements

Based on the new ConfigHub features discovered, here are simple, high-impact improvements that leverage ConfigHub's unique capabilities:

## 1. Functions Framework Integration

### SDK Enhancement: Add Functions Client
```go
// confighub.go - Add to ConfigHubClient

// ExecuteFunction runs a ConfigHub function on units
func (c *ConfigHubClient) ExecuteFunction(functionName string, params FunctionParams) (*FunctionResult, error) {
    // Use ConfigHub's function framework for direct manipulation
    // No need to download/modify/upload YAML
    endpoint := fmt.Sprintf("/function/do/%s", functionName)
    return c.doRequest("POST", endpoint, params, &FunctionResult{})
}

// Example: Set image version without downloading YAML
func (c *ConfigHubClient) SetImageVersion(unitID uuid.UUID, image string) error {
    return c.ExecuteFunction("set-image-reference", FunctionParams{
        UnitID: unitID,
        Args: map[string]interface{}{
            "container-name": "main",
            "image-reference": image,
        },
    })
}
```

### Drift Detector Improvement
```go
// Instead of downloading YAML and patching:
func (d *DriftDetector) fixDriftDirectly(unit Unit, correction DriftCorrection) error {
    // Use ConfigHub function to fix drift in-place
    _, err := d.app.Cub.ExecuteFunction("set-replicas", FunctionParams{
        UnitID: unit.UnitID,
        Args: map[string]interface{}{
            "replicas": correction.DesiredReplicas,
        },
    })
    return err
}
```

**Benefits**:
- No YAML parsing/manipulation needed
- Atomic operations
- Validation built-in

## 2. ChangeSets for Grouped Changes

### SDK Enhancement: ChangeSet Operations
```go
// Track related drift corrections in a ChangeSet
type ChangeSetManager struct {
    client *ConfigHubClient
}

func (m *ChangeSetManager) CreateDriftCorrectionSet(description string) (*ChangeSet, error) {
    return m.client.CreateChangeSet(ChangeSetRequest{
        DisplayName: fmt.Sprintf("Drift Correction - %s", time.Now().Format("2006-01-02")),
        Description: description,
    })
}

func (m *ChangeSetManager) AddUnitToChangeSet(unitID, changeSetID uuid.UUID) error {
    return m.client.UpdateUnit(unitID, UpdateUnitRequest{
        ChangeSetID: &changeSetID,
    })
}
```

### Drift Detector with ChangeSets
```go
func (d *DriftDetector) detectAndFixWithChangeSet() error {
    // Create ChangeSet for this drift correction batch
    changeSet, _ := d.app.Cub.CreateChangeSet(ChangeSetRequest{
        DisplayName: "Drift Corrections",
        Description: fmt.Sprintf("Auto-corrections at %s", time.Now()),
    })

    // All corrections go into the same ChangeSet
    for _, drift := range driftItems {
        // Fix drift and associate with ChangeSet
        d.app.Cub.ExecuteFunction("fix-drift", FunctionParams{
            UnitID:      drift.UnitID,
            ChangeSetID: changeSet.ID,
            Args:        drift.Corrections,
        })
    }

    // Can now approve/rollback entire ChangeSet atomically
    return d.app.Cub.ApplyChangeSet(changeSet.ID)
}
```

**Benefits**:
- Group related changes
- Atomic rollback capability
- Audit trail of grouped corrections

## 3. Helm Integration for App Deployment

### SDK Enhancement: Helm Operations
```go
// deployment_helper.go enhancement
func (d *DeploymentHelper) DeployHelmChart(chartPath string, values map[string]interface{}) error {
    // Store Helm chart as ConfigHub unit
    chartUnit, _ := d.cub.CreateUnit(d.BaseSpaceID, CreateUnitRequest{
        Slug:        "app-chart",
        Data:        chartPath, // ConfigHub handles Helm chart storage
        Labels:      map[string]string{"type": "helm-chart"},
    })

    // Deploy using ConfigHub's Helm support
    return d.cub.HelmInstall(HelmInstallRequest{
        ChartUnitID: chartUnit.UnitID,
        ReleaseName: d.AppName,
        Values:      values,
        SpaceID:     d.BaseSpaceID,
    })
}
```

### Cost Optimizer with Helm
```go
// Generate optimized Helm values instead of raw YAML
func (o *Optimizer) GenerateHelmValues(analysis *WasteAnalysis) map[string]interface{} {
    return map[string]interface{}{
        "resources": map[string]interface{}{
            "requests": map[string]interface{}{
                "cpu":    analysis.OptimizedCPU,
                "memory": analysis.OptimizedMemory,
            },
        },
        "replicas": analysis.OptimizedReplicas,
    }
}

// Apply via Helm upgrade
func (o *Optimizer) ApplyOptimizationViaHelm(values map[string]interface{}) error {
    return o.app.Cub.HelmUpgrade(HelmUpgradeRequest{
        ReleaseName: "cost-optimized-app",
        Values:      values,
    })
}
```

**Benefits**:
- Standard Helm workflow
- Values management in ConfigHub
- Rollback via Helm

## 4. Package System for App Distribution

### SDK Enhancement: Package Operations
```go
// Create reusable DevOps app packages
func (d *DeploymentHelper) PackageApp() (*Package, error) {
    return d.cub.CreatePackage(PackageRequest{
        Name:        fmt.Sprintf("%s-devops-app", d.AppName),
        Description: "Complete DevOps app with monitoring",
        Units:       d.getAppUnits(), // All units for the app
        Filters:     d.getAppFilters(),
    })
}

// Load package in new environment
func (d *DeploymentHelper) DeployFromPackage(packageID uuid.UUID, targetSpace string) error {
    return d.cub.LoadPackage(packageID, targetSpace)
}
```

**Benefits**:
- Distribute complete apps
- Version app bundles
- Simplified deployment

## 5. Function-Based Validation

### SDK Enhancement: Validation Functions
```go
// Use ConfigHub's validation functions
func (c *ConfigHubClient) ValidateNoPlaceholders(unitID uuid.UUID) (bool, error) {
    result, err := c.ExecuteFunction("no-placeholders", FunctionParams{
        UnitID: unitID,
    })
    return result.Valid, err
}

func (c *ConfigHubClient) ValidateCEL(unitID uuid.UUID, expression string) (bool, error) {
    result, err := c.ExecuteFunction("cel-validate", FunctionParams{
        UnitID: unitID,
        Args: map[string]interface{}{
            "expression": expression,
        },
    })
    return result.Valid, err
}
```

### Health Check Enhancement
```go
func (c *ComprehensiveHealthCheck) ValidateConfigurations() error {
    units, _ := c.ConfigHubClient.ListUnits(ListUnitsParams{
        SpaceID: c.SpaceID,
        Where:   "Labels.validate = 'true'",
    })

    for _, unit := range units {
        // Use ConfigHub functions for validation
        valid, _ := c.ConfigHubClient.ValidateNoPlaceholders(unit.UnitID)
        if !valid {
            c.issues = append(c.issues, fmt.Sprintf("Unit %s has placeholders", unit.Slug))
        }
    }
    return nil
}
```

## Simple Implementation Priority

### Phase 1: Functions (Easiest, Highest Impact)
1. Add `ExecuteFunction()` to SDK
2. Replace YAML manipulation with function calls
3. Use for drift correction and optimization

### Phase 2: ChangeSets (Better Grouping)
1. Add ChangeSet support to SDK
2. Group drift corrections
3. Enable atomic rollback

### Phase 3: Helm (If Using Charts)
1. Add Helm methods to SDK
2. Store charts as units
3. Manage values in ConfigHub

## Example: Updated Drift Detector with Functions

```go
// Before: Download, parse, modify, upload
func (d *DriftDetector) fixDriftOld(unit Unit, replicas int) error {
    // Download YAML
    yaml := d.downloadUnit(unit)
    // Parse and modify
    parsed := parseYAML(yaml)
    parsed["spec"]["replicas"] = replicas
    // Upload back
    return d.uploadUnit(unit, serializeYAML(parsed))
}

// After: Direct function call
func (d *DriftDetector) fixDriftNew(unit Unit, replicas int) error {
    return d.app.Cub.ExecuteFunction("set-replicas", FunctionParams{
        UnitID: unit.UnitID,
        Args:   map[string]interface{}{"replicas": replicas},
    })
}
```

## Key Advantages

1. **Less Code**: Functions eliminate YAML manipulation
2. **Atomic Operations**: ChangeSets group related changes
3. **Standard Patterns**: Helm for familiar workflows
4. **Validation**: Built-in validation functions
5. **Distribution**: Package system for sharing apps

These improvements make ConfigHub's unique features shine while keeping changes simple and focused.