package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	sdk "github.com/monadic/devops-sdk"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DriftCorrector recommends ConfigHub unit updates to fix drift
type DriftCorrector struct {
	app         *sdk.DevOpsApp
	driftItems  []DriftItem
	corrections []ConfigHubCorrection
}

// ConfigHubCorrection represents a recommended ConfigHub unit update
type ConfigHubCorrection struct {
	UnitName        string                 `json:"unit_name"`
	SpaceID         string                 `json:"space_id"`
	ResourceType    string                 `json:"resource_type"`
	ResourceName    string                 `json:"resource_name"`
	Namespace       string                 `json:"namespace"`
	DriftField      string                 `json:"drift_field"`
	CurrentValue    interface{}            `json:"current_value"`
	ExpectedValue   interface{}            `json:"expected_value"`
	UpdateAction    string                 `json:"update_action"`
	ConfigHubPatch  map[string]interface{} `json:"confighub_patch"`
	CubCommand      string                 `json:"cub_command"`
	YAMLPatch       string                 `json:"yaml_patch"`
}

// NewDriftCorrector creates a new drift corrector
func NewDriftCorrector(app *sdk.DevOpsApp) *DriftCorrector {
	return &DriftCorrector{
		app:         app,
		corrections: []ConfigHubCorrection{},
	}
}

// AnalyzeDrift analyzes drift and generates ConfigHub corrections
func (dc *DriftCorrector) AnalyzeDrift(driftItems []DriftItem) error {
	dc.driftItems = driftItems

	for _, item := range driftItems {
		correction := dc.generateCorrection(item)
		dc.corrections = append(dc.corrections, correction)
	}

	return nil
}

// generateCorrection creates a ConfigHub correction for a drift item
func (dc *DriftCorrector) generateCorrection(drift DriftItem) ConfigHubCorrection {
	correction := ConfigHubCorrection{
		UnitName:      drift.UnitName,
		SpaceID:       drift.SpaceID,
		ResourceType:  drift.ResourceType,
		ResourceName:  drift.ResourceName,
		Namespace:     drift.Namespace,
		DriftField:    drift.Field,
		CurrentValue:  drift.Actual,
		ExpectedValue: drift.Expected,
	}

	// Generate the appropriate ConfigHub update based on resource type
	switch drift.ResourceType {
	case "Deployment":
		correction.UpdateAction = "Update deployment unit in ConfigHub"
		correction.generateDeploymentPatch(drift)
	case "ConfigMap":
		correction.UpdateAction = "Update configmap unit in ConfigHub"
		correction.generateConfigMapPatch(drift)
	case "Service":
		correction.UpdateAction = "Update service unit in ConfigHub"
		correction.generateServicePatch(drift)
	}

	// Generate cub command
	correction.generateCubCommand()

	return correction
}

// generateDeploymentPatch creates a patch for deployment units
func (c *ConfigHubCorrection) generateDeploymentPatch(drift DriftItem) {
	// Parse the drift field to understand what needs updating
	switch drift.Field {
	case "spec.replicas":
		c.ConfigHubPatch = map[string]interface{}{
			"spec": map[string]interface{}{
				"replicas": drift.Expected,
			},
		}
		c.YAMLPatch = fmt.Sprintf(`spec:
  replicas: %v`, drift.Expected)

	case "spec.template.spec.containers[0].resources.requests.cpu":
		c.ConfigHubPatch = map[string]interface{}{
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []map[string]interface{}{
							{
								"name": "container-name", // Would need to get actual name
								"resources": map[string]interface{}{
									"requests": map[string]interface{}{
										"cpu": drift.Expected,
									},
								},
							},
						},
					},
				},
			},
		}
		c.YAMLPatch = fmt.Sprintf(`spec:
  template:
    spec:
      containers:
      - name: container-name
        resources:
          requests:
            cpu: %v`, drift.Expected)
	}
}

// generateConfigMapPatch creates a patch for configmap units
func (c *ConfigHubCorrection) generateConfigMapPatch(drift DriftItem) {
	// Extract the data key from the field (e.g., "data.log_level" -> "log_level")
	parts := strings.Split(drift.Field, ".")
	if len(parts) >= 2 && parts[0] == "data" {
		key := parts[1]
		c.ConfigHubPatch = map[string]interface{}{
			"data": map[string]interface{}{
				key: drift.Expected,
			},
		}
		c.YAMLPatch = fmt.Sprintf(`data:
  %s: "%v"`, key, drift.Expected)
	}
}

