package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	sdk "github.com/monadic/devops-sdk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

type DriftDetector struct {
	app           *sdk.DevOpsApp
	spaceID       uuid.UUID
	criticalSetID uuid.UUID
	targetID      uuid.UUID
}

type DriftAnalysis struct {
	HasDrift bool          `json:"has_drift"`
	Items    []DriftItem   `json:"items"`
	Summary  string        `json:"summary"`
	Fixes    []ProposedFix `json:"fixes"`
}

type DriftItem struct {
	UnitID   uuid.UUID `json:"unit_id"`
	UnitSlug string    `json:"unit_slug"`
	Resource string    `json:"resource"`
	Field    string    `json:"field"`
	Expected string    `json:"expected"`
	Actual   string    `json:"actual"`
}

type ProposedFix struct {
	UnitID      uuid.UUID   `json:"unit_id"`
	UnitSlug    string      `json:"unit_slug"`
	PatchPath   string      `json:"patch_path"`
	PatchValue  interface{} `json:"patch_value"`
	Explanation string      `json:"explanation"`
}

func main() {
	// Check if demo mode was requested
	if runDemoMode() {
		return
	}

	config := sdk.DevOpsAppConfig{
		Name:         "drift-detector",
		Version:      "2.0.0",
		Description:  "Detects and fixes Kubernetes configuration drift using ConfigHub Sets and Filters",
		RunInterval:  5 * time.Minute,
		HealthPort:   8080,
		ClaudeAPIKey: os.Getenv("CLAUDE_API_KEY"),
		CubToken:     os.Getenv("CUB_TOKEN"),
		CubBaseURL:   sdk.GetEnvOrDefault("CUB_API_URL", "https://api.confighub.com/v1"),
	}

	app, err := sdk.NewDevOpsApp(config)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	detector := &DriftDetector{
		app: app,
	}

	// Initialize ConfigHub resources on startup
	if err := detector.initialize(); err != nil {
		log.Fatalf("Failed to initialize ConfigHub resources: %v", err)
	}

	// Run drift detection using Kubernetes informers (event-driven)
	detector.RunWithInformers()
}

func (d *DriftDetector) initialize() error {
	d.app.Logger.Println("Initializing ConfigHub resources...")

	// Get or create space
	spaceName := sdk.GetEnvOrDefault("CUB_SPACE", "drift-detector")
	spaces, err := d.app.Cub.ListSpaces()
	if err != nil {
		return fmt.Errorf("list spaces: %w", err)
	}

	var space *sdk.Space
	for _, s := range spaces {
		if s.Slug == spaceName {
			space = s
			break
		}
	}

	if space == nil {
		// Create new space
		space, err = d.app.Cub.CreateSpace(sdk.CreateSpaceRequest{
			Slug:        spaceName,
			DisplayName: "Drift Detector Space",
			Labels: map[string]string{
				"app":  "drift-detector",
				"team": "devops",
			},
		})
		if err != nil {
			return fmt.Errorf("create space: %w", err)
		}
		d.app.Logger.Printf("Created new space: %s", space.SpaceID)
	} else {
		d.app.Logger.Printf("Using existing space: %s", space.SpaceID)
	}
	d.spaceID = space.SpaceID

	// Create or get critical services set
	sets, err := d.app.Cub.ListSets(d.spaceID)
	if err != nil {
		return fmt.Errorf("list sets: %w", err)
	}

	var criticalSet *sdk.Set
	for _, s := range sets {
		if s.Slug == "critical-services" {
			criticalSet = s
			break
		}
	}

	if criticalSet == nil {
		criticalSet, err = d.app.Cub.CreateSet(d.spaceID, sdk.CreateSetRequest{
			Slug:        "critical-services",
			DisplayName: "Critical Services",
			Labels: map[string]string{
				"tier":     "critical",
				"monitor":  "true",
				"auto-fix": "true",
			},
		})
		if err != nil {
			return fmt.Errorf("create set: %w", err)
		}
		d.app.Logger.Printf("Created critical services set: %s", criticalSet.SetID)
	} else {
		d.app.Logger.Printf("Using existing critical services set: %s", criticalSet.SetID)
	}
	d.criticalSetID = criticalSet.SetID

	// Create or get Kubernetes target
	targetSlug := sdk.GetEnvOrDefault("TARGET", "kubernetes-cluster")
	target, err := d.app.Cub.CreateTarget(sdk.Target{
		Slug:        targetSlug,
		DisplayName: "Kubernetes Cluster",
		TargetType:  "kubernetes",
		Config: map[string]string{
			"namespace": sdk.GetEnvOrDefault("NAMESPACE", "default"),
			"context":   sdk.GetEnvOrDefault("K8S_CONTEXT", ""),
		},
	})
	if err != nil {
		// Try to get existing
		d.app.Logger.Printf("Target might already exist: %v", err)
		// For now, use a placeholder UUID
		d.targetID = uuid.New()
	} else {
		d.targetID = target.TargetID
	}

	// Create filter for critical services
	filter, err := d.app.Cub.CreateFilter(d.spaceID, sdk.CreateFilterRequest{
		Slug:        "drift-detection-filter",
		DisplayName: "Drift Detection Filter",
		From:        "Unit",
		Where:       fmt.Sprintf("SetIDs contains '%s' AND Labels['monitor'] = 'true'", d.criticalSetID),
		Select:      []string{"UnitID", "Slug", "Data", "Labels"},
	})
	if err != nil {
		d.app.Logger.Printf("Filter might already exist: %v", err)
	} else {
		d.app.Logger.Printf("Created filter: %s", filter.FilterID)
	}

	return nil
}

