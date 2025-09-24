package main

import (
	"fmt"

	"github.com/google/uuid"
	sdk "github.com/monadic/devops-sdk"
)

// ConfigHubAnalyzer wraps SDK CostAnalyzer for backward compatibility
type ConfigHubAnalyzer struct {
	app          *sdk.DevOpsApp
	spaceID      string
	costAnalyzer *sdk.CostAnalyzer
}

// Legacy type aliases for backward compatibility
type PricingModel = sdk.PricingModel
type UnitCostEstimate = sdk.UnitCostEstimate
type CostBreakdown = sdk.CostBreakdown
type SpaceCostAnalysis = sdk.SpaceCostAnalysis

// DefaultPricing based on AWS EKS m5.large pricing
var DefaultPricing = sdk.DefaultPricing

// NewConfigHubAnalyzer creates analyzer for ConfigHub units
func NewConfigHubAnalyzer(app *sdk.DevOpsApp, spaceID string) *ConfigHubAnalyzer {
	// Parse spaceID as UUID
	spaceUUID, err := uuid.Parse(spaceID)
	if err != nil {
		app.Logger.Printf("⚠️ Invalid space ID %s, using nil UUID: %v", spaceID, err)
		spaceUUID = uuid.Nil
	}

	return &ConfigHubAnalyzer{
		app:          app,
		spaceID:      spaceID,
		costAnalyzer: sdk.NewCostAnalyzer(app, spaceUUID),
	}
}

// AnalyzeSpace analyzes all units in a ConfigHub space using SDK
func (ca *ConfigHubAnalyzer) AnalyzeSpace() (*SpaceCostAnalysis, error) {
	return ca.costAnalyzer.AnalyzeSpace()
}

// analyzeUnit is deprecated - use SDK CostAnalyzer directly
// Kept for backward compatibility, but delegates to SDK
func (ca *ConfigHubAnalyzer) analyzeUnit(unit sdk.Unit) (*UnitCostEstimate, error) {
	// This method is now handled internally by the SDK CostAnalyzer
	// We keep this signature for compatibility but don't implement the logic
	return nil, fmt.Errorf("analyzeUnit is deprecated, use AnalyzeSpace() instead")
}

// analyzeDeployment is deprecated - use SDK CostAnalyzer directly
func (ca *ConfigHubAnalyzer) analyzeDeployment(unit sdk.Unit, manifest map[string]interface{}) (*UnitCostEstimate, error) {
	return nil, fmt.Errorf("analyzeDeployment is deprecated, use SDK CostAnalyzer instead")
}

// analyzeStatefulSet is deprecated - use SDK CostAnalyzer directly
func (ca *ConfigHubAnalyzer) analyzeStatefulSet(unit sdk.Unit, manifest map[string]interface{}) (*UnitCostEstimate, error) {
	return nil, fmt.Errorf("analyzeStatefulSet is deprecated, use SDK CostAnalyzer instead")
}

// analyzeDaemonSet is deprecated - use SDK CostAnalyzer directly
func (ca *ConfigHubAnalyzer) analyzeDaemonSet(unit sdk.Unit, manifest map[string]interface{}) (*UnitCostEstimate, error) {
	return nil, fmt.Errorf("analyzeDaemonSet is deprecated, use SDK CostAnalyzer instead")
}

// extractContainerResources is deprecated - use SDK CostAnalyzer directly
func (ca *ConfigHubAnalyzer) extractContainerResources(container map[string]interface{}, estimate *UnitCostEstimate) {
	// This functionality is now handled by the SDK CostAnalyzer
	ca.app.Logger.Printf("⚠️ extractContainerResources is deprecated, use SDK CostAnalyzer instead")
}

// extractStorageResources is deprecated - use SDK CostAnalyzer directly
func (ca *ConfigHubAnalyzer) extractStorageResources(vct map[string]interface{}, estimate *UnitCostEstimate) {
	// This functionality is now handled by the SDK CostAnalyzer
	ca.app.Logger.Printf("⚠️ extractStorageResources is deprecated, use SDK CostAnalyzer instead")
}

// calculateMonthlyCost is deprecated - use SDK CostAnalyzer directly
// The SDK CostAnalyzer now handles all cost calculations internally
func (ca *ConfigHubAnalyzer) calculateMonthlyCost(estimate *UnitCostEstimate) float64 {
	ca.app.Logger.Printf("⚠️ calculateMonthlyCost is deprecated, SDK handles this internally")
	return 0.0
}

// AnalyzeHierarchy analyzes a full environment hierarchy using SDK
func (ca *ConfigHubAnalyzer) AnalyzeHierarchy(baseSpaceSlug string) (*SpaceCostAnalysis, error) {
	return ca.costAnalyzer.AnalyzeHierarchy(baseSpaceSlug)
}

// GenerateReport creates a human-readable cost report using SDK
func (ca *ConfigHubAnalyzer) GenerateReport(analysis *SpaceCostAnalysis) string {
	return ca.costAnalyzer.GenerateReport(analysis)
}

// StoreAnalysisInConfigHub stores cost analysis as ConfigHub annotations using SDK
func (ca *ConfigHubAnalyzer) StoreAnalysisInConfigHub(analysis *SpaceCostAnalysis) error {
	return ca.costAnalyzer.StoreAnalysisInConfigHub(analysis)
}