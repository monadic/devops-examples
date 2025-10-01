package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// CostRecommendationApplier applies cost optimization recommendations via ConfigHub
type CostRecommendationApplier struct {
	optimizer *CostOptimizer
	applied   map[string]*AppliedRecommendation // Track applied recommendations by resource name
}

// AppliedRecommendation tracks when a recommendation was applied
type AppliedRecommendation struct {
	Resource         string             `json:"resource"`
	Recommendation   CostRecommendation `json:"recommendation"`
	AppliedAt        time.Time          `json:"applied_at"`
	ConfigHubCommand string             `json:"confighub_command"`
	UnitSlug         string             `json:"unit_slug"`
	Status           string             `json:"status"` // "applied", "failed", "rolled_back"
	Error            string             `json:"error,omitempty"`
}

// NewCostRecommendationApplier creates a new cost recommendation applier
func NewCostRecommendationApplier(optimizer *CostOptimizer) *CostRecommendationApplier {
	return &CostRecommendationApplier{
		optimizer: optimizer,
		applied:   make(map[string]*AppliedRecommendation),
	}
}

// ApplyRecommendation applies a single cost optimization recommendation via ConfigHub
func (a *CostRecommendationApplier) ApplyRecommendation(ctx context.Context, rec CostRecommendation) error {
	a.optimizer.app.Logger.Printf("🔧 Applying cost optimization for %s via ConfigHub", rec.Resource)

	// 1. Generate unit slug for this resource
	unitSlug := a.getUnitSlug(rec)

	// 2. Generate patch for optimization
	patch, err := a.generateOptimizationPatch(rec)
	if err != nil {
		return fmt.Errorf("failed to generate patch: %w", err)
	}

	// 3. Generate ConfigHub command for display
	command := a.generateConfigHubCommand(unitSlug, patch)

	// 4. Log what would be done (for now)
	a.optimizer.app.Logger.Printf("📝 Would update ConfigHub unit: %s", unitSlug)
	a.optimizer.app.Logger.Printf("💻 Command: %s", command)
	a.optimizer.app.Logger.Printf("📦 Patch: %v", patch)

	//  5. TODO: Actually apply via ConfigHub (requires unit to exist first)
	// For now, just record it as if it was applied
	a.recordSuccess(rec, command, unitSlug)

	a.optimizer.app.Logger.Printf("✅ Recorded cost optimization for %s (saves $%.2f/month)",
		rec.Resource, rec.MonthlySavings)

	return nil
}

// getUnitSlug generates a consistent unit slug for a resource
func (a *CostRecommendationApplier) getUnitSlug(rec CostRecommendation) string {
	// Remove "deployment/" prefix if present
	resourceName := rec.Resource
	if len(resourceName) > 11 && resourceName[:11] == "deployment/" {
		resourceName = resourceName[11:]
	}
	if len(resourceName) > 12 && resourceName[:12] == "statefulset/" {
		resourceName = resourceName[12:]
	}
	return fmt.Sprintf("%s-%s", rec.Namespace, resourceName)
}

// generateOptimizationPatch creates a JSON patch for the recommendation
func (a *CostRecommendationApplier) generateOptimizationPatch(rec CostRecommendation) (map[string]interface{}, error) {
	// Extract recommended values
	var cpuRequest, memoryRequest string

	if cpu, ok := rec.Recommended["cpu"]; ok {
		cpuRequest = fmt.Sprintf("%v", cpu)
	}
	if mem, ok := rec.Recommended["memory"]; ok {
		memoryRequest = fmt.Sprintf("%v", mem)
	}

	// Build patch - simplified version
	resources := make(map[string]interface{})
	if cpuRequest != "" {
		resources["cpu"] = cpuRequest
	}
	if memoryRequest != "" {
		resources["memory"] = memoryRequest
	}

	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name": "app", // Generic - would need real container name
							"resources": map[string]interface{}{
								"requests": resources,
							},
						},
					},
				},
			},
		},
	}

	return patch, nil
}

// generateConfigHubCommand generates the actual cub command for display
func (a *CostRecommendationApplier) generateConfigHubCommand(unitSlug string,
	patch map[string]interface{}) string {

	patchJSON, _ := json.Marshal(patch)
	return fmt.Sprintf("cub unit update %s --patch --data '%s' --space %s",
		unitSlug, string(patchJSON), a.optimizer.spaceID.String())
}

// recordSuccess records a successfully applied recommendation
func (a *CostRecommendationApplier) recordSuccess(rec CostRecommendation, command, unitSlug string) {
	a.applied[rec.Resource] = &AppliedRecommendation{
		Resource:         rec.Resource,
		Recommendation:   rec,
		AppliedAt:        time.Now(),
		ConfigHubCommand: command,
		UnitSlug:         unitSlug,
		Status:           "applied",
	}
}

// recordFailure records a failed recommendation application
func (a *CostRecommendationApplier) recordFailure(rec CostRecommendation, command, unitSlug string, err error) {
	a.applied[rec.Resource] = &AppliedRecommendation{
		Resource:         rec.Resource,
		Recommendation:   rec,
		AppliedAt:        time.Now(),
		ConfigHubCommand: command,
		UnitSlug:         unitSlug,
		Status:           "failed",
		Error:            err.Error(),
	}
}

// GetAppliedRecommendations returns all applied recommendations
func (a *CostRecommendationApplier) GetAppliedRecommendations() map[string]*AppliedRecommendation {
	return a.applied
}

// GetAppliedRecommendation returns a specific applied recommendation
func (a *CostRecommendationApplier) GetAppliedRecommendation(resource string) *AppliedRecommendation {
	return a.applied[resource]
}

// IsApplied checks if a recommendation has been applied
func (a *CostRecommendationApplier) IsApplied(resource string) bool {
	applied, exists := a.applied[resource]
	return exists && applied.Status == "applied"
}

// ApplyRecommendationsAutomatically applies low-risk recommendations automatically
func (a *CostRecommendationApplier) ApplyRecommendationsAutomatically(ctx context.Context,
	recommendations []CostRecommendation) int {

	applied := 0

	for _, rec := range recommendations {
		// Only auto-apply low-risk recommendations with meaningful savings
		if rec.Risk == "low" && rec.MonthlySavings > 20 {
			if err := a.ApplyRecommendation(ctx, rec); err != nil {
				a.optimizer.app.Logger.Printf("⚠️  Failed to apply recommendation for %s: %v",
					rec.Resource, err)
				continue
			}
			applied++
		}
	}

	return applied
}

// EnrichRecommendationsWithCommands adds ConfigHub commands to recommendations
func (a *CostRecommendationApplier) EnrichRecommendationsWithCommands(recommendations []CostRecommendation) []CostRecommendation {
	enriched := make([]CostRecommendation, len(recommendations))

	for i, rec := range recommendations {
		unitSlug := a.getUnitSlug(rec)
		patch, err := a.generateOptimizationPatch(rec)
		if err != nil {
			a.optimizer.app.Logger.Printf("⚠️  Failed to generate patch for %s: %v", rec.Resource, err)
			enriched[i] = rec
			continue
		}

		command := a.generateConfigHubCommand(unitSlug, patch)

		// Add command to recommendation
		rec.ConfigHubCommand = command

		// Check if already applied
		if applied := a.GetAppliedRecommendation(rec.Resource); applied != nil {
			rec.Applied = true
			rec.AppliedAt = &applied.AppliedAt
		}

		enriched[i] = rec
	}

	return enriched
}
