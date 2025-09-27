# Package System Design for DevOps Apps

Based on ConfigHub source code review, here's how the Package system actually works.

## What Are ConfigHub Packages?

ConfigHub Packages are **serialized collections** of configuration resources:
- **Export**: Serialize spaces, units, links, filters, etc. to a directory structure
- **Import**: Load package from directory or remote URL (including GitHub)
- **Distribution**: Share complete app configurations as packages

## Current Implementation (from source)

### Package Structure
```
my-package/
├── manifest.json           # Package manifest listing all resources
├── spaces/                 # Space definitions
│   └── *.json
├── units/                  # Unit definitions
│   └── *.json
├── unit_data/             # Unit YAML/JSON data
│   └── *.yaml
├── links/                 # Link definitions
│   └── *.json
├── filters/               # Filter definitions
│   └── *.json
├── workers/               # Worker definitions
│   └── *.json
├── targets/               # Target definitions
│   └── *.json
└── views/                 # View definitions
    └── *.json
```

### CLI Commands (Experimental)
```bash
# Create package from existing resources
cub package create ./my-package --space my-space --where "Labels.app='drift-detector'"

# Load package (local or remote)
cub package load ./my-package --prefix staging
cub package load https://github.com/monadic/packages/drift-detector
```

**Note**: Package commands require `CONFIGHUB_EXPERIMENTAL=1` environment variable.

## SDK Package Helper Design

### Option 1: CLI Wrapper (Quick Implementation)
```go
type PackageHelper struct {
    cub *ConfigHubClient
}

func (p *PackageHelper) CreatePackage(dir string, opts PackageOptions) error {
    cmd := exec.Command("cub", "package", "create", dir,
        "--space", opts.SpaceID.String())

    if opts.Where != "" {
        cmd.Args = append(cmd.Args, "--where", opts.Where)
    }

    if opts.Filter != "" {
        cmd.Args = append(cmd.Args, "--filter", opts.Filter)
    }

    cmd.Env = append(os.Environ(), "CONFIGHUB_EXPERIMENTAL=1")
    return cmd.Run()
}

func (p *PackageHelper) LoadPackage(source string, prefix string) error {
    cmd := exec.Command("cub", "package", "load", source)

    if prefix != "" {
        cmd.Args = append(cmd.Args, "--prefix", prefix)
    }

    cmd.Env = append(os.Environ(), "CONFIGHUB_EXPERIMENTAL=1")
    return cmd.Run()
}
```

### Option 2: Native Implementation (Full Control)
```go
type PackageManager struct {
    cub *ConfigHubClient
}

// CreatePackage exports resources to a package directory
func (p *PackageManager) CreatePackage(req CreatePackageRequest) error {
    // 1. Create directory structure
    dirs := []string{"spaces", "units", "unit_data", "links", "filters"}
    for _, dir := range dirs {
        os.MkdirAll(filepath.Join(req.OutputDir, dir), 0755)
    }

    // 2. Fetch resources based on criteria
    units, err := p.cub.ListUnits(ListUnitsParams{
        SpaceID: req.SpaceID,
        Where:   req.Where,
    })

    // 3. Build manifest
    manifest := PackageManifest{
        Spaces: []SpaceEntry{},
        Units:  []UnitEntry{},
    }

    // 4. Serialize each resource
    for _, unit := range units {
        // Save unit details
        unitFile := fmt.Sprintf("units/%s.json", unit.Slug)
        p.saveJSON(req.OutputDir, unitFile, unit)

        // Save unit data
        dataFile := fmt.Sprintf("unit_data/%s.yaml", unit.Slug)
        p.saveData(req.OutputDir, dataFile, unit.Data)

        // Add to manifest
        manifest.Units = append(manifest.Units, UnitEntry{
            Slug:        unit.Slug,
            SpaceSlug:   req.SpaceSlug,
            DetailsLoc:  unitFile,
            UnitDataLoc: dataFile,
        })
    }

    // 5. Save manifest
    return p.saveJSON(req.OutputDir, "manifest.json", manifest)
}

// LoadPackage imports a package from directory or URL
func (p *PackageManager) LoadPackage(req LoadPackageRequest) error {
    var loader PackageLoader

    if strings.HasPrefix(req.Source, "http") {
        loader = NewRemotePackageLoader(req.Source)
    } else {
        loader = NewLocalPackageLoader(req.Source)
    }

    // Load manifest
    manifest, err := loader.LoadManifest()
    if err != nil {
        return err
    }

    // Create spaces with prefix
    for _, space := range manifest.Spaces {
        if req.Prefix != "" {
            space.Slug = req.Prefix + "-" + space.Slug
        }

        details, _ := loader.LoadSpaceDetails(space)
        createdSpace, _ := p.cub.CreateSpace(CreateSpaceRequest{
            Slug:        space.Slug,
            DisplayName: details.DisplayName,
            Labels:      details.Labels,
        })

        req.spaceMapping[space.Slug] = createdSpace.SpaceID
    }

    // Create units
    for _, unit := range manifest.Units {
        data, _ := loader.LoadUnitData(unit)
        details, _ := loader.LoadUnitDetails(unit)

        p.cub.CreateUnit(CreateUnitRequest{
            SpaceID: req.spaceMapping[unit.SpaceSlug],
            Slug:    unit.Slug,
            Data:    string(data),
            Labels:  details.Labels,
        })
    }

    return nil
}
```

## DevOps App Package Examples

### 1. Package a Complete DevOps App
```go
func PackageDriftDetector() error {
    pkg := NewPackageHelper(cub)

    // Export drift detector and all its resources
    return pkg.CreatePackage("./packages/drift-detector", PackageOptions{
        SpaceID: driftSpaceID,
        Where:   "Labels.app = 'drift-detector'",
        Filter:  "drift-detector/*",
    })
}
```