// generateServicePatch creates a patch for service units
func (c *ConfigHubCorrection) generateServicePatch(drift DriftItem) {
	// Handle service-specific patches
	if strings.HasPrefix(drift.Field, "spec.ports") {
		c.ConfigHubPatch = map[string]interface{}{
			"spec": map[string]interface{}{
				"ports": drift.Expected,
			},
		}
	}
}

// generateCubCommand creates the cub CLI command to apply the correction
func (c *ConfigHubCorrection) generateCubCommand() {
	unitSlug := fmt.Sprintf("%s-%s", strings.ToLower(c.ResourceType), c.ResourceName)

	// Generate the cub command based on the correction type
	c.CubCommand = fmt.Sprintf(`# Fix drift in %s/%s
# Step 1: View current unit
cub unit get %s --space %s

# Step 2: Edit unit to fix drift (update %s)
cub unit edit %s --space %s
# In editor, update: %s

# Step 3: Apply updated unit
cub unit apply %s --space %s

# Alternative: Use patch command
cub unit update %s --space %s --patch --data '%s'`,
		c.ResourceType, c.ResourceName,
		unitSlug, c.SpaceID,
		c.DriftField,
		unitSlug, c.SpaceID,
		c.YAMLPatch,
		unitSlug, c.SpaceID,
		unitSlug, c.SpaceID, c.getJSONPatch())
}

// getJSONPatch returns JSON patch for the cub command
func (c *ConfigHubCorrection) getJSONPatch() string {
	data, _ := json.Marshal(c.ConfigHubPatch)
	return string(data)
}

// GenerateReport creates a human-readable report of corrections
func (dc *DriftCorrector) GenerateReport() string {
	if len(dc.corrections) == 0 {
		return "✅ No drift detected - all resources match ConfigHub units"
	}

	var report strings.Builder

	report.WriteString("📋 ConfigHub Drift Correction Report\n")
	report.WriteString("=====================================\n\n")
	report.WriteString(fmt.Sprintf("Found %d drift items that need ConfigHub unit updates:\n\n", len(dc.corrections)))

	for i, correction := range dc.corrections {
		report.WriteString(fmt.Sprintf("%d. %s/%s in namespace '%s'\n",
			i+1, correction.ResourceType, correction.ResourceName, correction.Namespace))
		report.WriteString(fmt.Sprintf("   Drift Field: %s\n", correction.DriftField))
		report.WriteString(fmt.Sprintf("   Current (in cluster): %v\n", correction.CurrentValue))
		report.WriteString(fmt.Sprintf("   Expected (in ConfigHub): %v\n", correction.ExpectedValue))
		report.WriteString(fmt.Sprintf("   Action: %s\n", correction.UpdateAction))
		report.WriteString("\n   ConfigHub Commands:\n")
		report.WriteString(fmt.Sprintf("   %s\n\n", strings.ReplaceAll(correction.CubCommand, "\n", "\n   ")))
	}

	report.WriteString("\n🔧 Bulk Correction Option:\n")
	report.WriteString("=====================================\n")
	report.WriteString(dc.generateBulkCorrection())

	return report.String()
}

// generateBulkCorrection creates a bulk update command for all corrections
func (dc *DriftCorrector) generateBulkCorrection() string {
	if len(dc.corrections) == 0 {
		return "No corrections needed"
	}

	// Group corrections by space
	spaceCorrections := make(map[string][]ConfigHubCorrection)
	for _, c := range dc.corrections {
		spaceCorrections[c.SpaceID] = append(spaceCorrections[c.SpaceID], c)
	}

	var bulk strings.Builder
	for spaceID, corrections := range spaceCorrections {
		bulk.WriteString(fmt.Sprintf("# For space %s:\n", spaceID))
		bulk.WriteString("# Option 1: Use push-upgrade to propagate fixes\n")
		bulk.WriteString(fmt.Sprintf("cub unit update --space %s --upgrade --patch\n\n", spaceID))

		bulk.WriteString("# Option 2: Create a correction script\n")
		bulk.WriteString("cat > fix-drift.sh << 'EOF'\n#!/bin/bash\n")
		for _, c := range corrections {
			unitSlug := fmt.Sprintf("%s-%s", strings.ToLower(c.ResourceType), c.ResourceName)
			bulk.WriteString(fmt.Sprintf("echo \"Fixing %s...\"\n", unitSlug))
			bulk.WriteString(fmt.Sprintf("cub unit update %s --space %s --patch --data '%s'\n",
				unitSlug, spaceID, c.getJSONPatch()))
		}
		bulk.WriteString("EOF\n")
		bulk.WriteString("chmod +x fix-drift.sh && ./fix-drift.sh\n\n")
	}

	return bulk.String()
}

