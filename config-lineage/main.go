package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	sdk "github.com/monadic/devops-sdk"
	"gopkg.in/yaml.v3"
)

type ConfigLineage struct {
	app *sdk.DevOpsApp
}

type LineageNode struct {
	Space      string            `json:"space"`
	UnitSlug   string            `json:"unit_slug"`
	UnitID     string            `json:"unit_id"`
	Data       map[string]any    `json:"data,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	UpstreamID *string           `json:"upstream_id,omitempty"`
	Overrides  []Override        `json:"overrides,omitempty"`
}

type Override struct {
	Path     string `json:"path"`
	Value    any    `json:"value"`
	SetAt    string `json:"set_at"` // space/unit where this was set
	Previous any    `json:"previous,omitempty"`
}

type ImpactResult struct {
	DownstreamUnits []DownstreamUnit `json:"downstream_units"`
}

type DownstreamUnit struct {
	Space    string   `json:"space"`
	UnitSlug string   `json:"unit_slug"`
	Inherits []string `json:"inherits"` // which fields it inherits
	Overrides []string `json:"overrides"` // which fields it overrides
}

func main() {
	// Subcommands
	showCmd := flag.NewFlagSet("show", flag.ExitOnError)
	blameCmd := flag.NewFlagSet("blame", flag.ExitOnError)
	impactCmd := flag.NewFlagSet("impact", flag.ExitOnError)
	diffCmd := flag.NewFlagSet("diff", flag.ExitOnError)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Initialize SDK
	appConfig := sdk.DevOpsAppConfig{
		Name:        "config-lineage",
		Version:     "1.0.0",
		Description: "Query configuration inheritance across fleet",
		CubToken:    os.Getenv("CUB_TOKEN"),
		CubBaseURL:  sdk.GetEnvOrDefault("CUB_API_URL", "https://hub.confighub.com/api"),
	}

	app, err := sdk.NewDevOpsApp(appConfig)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	lineage := &ConfigLineage{app: app}

	switch os.Args[1] {
	case "show":
		showCmd.Parse(os.Args[2:])
		if showCmd.NArg() < 1 {
			fmt.Println("Usage: config-lineage show <space/unit>")
			os.Exit(1)
		}
		lineage.Show(showCmd.Arg(0))

	case "blame":
		blameCmd.Parse(os.Args[2:])
		if blameCmd.NArg() < 2 {
			fmt.Println("Usage: config-lineage blame <space/unit> <path>")
			fmt.Println("Example: config-lineage blame prod-eu/trade-service spec.replicas")
			os.Exit(1)
		}
		lineage.Blame(blameCmd.Arg(0), blameCmd.Arg(1))

	case "impact":
		impactCmd.Parse(os.Args[2:])
		if impactCmd.NArg() < 1 {
			fmt.Println("Usage: config-lineage impact <space/unit>")
			os.Exit(1)
		}
		lineage.Impact(impactCmd.Arg(0))

	case "diff":
		diffCmd.Parse(os.Args[2:])
		if diffCmd.NArg() < 2 {
			fmt.Println("Usage: config-lineage diff <space/unit> <space/unit>")
			os.Exit(1)
		}
		lineage.Diff(diffCmd.Arg(0), diffCmd.Arg(1))

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("config-lineage - Query configuration inheritance")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  show <space/unit>              Show inheritance chain")
	fmt.Println("  blame <space/unit> <path>      Show where a value came from")
	fmt.Println("  impact <space/unit>            Show downstream units affected by changes")
	fmt.Println("  diff <space/unit> <space/unit> Compare two units")
}

// Show displays the inheritance chain for a unit
func (l *ConfigLineage) Show(ref string) {
	space, unit := parseRef(ref)

	chain, err := l.getLineageChain(space, unit)
	if err != nil {
		log.Fatalf("Failed to get lineage: %v", err)
	}

	fmt.Printf("Lineage for %s/%s:\n\n", space, unit)

	for i, node := range chain {
		indent := strings.Repeat("  ", i)
		if i == 0 {
			fmt.Printf("%s%s/%s (this unit)\n", indent, node.Space, node.UnitSlug)
		} else {
			fmt.Printf("%s└── %s/%s (upstream)\n", indent, node.Space, node.UnitSlug)
		}

		// Show key values with override info
		if node.Data != nil {
			if spec, ok := node.Data["spec"].(map[string]any); ok {
				if replicas, ok := spec["replicas"]; ok {
					override := ""
					for _, o := range node.Overrides {
						if o.Path == "spec.replicas" {
							override = fmt.Sprintf(" (overrides %v from %s)", o.Previous, o.SetAt)
						}
					}
					fmt.Printf("%s    replicas: %v%s\n", indent, replicas, override)
				}
			}
		}
	}
}

// Blame shows where a specific value came from
func (l *ConfigLineage) Blame(ref, path string) {
	space, unit := parseRef(ref)

	chain, err := l.getLineageChain(space, unit)
	if err != nil {
		log.Fatalf("Failed to get lineage: %v", err)
	}

	// Find where the value was set
	var currentValue any
	var setAt string
	var previousValue any

	for i := len(chain) - 1; i >= 0; i-- {
		node := chain[i]
		value := getValueAtPath(node.Data, path)
		if value != nil {
			if currentValue == nil {
				currentValue = value
				setAt = fmt.Sprintf("%s/%s", node.Space, node.UnitSlug)
			} else if !equalValues(value, currentValue) {
				previousValue = currentValue
				currentValue = value
				setAt = fmt.Sprintf("%s/%s", node.Space, node.UnitSlug)
			}
		}
	}

	fmt.Printf("%s = %v\n", path, currentValue)
	fmt.Printf("  Set in: %s\n", setAt)
	if previousValue != nil {
		fmt.Printf("  Previous: %v\n", previousValue)
	}

	// Show full chain
	fmt.Printf("\nLineage:\n")
	for _, node := range chain {
		value := getValueAtPath(node.Data, path)
		marker := " "
		if fmt.Sprintf("%s/%s", node.Space, node.UnitSlug) == setAt {
			marker = "→"
		}
		fmt.Printf("  %s %s/%s: %v\n", marker, node.Space, node.UnitSlug, value)
	}
}

// Impact shows what downstream units would be affected by changes
func (l *ConfigLineage) Impact(ref string) {
	space, unit := parseRef(ref)

	// Get the source unit
	spaceObj, err := l.app.Cub.GetSpaceBySlug(space)
	if err != nil {
		log.Fatalf("Failed to get space: %v", err)
	}

	units, err := l.app.Cub.ListUnits(sdk.ListUnitsParams{
		SpaceID: spaceObj.SpaceID,
		Where:   fmt.Sprintf("Slug = '%s'", unit),
	})
	if err != nil || len(units) == 0 {
		log.Fatalf("Failed to get unit: %v", err)
	}

	sourceUnit := units[0]

	// Find all downstream units across all spaces
	allSpaces, err := l.app.Cub.ListSpaces()
	if err != nil {
		log.Fatalf("Failed to list spaces: %v", err)
	}

	var downstream []DownstreamUnit

	for _, s := range allSpaces {
		units, err := l.app.Cub.ListUnits(sdk.ListUnitsParams{
			SpaceID: s.SpaceID,
		})
		if err != nil {
			continue
		}

		for _, u := range units {
			if u.UpstreamUnitID != nil && *u.UpstreamUnitID == sourceUnit.UnitID {
				downstream = append(downstream, DownstreamUnit{
					Space:    s.Slug,
					UnitSlug: u.Slug,
				})
			}
		}
	}

	fmt.Printf("Downstream units affected by changes to %s/%s:\n\n", space, unit)

	if len(downstream) == 0 {
		fmt.Println("  (no downstream units found)")
		return
	}

	for _, d := range downstream {
		fmt.Printf("  - %s/%s\n", d.Space, d.UnitSlug)
	}

	fmt.Printf("\nTotal: %d downstream units\n", len(downstream))
}

// Diff compares two units
func (l *ConfigLineage) Diff(ref1, ref2 string) {
	space1, unit1 := parseRef(ref1)
	space2, unit2 := parseRef(ref2)

	data1, err := l.getUnitData(space1, unit1)
	if err != nil {
		log.Fatalf("Failed to get %s: %v", ref1, err)
	}

	data2, err := l.getUnitData(space2, unit2)
	if err != nil {
		log.Fatalf("Failed to get %s: %v", ref2, err)
	}

	fmt.Printf("Differences from %s to %s:\n\n", ref1, ref2)

	diffs := compareData("", data1, data2)
	if len(diffs) == 0 {
		fmt.Println("  (no differences)")
		return
	}

	for _, d := range diffs {
		fmt.Printf("  %s: %v → %v\n", d.Path, d.Value, d.Previous)
	}
}

// Helper functions

func (l *ConfigLineage) getLineageChain(space, unit string) ([]LineageNode, error) {
	var chain []LineageNode

	currentSpace := space
	currentUnit := unit

	for {
		spaceObj, err := l.app.Cub.GetSpaceBySlug(currentSpace)
		if err != nil {
			return nil, fmt.Errorf("get space %s: %w", currentSpace, err)
		}

		units, err := l.app.Cub.ListUnits(sdk.ListUnitsParams{
			SpaceID: spaceObj.SpaceID,
			Where:   fmt.Sprintf("Slug = '%s'", currentUnit),
		})
		if err != nil || len(units) == 0 {
			return nil, fmt.Errorf("get unit %s/%s: %w", currentSpace, currentUnit, err)
		}

		u := units[0]

		// Parse unit data
		var data map[string]any
		if err := yaml.Unmarshal([]byte(u.Data), &data); err != nil {
			// Try JSON
			json.Unmarshal([]byte(u.Data), &data)
		}

		node := LineageNode{
			Space:    currentSpace,
			UnitSlug: u.Slug,
			UnitID:   u.UnitID.String(),
			Data:     data,
			Labels:   u.Labels,
		}

		if u.UpstreamUnitID != nil {
			upstreamID := u.UpstreamUnitID.String()
			node.UpstreamID = &upstreamID
		}

		chain = append(chain, node)

		// Follow upstream
		if u.UpstreamUnitID == nil {
			break
		}

		// Find upstream unit
		found := false
		for _, s := range getAllSpaces(l.app) {
			units, err := l.app.Cub.ListUnits(sdk.ListUnitsParams{
				SpaceID: s.SpaceID,
				Where:   fmt.Sprintf("UnitID = '%s'", u.UpstreamUnitID),
			})
			if err == nil && len(units) > 0 {
				currentSpace = s.Slug
				currentUnit = units[0].Slug
				found = true
				break
			}
		}

		if !found {
			break
		}
	}

	return chain, nil
}

func (l *ConfigLineage) getUnitData(space, unit string) (map[string]any, error) {
	spaceObj, err := l.app.Cub.GetSpaceBySlug(space)
	if err != nil {
		return nil, err
	}

	units, err := l.app.Cub.ListUnits(sdk.ListUnitsParams{
		SpaceID: spaceObj.SpaceID,
		Where:   fmt.Sprintf("Slug = '%s'", unit),
	})
	if err != nil || len(units) == 0 {
		return nil, fmt.Errorf("unit not found")
	}

	var data map[string]any
	if err := yaml.Unmarshal([]byte(units[0].Data), &data); err != nil {
		json.Unmarshal([]byte(units[0].Data), &data)
	}

	return data, nil
}

func getAllSpaces(app *sdk.DevOpsApp) []*sdk.Space {
	spaces, _ := app.Cub.ListSpaces()
	return spaces
}

func parseRef(ref string) (string, string) {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		log.Fatalf("Invalid reference: %s (expected space/unit)", ref)
	}
	return parts[0], parts[1]
}

func getValueAtPath(data map[string]any, path string) any {
	parts := strings.Split(path, ".")
	current := any(data)

	for _, part := range parts {
		if m, ok := current.(map[string]any); ok {
			current = m[part]
		} else {
			return nil
		}
	}

	return current
}

func equalValues(a, b any) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func compareData(prefix string, a, b map[string]any) []Override {
	var diffs []Override

	// Check all keys in a
	for k, va := range a {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}

		vb, exists := b[k]
		if !exists {
			diffs = append(diffs, Override{Path: path, Value: nil, Previous: va})
			continue
		}

		// Recurse for nested maps
		if ma, ok := va.(map[string]any); ok {
			if mb, ok := vb.(map[string]any); ok {
				diffs = append(diffs, compareData(path, ma, mb)...)
				continue
			}
		}

		if !equalValues(va, vb) {
			diffs = append(diffs, Override{Path: path, Value: vb, Previous: va})
		}
	}

	// Check for keys only in b
	for k, vb := range b {
		if _, exists := a[k]; !exists {
			path := k
			if prefix != "" {
				path = prefix + "." + k
			}
			diffs = append(diffs, Override{Path: path, Value: vb, Previous: nil})
		}
	}

	return diffs
}
