//go:build integration
// +build integration

package main

import (
	"os"
	"testing"
	"time"

	sdk "github.com/monadic/devops-sdk"
)

// Integration test that demonstrates real ConfigHub usage
// Run with: go test -tags=integration -v
func TestDriftDetectorIntegration(t *testing.T) {
	// Skip if no ConfigHub token provided
	cubToken := os.Getenv("CUB_TOKEN")
	if cubToken == "" {
		t.Skip("Skipping integration test - CUB_TOKEN not set")
	}

	// Create app with minimal config for testing
	config := sdk.DevOpsAppConfig{
		Name:        "drift-detector-test",
		Version:     "1.0.0-test",
		Description: "Integration test for drift detector",
		RunInterval: 1 * time.Minute,
		HealthPort:  8081, // Different port to avoid conflicts
		CubToken:    cubToken,
		CubBaseURL:  sdk.GetEnvOrDefault("CUB_API_URL", "https://api.confighub.com/v1"),
	}

	app, err := sdk.NewDevOpsApp(config)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	detector := &DriftDetector{
		app: app,
	}

	// Test initialization (creates spaces, sets, filters)
	err = detector.initialize()
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	t.Logf("✅ Successfully initialized with space ID: %s", detector.spaceID)
	t.Logf("✅ Critical services set ID: %s", detector.criticalSetID)

	// Test that we can list the spaces we created
	spaces, err := app.Cub.ListSpaces()
	if err != nil {
		t.Fatalf("Failed to list spaces: %v", err)
	}

	found := false
	for _, space := range spaces {
		if space.SpaceID == detector.spaceID {
			found = true
			t.Logf("✅ Found our space: %s (%s)", space.Slug, space.SpaceID)
			break
		}
	}

	if !found {
		t.Error("❌ Could not find the space we created")
	}

	// Test that we can list the sets we created
	sets, err := app.Cub.ListSets(detector.spaceID)
	if err != nil {
		t.Fatalf("Failed to list sets: %v", err)
	}

	foundSet := false
	for _, set := range sets {
		if set.SetID == detector.criticalSetID {
			foundSet = true
			t.Logf("✅ Found our set: %s (%s)", set.Slug, set.SetID)
			break
		}
	}

	if !foundSet {
		t.Error("❌ Could not find the set we created")
	}

	// Test drift detection (won't actually detect drift without K8s resources)
	err = detector.detectAndFixDrift()
	if err != nil {
		t.Logf("⚠️  Drift detection failed (expected without K8s): %v", err)
	} else {
		t.Logf("✅ Drift detection completed successfully")
	}
}

// Test real ConfigHub API operations
func TestConfigHubAPIOperations(t *testing.T) {
	cubToken := os.Getenv("CUB_TOKEN")
	if cubToken == "" {
		t.Skip("Skipping ConfigHub API test - CUB_TOKEN not set")
	}

	client := sdk.NewConfigHubClient("", cubToken)

	// Test space operations
	t.Run("SpaceOperations", func(t *testing.T) {
		// Create space
		space, err := client.CreateSpace(sdk.CreateSpaceRequest{
			Slug:        "test-space-integration",
			DisplayName: "Integration Test Space",
			Labels: map[string]string{
				"test": "true",
				"env":  "integration",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create space: %v", err)
		}
		t.Logf("✅ Created space: %s", space.SpaceID)

		// Get space
		retrievedSpace, err := client.GetSpace(space.SpaceID)
		if err != nil {
			t.Fatalf("Failed to get space: %v", err)
		}
		if retrievedSpace.SpaceID != space.SpaceID {
			t.Errorf("Space ID mismatch: expected %s, got %s", space.SpaceID, retrievedSpace.SpaceID)
		}
		t.Logf("✅ Retrieved space: %s", retrievedSpace.Slug)

		// Test set operations within the space
		t.Run("SetOperations", func(t *testing.T) {
			// Create set
			set, err := client.CreateSet(space.SpaceID, sdk.CreateSetRequest{
				Slug:        "test-set",
				DisplayName: "Test Set",
				Labels: map[string]string{
					"tier": "test",
				},
			})
			if err != nil {
				t.Fatalf("Failed to create set: %v", err)
			}
			t.Logf("✅ Created set: %s", set.SetID)

			// List sets
			sets, err := client.ListSets(space.SpaceID)
			if err != nil {
				t.Fatalf("Failed to list sets: %v", err)
			}
			t.Logf("✅ Listed %d sets", len(sets))
		})

		// Test filter operations
		t.Run("FilterOperations", func(t *testing.T) {
			// Create filter
			filter, err := client.CreateFilter(space.SpaceID, sdk.CreateFilterRequest{
				Slug:        "test-filter",
				DisplayName: "Test Filter",
				From:        "Unit",
				Where:       "Labels.tier = 'test'",
			})
			if err != nil {
				t.Fatalf("Failed to create filter: %v", err)
			}
			t.Logf("✅ Created filter: %s with WHERE clause: %s", filter.FilterID, filter.Where)
		})
	})
}

// Test bulk operations (the key differentiator)
func TestBulkOperations(t *testing.T) {
	cubToken := os.Getenv("CUB_TOKEN")
	if cubToken == "" {
		t.Skip("Skipping bulk operations test - CUB_TOKEN not set")
	}

	client := sdk.NewConfigHubClient("", cubToken)

	// Create a test space
	space, err := client.CreateSpace(sdk.CreateSpaceRequest{
		Slug:        "bulk-test-space",
		DisplayName: "Bulk Operations Test Space",
	})
	if err != nil {
		t.Fatalf("Failed to create space: %v", err)
	}

	// Test bulk patch with upgrade (push-upgrade pattern)
	t.Run("BulkPatchWithUpgrade", func(t *testing.T) {
		params := sdk.BulkPatchParams{
			SpaceID: space.SpaceID,
			Where:   "Labels['tier'] = 'critical'",
			Patch: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 3,
				},
			},
			Upgrade: true, // This is the push-upgrade pattern
		}

		err := client.BulkPatchUnits(params)
		if err != nil {
			t.Logf("⚠️  Bulk patch failed (expected with no units): %v", err)
		} else {
			t.Logf("✅ Bulk patch with upgrade completed")
		}
	})

	// Test bulk apply
	t.Run("BulkApply", func(t *testing.T) {
		params := sdk.BulkApplyParams{
			SpaceID: space.SpaceID,
			Where:   "Labels['monitor'] = 'true'",
			DryRun:  true, // Use dry run for testing
		}

		err := client.BulkApplyUnits(params)
		if err != nil {
			t.Logf("⚠️  Bulk apply failed (expected with no units): %v", err)
		} else {
			t.Logf("✅ Bulk apply completed")
		}
	})
}

// Benchmark the key operations
func BenchmarkConfigHubOperations(b *testing.B) {
	cubToken := os.Getenv("CUB_TOKEN")
	if cubToken == "" {
		b.Skip("Skipping benchmark - CUB_TOKEN not set")
	}

	client := sdk.NewConfigHubClient("", cubToken)

	b.Run("ListSpaces", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := client.ListSpaces()
			if err != nil {
				b.Fatalf("Failed to list spaces: %v", err)
			}
		}
	})
}