func (d *DriftDetector) detectAndFixDrift() error {
	d.app.Logger.Println("Detecting drift using Sets and Filters...")

	// 1. Get units using filter for critical services
	filter, err := d.getOrCreateFilter()
	if err != nil {
		return fmt.Errorf("get filter: %w", err)
	}

	units, err := d.app.Cub.ListUnits(sdk.ListUnitsParams{
		SpaceID:  d.spaceID,
		FilterID: &filter.FilterID,
	})
	if err != nil {
		return fmt.Errorf("list units with filter: %w", err)
	}

	d.app.Logger.Printf("Found %d critical units to monitor", len(units))

	// 2. Check each unit's live state
	var driftItems []DriftItem
	for _, unit := range units {
		liveState, err := d.app.Cub.GetUnitLiveState(d.spaceID, unit.UnitID)
		if err != nil {
			d.app.Logger.Printf("Failed to get live state for %s: %v", unit.Slug, err)
			continue
		}

		if liveState.DriftDetected {
			// Get actual state from Kubernetes
			actualState, err := d.getActualK8sState(unit)
			if err != nil {
				d.app.Logger.Printf("Failed to get actual state for %s: %v", unit.Slug, err)
				continue
			}

			// Compare and identify drift
			items := d.compareStates(unit, actualState)
			driftItems = append(driftItems, items...)
		}
	}

	if len(driftItems) == 0 {
		d.app.Logger.Println("No drift detected")
		return nil
	}

	// 3. Analyze drift with Claude if available
	analysis := &DriftAnalysis{
		HasDrift: true,
		Items:    driftItems,
		Summary:  fmt.Sprintf("Detected %d drift items across %d units", len(driftItems), len(units)),
	}

	if d.app.Claude != nil {
		enhancedAnalysis, err := d.analyzeWithClaude(driftItems, units)
		if err != nil {
			d.app.Logger.Printf("Claude analysis failed: %v", err)
		} else {
			analysis = enhancedAnalysis
		}
	}

	// 4. Report drift
	d.reportDrift(analysis)

	// 5. Auto-fix using bulk operations if enabled
	if sdk.GetEnvBool("AUTO_FIX", false) && len(analysis.Fixes) > 0 {
		if err := d.applyFixes(analysis); err != nil {
			d.app.Logger.Printf("Failed to apply fixes: %v", err)
		}
	}

	return nil
}

func (d *DriftDetector) getOrCreateFilter() (*sdk.Filter, error) {
	// In production, would cache this or get by ID
	return d.app.Cub.CreateFilter(d.spaceID, sdk.CreateFilterRequest{
		Slug:        "critical-drift-filter",
		DisplayName: "Critical Services Drift Filter",
		From:        "Unit",
		Where:       fmt.Sprintf("SetIDs contains '%s'", d.criticalSetID),
	})
}