### 2. Deploy App from Package
```go
func DeployFromPackage(env string) error {
    pkg := NewPackageHelper(cub)

    // Load package with environment prefix
    return pkg.LoadPackage(
        "https://github.com/monadic/packages/drift-detector",
        env, // e.g., "staging" creates "staging-drift-detector" space
    )
}
```

### 3. App Distribution Workflow
```go
// Publisher side
func PublishApp(app *DevOpsApp) error {
    // 1. Create package
    pkg := NewPackageManager(app.Cub)
    err := pkg.CreatePackage(CreatePackageRequest{
        OutputDir: "./dist",
        SpaceID:   app.SpaceID,
        Where:     fmt.Sprintf("Labels.app = '%s'", app.Name),
    })

    // 2. Push to GitHub
    cmd := exec.Command("git", "add", ".")
    cmd.Dir = "./dist"
    cmd.Run()

    cmd = exec.Command("git", "commit", "-m", "Release v1.0.0")
    cmd.Dir = "./dist"
    cmd.Run()

    cmd = exec.Command("git", "push", "origin", "main")
    cmd.Dir = "./dist"
    return cmd.Run()
}

// Consumer side
func InstallApp(packageURL string, targetEnv string) error {
    pkg := NewPackageManager(cub)

    // Load from GitHub
    return pkg.LoadPackage(LoadPackageRequest{
        Source: packageURL,
        Prefix: targetEnv,
    })
}
```

### 4. Package Versioning
```go
type VersionedPackage struct {
    Version     string
    ReleaseDate time.Time
    Changelog   string
    PackageURL  string
}

func (p *PackageManager) CreateVersionedPackage(version string) error {
    // Create package with version in manifest
    manifest := PackageManifest{
        Version:     version,
        CreatedAt:   time.Now(),
        Description: "Drift Detector v" + version,
    }

    // Tag in git
    cmd := exec.Command("git", "tag", "v"+version)
    cmd.Run()

    return nil
}
```

## Package Benefits for DevOps Apps

### 1. Complete App Distribution
- **Bundle Everything**: Units, filters, links, workers, targets
- **Self-Contained**: No external dependencies
- **Version Control**: Git-based versioning

### 2. Environment Cloning
```bash
# Clone production to staging for testing
cub package create ./prod-backup --space production
cub package load ./prod-backup --prefix staging-test
```

### 3. Disaster Recovery
```bash
# Backup entire app
cub package create ./backups/$(date +%Y%m%d) --space prod

# Restore from backup
cub package load ./backups/20240115 --prefix restored
```

### 4. App Marketplace
```yaml
# package-registry.yaml
apps:
  - name: drift-detector
    version: 1.2.0
    url: https://github.com/monadic/packages/drift-detector

  - name: cost-optimizer
    version: 2.0.1
    url: https://github.com/monadic/packages/cost-optimizer

  - name: compliance-checker
    version: 1.0.0
    url: https://github.com/monadic/packages/compliance-checker
```

## Integration with DevOps Apps

### Drift Detector Package
```go
func (d *DriftDetector) ExportAsPackage() error {
    return d.app.PackageHelper.CreatePackage("./drift-detector-package",
        PackageOptions{
            SpaceID: d.spaceID,
            Where:   "Labels.managed-by = 'drift-detector'",
        })
}

func (d *DriftDetector) DeployToNewCluster(clusterName string) error {
    return d.app.PackageHelper.LoadPackage(
        "./drift-detector-package",
        clusterName,
    )
}
```

### Cost Optimizer Package
```go
func (o *CostOptimizer) PackageOptimizedConfigs() error {
    // Export only optimized configurations
    return o.app.PackageHelper.CreatePackage("./optimized-configs",
        PackageOptions{
            SpaceID: o.spaceID,
            Where:   "Labels.optimized = 'true'",
        })
}
```

## Testing Package Operations

```bash
# Enable experimental features
export CONFIGHUB_EXPERIMENTAL=1

# Test package creation
cub package create ./test-package --space my-space

# Verify package structure
ls -la ./test-package/
cat ./test-package/manifest.json

# Test package loading
cub package load ./test-package --prefix test

# Test remote loading
cub package load https://github.com/user/repo/packages/app
```

## Implementation Roadmap

### Phase 1: CLI Wrapper (1 day)
- [x] Review ConfigHub package implementation
- [ ] Create PackageHelper with CLI wrapper
- [ ] Add to SDK with basic create/load methods
- [ ] Test with drift-detector

### Phase 2: Native Implementation (3 days)
- [ ] Implement manifest parsing
- [ ] Add local/remote loaders
- [ ] Handle space/unit mapping
- [ ] Add progress reporting

### Phase 3: Advanced Features (1 week)
- [ ] Versioning support
- [ ] Dependency resolution
- [ ] Selective resource export
- [ ] Package validation
- [ ] Diff before load

## Key Advantages

1. **Complete App Portability**: Move entire apps between environments
2. **Disaster Recovery**: Backup and restore complete configurations
3. **App Distribution**: Share DevOps apps like software packages
4. **Environment Cloning**: Perfect copies for testing
5. **Version Control**: Git-based package versioning

## Conclusion

ConfigHub's Package system provides a powerful way to distribute DevOps apps:
- Export complete app configurations
- Load from local directories or remote URLs
- Perfect for app distribution and disaster recovery
- Currently experimental but fully functional

This gives us a competitive advantage over workflow-based tools that can't easily package and distribute complete applications.