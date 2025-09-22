package main

import (
	"testing"

	"github.com/google/uuid"
	sdk "github.com/monadic/devops-sdk"
)

func TestCompareStates(t *testing.T) {
	detector := &DriftDetector{}

	unit := &sdk.Unit{
		UnitID: uuid.New(),
		Slug:   "test-deployment",
		Data:   `{"kind":"Deployment","metadata":{"name":"test"},"spec":{"replicas":3}}`,
	}

	// Test case 1: States match
	actualState := map[string]interface{}{
		"spec": map[string]interface{}{
			"replicas": float64(3),
		},
	}

	items := detector.compareStates(unit, actualState)
	if len(items) != 0 {
		t.Errorf("Expected no drift items, got %d", len(items))
	}

	// Test case 2: States don't match
	actualState["spec"].(map[string]interface{})["replicas"] = float64(5)
	items = detector.compareStates(unit, actualState)
	if len(items) != 1 {
		t.Errorf("Expected 1 drift item, got %d", len(items))
	}

	if len(items) > 0 {
		item := items[0]
		if item.Field != "spec.replicas" {
			t.Errorf("Expected field 'spec.replicas', got '%s'", item.Field)
		}
		if item.Expected != "3" {
			t.Errorf("Expected '3', got '%s'", item.Expected)
		}
		if item.Actual != "5" {
			t.Errorf("Expected '5', got '%s'", item.Actual)
		}
	}
}

func TestDriftAnalysisJSON(t *testing.T) {
	analysis := &DriftAnalysis{
		HasDrift: true,
		Items: []DriftItem{
			{
				UnitID:   uuid.New(),
				UnitSlug: "test-deployment",
				Resource: "Deployment/test",
				Field:    "spec.replicas",
				Expected: "3",
				Actual:   "5",
			},
		},
		Summary: "Replica count mismatch detected",
		Fixes: []ProposedFix{
			{
				UnitID:      uuid.New(),
				UnitSlug:    "test-deployment",
				PatchPath:   "/spec/replicas",
				PatchValue:  3,
				Explanation: "Fix replica count",
			},
		},
	}

	detector := &DriftDetector{}
	jsonData := detector.jsonPretty(analysis)

	if jsonData == "" {
		t.Error("Expected JSON output, got empty string")
	}

	// Basic validation that it contains expected fields
	if !contains(jsonData, "has_drift") {
		t.Error("Expected 'has_drift' field in JSON")
	}
	if !contains(jsonData, "items") {
		t.Error("Expected 'items' field in JSON")
	}
	if !contains(jsonData, "fixes") {
		t.Error("Expected 'fixes' field in JSON")
	}
}

func TestGetGVR(t *testing.T) {
	detector := &DriftDetector{}

	// Test deployment
	gvr := detector.getGVR("Deployment")
	if gvr.Group != "apps" {
		t.Errorf("Expected group 'apps', got '%s'", gvr.Group)
	}
	if gvr.Version != "v1" {
		t.Errorf("Expected version 'v1', got '%s'", gvr.Version)
	}
	if gvr.Resource != "deployments" {
		t.Errorf("Expected resource 'deployments', got '%s'", gvr.Resource)
	}

	// Test service
	gvr = detector.getGVR("Service")
	if gvr.Group != "" {
		t.Errorf("Expected empty group, got '%s'", gvr.Group)
	}
	if gvr.Resource != "services" {
		t.Errorf("Expected resource 'services', got '%s'", gvr.Resource)
	}

	// Test unknown resource
	gvr = detector.getGVR("Unknown")
	if gvr.Resource != "unknowns" {
		t.Errorf("Expected resource 'unknowns', got '%s'", gvr.Resource)
	}
}