func (d *DriftDetector) getActualK8sState(unit *sdk.Unit) (map[string]interface{}, error) {
	// Parse unit data to understand what resource to check
	var unitData map[string]interface{}
	if err := json.Unmarshal([]byte(unit.Data), &unitData); err != nil {
		return nil, fmt.Errorf("parse unit data: %w", err)
	}

	// Extract resource type and name
	resourceType := unitData["kind"].(string)
	metadata := unitData["metadata"].(map[string]interface{})
	name := metadata["name"].(string)
	namespace := sdk.GetEnvOrDefault("NAMESPACE", "default")

	// Use Kubernetes client to get the resource
	switch strings.ToLower(resourceType) {
	case "deployment":
		deployment, err := d.app.K8s.Clientset.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"spec": map[string]interface{}{
				"replicas": *deployment.Spec.Replicas,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

func (d *DriftDetector) getGVR(kind string) schema.GroupVersionResource {
	// Map common resource types to GVR
	switch strings.ToLower(kind) {
	case "deployment":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	case "service":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	case "configmap":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	case "secret":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
	default:
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: strings.ToLower(kind) + "s"}
	}
}

func (d *DriftDetector) compareStates(unit *sdk.Unit, actualState map[string]interface{}) []DriftItem {
	var items []DriftItem

	// Parse expected state from unit
	var expectedState map[string]interface{}
	if err := json.Unmarshal([]byte(unit.Data), &expectedState); err != nil {
		d.app.Logger.Printf("Failed to parse unit data: %v", err)
		return items
	}

	// Simple comparison - check key fields
	if expectedSpec, ok := expectedState["spec"].(map[string]interface{}); ok {
		if actualSpec, ok := actualState["spec"].(map[string]interface{}); ok {
			// Check replicas for deployments
			if expectedReplicas, ok := expectedSpec["replicas"].(float64); ok {
				if actualReplicas, ok := actualSpec["replicas"].(float64); ok {
					if expectedReplicas != actualReplicas {
						items = append(items, DriftItem{
							UnitID:   unit.UnitID,
							UnitSlug: unit.Slug,
							Resource: fmt.Sprintf("%s/%s", expectedState["kind"], expectedState["metadata"].(map[string]interface{})["name"]),
							Field:    "spec.replicas",
							Expected: fmt.Sprintf("%.0f", expectedReplicas),
							Actual:   fmt.Sprintf("%.0f", actualReplicas),
						})
					}
				}
			}
		}
	}

	return items
}

func (d *DriftDetector) analyzeWithClaude(driftItems []DriftItem, units []*sdk.Unit) (*DriftAnalysis, error) {
	prompt := fmt.Sprintf(`Analyze this Kubernetes configuration drift and suggest fixes.

Drift Items:
%s

Return JSON with this structure:
{
  "has_drift": true,
  "items": [...existing items...],
  "summary": "Clear explanation of the drift and its impact",
  "fixes": [
    {
      "unit_id": "uuid",
      "unit_slug": "unit-name",
      "patch_path": "/spec/replicas",
      "patch_value": 3,
      "explanation": "Why this fix is needed"
    }
  ]
}`,
		d.jsonPretty(driftItems))

	response, err := d.app.Claude.Complete(prompt)
	if err != nil {
		return nil, err
	}

	var analysis DriftAnalysis
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		return nil, fmt.Errorf("parse Claude response: %w", err)
	}

	return &analysis, nil
}

func (d *DriftDetector) reportDrift(analysis *DriftAnalysis) {
	d.app.Logger.Println("=== DRIFT DETECTION REPORT ===")
	d.app.Logger.Printf("Summary: %s", analysis.Summary)
	d.app.Logger.Printf("Total Drift Items: %d", len(analysis.Items))

	for _, item := range analysis.Items {
		d.app.Logger.Printf("  ⚠️  %s [%s]: %s expected=%s, actual=%s",
			item.UnitSlug, item.Resource, item.Field, item.Expected, item.Actual)
	}

	if len(analysis.Fixes) > 0 {
		d.app.Logger.Println("Proposed Fixes:")
		for _, fix := range analysis.Fixes {
			d.app.Logger.Printf("  ✅ %s: %s", fix.UnitSlug, fix.Explanation)
		}
	}
}

func (d *DriftDetector) applyFixes(analysis *DriftAnalysis) error {
	d.app.Logger.Println("Applying fixes using push-upgrade pattern...")

	// Group fixes by unit
	fixesByUnit := make(map[uuid.UUID][]ProposedFix)
	for _, fix := range analysis.Fixes {
		fixesByUnit[fix.UnitID] = append(fixesByUnit[fix.UnitID], fix)
	}

	// Apply fixes using bulk patch with upgrade
	for unitID, fixes := range fixesByUnit {
		patch := make(map[string]interface{})
		for _, fix := range fixes {
			// Build patch document
			pathParts := strings.Split(fix.PatchPath, "/")
			current := patch
			for _, part := range pathParts[1 : len(pathParts)-1] {
				if _, ok := current[part]; !ok {
					current[part] = make(map[string]interface{})
				}
				current = current[part].(map[string]interface{})
			}
			lastPart := pathParts[len(pathParts)-1]
			current[lastPart] = fix.PatchValue
		}

		// Apply patch with push-upgrade
		err := d.app.Cub.BulkPatchUnits(sdk.BulkPatchParams{
			SpaceID: d.spaceID,
			Where:   fmt.Sprintf("UnitID = '%s'", unitID),
			Patch:   patch,
			Upgrade: true, // Push changes downstream
		})
		if err != nil {
			d.app.Logger.Printf("Failed to patch unit %s: %v", unitID, err)
			continue
		}

		// Apply the fixed unit to Kubernetes
		err = d.app.Cub.ApplyUnit(d.spaceID, unitID)
		if err != nil {
			d.app.Logger.Printf("Failed to apply unit %s: %v", unitID, err)
			continue
		}

		d.app.Logger.Printf("Successfully applied fix to unit %s", unitID)
	}

	// Bulk apply all units in the critical set
	err := d.app.Cub.BulkApplyUnits(sdk.BulkApplyParams{
		SpaceID: d.spaceID,
		Where:   fmt.Sprintf("SetIDs contains '%s'", d.criticalSetID),
		DryRun:  false,
	})
	if err != nil {
		return fmt.Errorf("bulk apply critical services: %w", err)
	}

	d.app.Logger.Printf("Applied fixes to %d units", len(fixesByUnit))
	return nil
}

// RunWithInformers implements event-driven architecture using Kubernetes informers
func (d *DriftDetector) RunWithInformers() error {
	d.app.Logger.Printf("%s v%s started with informers", d.app.Name, d.app.Version)

	// Create informer factory
	informerFactory := informers.NewSharedInformerFactory(d.app.K8s.Clientset, time.Minute*10)

	// Register handlers for relevant resources
	deploymentInformer := informerFactory.Apps().V1().Deployments().Informer()
	deploymentInformer.AddEventHandler(&ResourceEventHandler{
		detector: d,
	})

	serviceInformer := informerFactory.Core().V1().Services().Informer()
	serviceInformer.AddEventHandler(&ResourceEventHandler{
		detector: d,
	})

	configMapInformer := informerFactory.Core().V1().ConfigMaps().Informer()
	configMapInformer.AddEventHandler(&ResourceEventHandler{
		detector: d,
	})

	// Start informers
	stopCh := make(chan struct{})
	defer close(stopCh)
	informerFactory.Start(stopCh)

	// Wait for caches to sync
	if !cache.WaitForCacheSync(stopCh, deploymentInformer.HasSynced, serviceInformer.HasSynced, configMapInformer.HasSynced) {
		return fmt.Errorf("failed to sync caches")
	}

	d.app.Logger.Println("Informers started, watching for changes...")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run initial detection
	if err := d.detectAndFixDrift(); err != nil {
		d.app.Logger.Printf("Initial detection error: %v", err)
	}

	// Wait for shutdown signal
	<-sigChan
	d.app.Logger.Println("Received shutdown signal")
	return nil
}

type ResourceEventHandler struct {
	detector *DriftDetector
}

func (h *ResourceEventHandler) OnAdd(obj interface{}, isInInitialList bool) {
	if !isInInitialList {
		h.detector.app.Logger.Printf("Resource added, triggering drift detection...")
		if err := h.detector.detectAndFixDrift(); err != nil {
			h.detector.app.Logger.Printf("Handler error: %v", err)
		}
	}
}

func (h *ResourceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	h.detector.app.Logger.Printf("Resource updated, triggering drift detection...")
	if err := h.detector.detectAndFixDrift(); err != nil {
		h.detector.app.Logger.Printf("Handler error: %v", err)
	}
}

func (h *ResourceEventHandler) OnDelete(obj interface{}) {
	h.detector.app.Logger.Printf("Resource deleted, triggering drift detection...")
	if err := h.detector.detectAndFixDrift(); err != nil {
		h.detector.app.Logger.Printf("Handler error: %v", err)
	}
}

func (d *DriftDetector) jsonPretty(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