// ApplyCorrections applies the recommended corrections to ConfigHub
func (dc *DriftCorrector) ApplyCorrections(autoApply bool) error {
	if !autoApply {
		fmt.Println("📝 Corrections generated but not applied (autoApply=false)")
		fmt.Println("Review the report and apply manually using the provided commands")
		return nil
	}

	if dc.app.Cub == nil {
		return fmt.Errorf("ConfigHub client not initialized")
	}

	fmt.Println("🔧 Applying corrections to ConfigHub...")

	for _, correction := range dc.corrections {
		unitSlug := fmt.Sprintf("%s-%s", strings.ToLower(correction.ResourceType), correction.ResourceName)

		// Get the unit
		spaceID, err := uuid.Parse(correction.SpaceID)
		if err != nil {
			log.Printf("Invalid space ID: %v", err)
			continue
		}

		units, err := dc.app.Cub.ListUnits(spaceID)
		if err != nil {
			log.Printf("Failed to list units: %v", err)
			continue
		}

		// Find the matching unit
		var targetUnit *sdk.Unit
		for _, unit := range units {
			if unit.Slug == unitSlug {
				targetUnit = unit
				break
			}
		}

		if targetUnit == nil {
			log.Printf("Unit %s not found in space %s", unitSlug, correction.SpaceID)
			continue
		}

		// Parse existing data
		var existingData map[string]interface{}
		if err := yaml.Unmarshal([]byte(targetUnit.Data), &existingData); err != nil {
			log.Printf("Failed to parse unit data: %v", err)
			continue
		}

		// Merge the patch
		mergedData := mergeMaps(existingData, correction.ConfigHubPatch)

		// Convert back to YAML
		newData, err := yaml.Marshal(mergedData)
		if err != nil {
			log.Printf("Failed to marshal updated data: %v", err)
			continue
		}

		// Update the unit
		err = dc.app.Cub.UpdateUnit(targetUnit.UnitID, string(newData))
		if err != nil {
			log.Printf("Failed to update unit %s: %v", unitSlug, err)
			continue
		}

		fmt.Printf("✅ Updated ConfigHub unit: %s\n", unitSlug)

		// Apply the unit to propagate changes
		err = dc.app.Cub.ApplyUnit(targetUnit.UnitID)
		if err != nil {
			log.Printf("Warning: Failed to apply unit %s: %v", unitSlug, err)
		}
	}

	// Use push-upgrade to propagate changes downstream
	fmt.Println("📤 Using push-upgrade to propagate changes...")
	for spaceID := range dc.groupBySpace() {
		space, _ := uuid.Parse(spaceID)
		err := dc.app.Cub.BulkPatchUnits(sdk.BulkPatchParams{
			SpaceID: space,
			Upgrade: true, // Push changes downstream
		})
		if err != nil {
			log.Printf("Failed to push-upgrade in space %s: %v", spaceID, err)
		} else {
			fmt.Printf("✅ Push-upgrade completed for space %s\n", spaceID)
		}
	}

	return nil
}

// groupBySpace groups corrections by space ID
func (dc *DriftCorrector) groupBySpace() map[string][]ConfigHubCorrection {
	groups := make(map[string][]ConfigHubCorrection)
	for _, c := range dc.corrections {
		groups[c.SpaceID] = append(groups[c.SpaceID], c)
	}
	return groups
}

// mergeMaps deeply merges two maps
func mergeMaps(base, overlay map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy base
	for k, v := range base {
		result[k] = v
	}

	// Merge overlay
	for k, v := range overlay {
		if existingValue, exists := result[k]; exists {
			// If both are maps, merge recursively
			if existingMap, ok := existingValue.(map[string]interface{}); ok {
				if overlayMap, ok := v.(map[string]interface{}); ok {
					result[k] = mergeMaps(existingMap, overlayMap)
					continue
				}
			}
		}
		result[k] = v
	}

	return result
}