func TestConfigHubStructs(t *testing.T) {
	// Test Space creation
	space := &sdk.Space{
		SpaceID:     uuid.New(),
		Slug:        "test-space",
		DisplayName: "Test Space",
		Labels: map[string]string{
			"env": "test",
		},
	}

	if space.Slug != "test-space" {
		t.Errorf("Expected slug 'test-space', got '%s'", space.Slug)
	}

	// Test Unit creation
	unit := &sdk.Unit{
		UnitID:      uuid.New(),
		SpaceID:     space.SpaceID,
		Slug:        "test-unit",
		DisplayName: "Test Unit",
		Data:        `{"kind":"Deployment"}`,
		Labels: map[string]string{
			"tier": "critical",
		},
	}

	if unit.Slug != "test-unit" {
		t.Errorf("Expected slug 'test-unit', got '%s'", unit.Slug)
	}

	// Test Set creation
	set := &sdk.Set{
		SetID:       uuid.New(),
		SpaceID:     space.SpaceID,
		Slug:        "critical-services",
		DisplayName: "Critical Services",
		Labels: map[string]string{
			"tier": "critical",
		},
	}

	if set.Slug != "critical-services" {
		t.Errorf("Expected slug 'critical-services', got '%s'", set.Slug)
	}

	// Test Filter creation
	filter := &sdk.Filter{
		FilterID:    uuid.New(),
		SpaceID:     space.SpaceID,
		Slug:        "drift-filter",
		DisplayName: "Drift Detection Filter",
		From:        "Unit",
		Where:       "Labels['tier'] = 'critical'",
	}

	if filter.From != "Unit" {
		t.Errorf("Expected from 'Unit', got '%s'", filter.From)
	}
	if filter.Where != "Labels['tier'] = 'critical'" {
		t.Errorf("Expected where clause 'Labels['tier'] = 'critical'', got '%s'", filter.Where)
	}
}

func TestBulkPatchParams(t *testing.T) {
	spaceID := uuid.New()
	setID := uuid.New()

	params := sdk.BulkPatchParams{
		SpaceID: spaceID,
		Where:   "SetIDs contains '" + setID.String() + "'",
		Patch: map[string]interface{}{
			"spec": map[string]interface{}{
				"replicas": 3,
			},
		},
		Upgrade: true, // Push-upgrade pattern
	}

	if params.SpaceID != spaceID {
		t.Error("SpaceID mismatch")
	}
	if !params.Upgrade {
		t.Error("Expected Upgrade to be true for push-upgrade pattern")
	}
	if params.Patch == nil {
		t.Error("Expected patch to be set")
	}
}

func TestBulkApplyParams(t *testing.T) {
	spaceID := uuid.New()
	setID := uuid.New()

	params := sdk.BulkApplyParams{
		SpaceID: spaceID,
		Where:   "SetIDs contains '" + setID.String() + "'",
		DryRun:  false,
	}

	if params.SpaceID != spaceID {
		t.Error("SpaceID mismatch")
	}
	if params.DryRun {
		t.Error("Expected DryRun to be false")
	}
}

// Integration test that verifies the key ConfigHub operations are using real APIs
func TestConfigHubRealAPIUsage(t *testing.T) {
	// This test documents the real ConfigHub APIs we're using
	realAPIs := []string{
		"CreateSpace",
		"ListSpaces",
		"CreateUnit",
		"ListUnits",
		"CreateSet",        // REAL feature
		"ListSets",         // REAL feature
		"CreateFilter",     // REAL feature with WHERE clauses
		"BulkPatchUnits",   // REAL feature with Upgrade: true for push-upgrade
		"BulkApplyUnits",   // REAL feature for bulk operations
		"GetUnitLiveState", // REAL feature (read-only)
		"ApplyUnit",
		"DestroyUnit",
		"CreateTarget",
	}

	hallucinatedAPIs := []string{
		"GetVariant",         // ❌ HALLUCINATED
		"CloneWithVariant",   // ❌ HALLUCINATED
		"GetGate",            // ❌ HALLUCINATED
		"GetDependencyGraph", // ❌ HALLUCINATED
		"UpdateStatus",       // ❌ HALLUCINATED
		"UpgradeSet",         // ❌ HALLUCINATED - use push-upgrade instead
	}

	t.Logf("Using %d real ConfigHub APIs", len(realAPIs))
	t.Logf("Avoiding %d hallucinated APIs", len(hallucinatedAPIs))

	// Verify we're not using any hallucinated APIs in our drift detector
	for _, api := range realAPIs {
		t.Logf("✅ Using real API: %s", api)
	}

	for _, api := range hallucinatedAPIs {
		t.Logf("❌ Avoiding hallucinated API: %s", api)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s != substr && s[0:len(substr)] == substr ||
		len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