// Example usage function
func ExampleDriftCorrection() {
	// Initialize app
	app, _ := sdk.NewDevOpsApp(sdk.DevOpsAppConfig{
		Name: "drift-corrector",
	})

	corrector := NewDriftCorrector(app)

	// Example drift items
	driftItems := []DriftItem{
		{
			ResourceType: "Deployment",
			ResourceName: "test-app",
			Namespace:    "drift-test",
			Field:        "spec.replicas",
			Expected:     2,
			Actual:       5,
			UnitName:     "deployment-test-app",
			SpaceID:      "abc-123",
		},
		{
			ResourceType: "ConfigMap",
			ResourceName: "app-config",
			Namespace:    "drift-test",
			Field:        "data.log_level",
			Expected:     "info",
			Actual:       "debug",
			UnitName:     "configmap-app-config",
			SpaceID:      "abc-123",
		},
	}

	// Analyze and generate corrections
	corrector.AnalyzeDrift(driftItems)

	// Generate report
	report := corrector.GenerateReport()
	fmt.Println(report)

	// Apply corrections (set to true to actually apply)
	corrector.ApplyCorrections(false)
}

// IntegrateWithDriftDetector integrates the corrector with the main drift detector
func (dc *DriftCorrector) IntegrateWithDriftDetector(detector *DriftDetector) {
	// This would be called from the main drift detector
	// after drift is detected
	dc.AnalyzeDrift(detector.GetDriftItems())

	// Generate and display report
	report := dc.GenerateReport()
	fmt.Println(report)

	// Check if auto-correction is enabled
	if detector.AutoCorrect {
		dc.ApplyCorrections(true)
	}
}

// GenerateConfigHubWorkflow creates a complete ConfigHub workflow
func (dc *DriftCorrector) GenerateConfigHubWorkflow() string {
	var workflow strings.Builder

	workflow.WriteString("🔄 ConfigHub Drift Correction Workflow\n")
	workflow.WriteString("=====================================\n\n")

	workflow.WriteString("## Step 1: Review Drift\n")
	workflow.WriteString("```bash\n")
	workflow.WriteString("# List all units in the space\n")
	workflow.WriteString("cub unit list --space <space-id>\n\n")
	workflow.WriteString("# Check unit status\n")
	workflow.WriteString("cub unit get <unit-name> --space <space-id>\n")
	workflow.WriteString("```\n\n")

	workflow.WriteString("## Step 2: Update ConfigHub Units\n")
	workflow.WriteString("```bash\n")
	for _, c := range dc.corrections {
		unitSlug := fmt.Sprintf("%s-%s", strings.ToLower(c.ResourceType), c.ResourceName)
		workflow.WriteString(fmt.Sprintf("# Fix: %s\n", unitSlug))
		workflow.WriteString(fmt.Sprintf("cub unit edit %s --space %s\n", unitSlug, c.SpaceID))
		workflow.WriteString(fmt.Sprintf("# Update: %s to %v\n\n", c.DriftField, c.ExpectedValue))
	}
	workflow.WriteString("```\n\n")

	workflow.WriteString("## Step 3: Apply Changes\n")
	workflow.WriteString("```bash\n")
	workflow.WriteString("# Apply all updated units\n")
	for _, c := range dc.corrections {
		unitSlug := fmt.Sprintf("%s-%s", strings.ToLower(c.ResourceType), c.ResourceName)
		workflow.WriteString(fmt.Sprintf("cub unit apply %s --space %s\n", unitSlug, c.SpaceID))
	}
	workflow.WriteString("\n# Or use bulk apply\n")
	workflow.WriteString("cub unit apply --all --space <space-id>\n")
	workflow.WriteString("```\n\n")

	workflow.WriteString("## Step 4: Verify Corrections\n")
	workflow.WriteString("```bash\n")
	workflow.WriteString("# Check that Kubernetes resources match ConfigHub\n")
	workflow.WriteString("cub unit list --space <space> --filter deployments\n")
	workflow.WriteString("cub unit list --space <space> --filter configmaps\n")
	workflow.WriteString("```\n\n")

	workflow.WriteString("## Step 5: Push-Upgrade (Optional)\n")
	workflow.WriteString("```bash\n")
	workflow.WriteString("# Propagate changes to downstream environments\n")
	workflow.WriteString("cub unit update --space <downstream-space> --upgrade --patch\n")
	workflow.WriteString("```\n")

	return workflow.String()
}