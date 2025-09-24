package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	sdk "github.com/monadic/devops-sdk"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CostImpactMonitor monitors ConfigHub for cost impacts of deployments
type CostImpactMonitor struct {
	app              *sdk.DevOpsApp
	monitoredSpaces  map[uuid.UUID]*SpaceMonitor
	triggerProcessor *TriggerProcessor
	dashboard        *MonitorDashboard
	mu               sync.RWMutex
}

// SpaceMonitor tracks costs for a specific ConfigHub space
type SpaceMonitor struct {
	SpaceID          uuid.UUID              `json:"space_id"`
	SpaceName        string                 `json:"space_name"`
	LastAnalysis     time.Time              `json:"last_analysis"`
	CurrentCost      float64                `json:"current_cost"`
	ProjectedCost    float64                `json:"projected_cost"`
	PendingChanges   []PendingChange        `json:"pending_changes"`
	DeploymentHistory []DeploymentCostRecord `json:"deployment_history"`
	CostTrend        CostTrend              `json:"cost_trend"`
}

// PendingChange represents a unit change awaiting deployment
type PendingChange struct {
	UnitID           string    `json:"unit_id"`
	UnitName         string    `json:"unit_name"`
	ChangeType       string    `json:"change_type"` // "create", "update", "delete"
	CurrentCost      float64   `json:"current_cost"`
	ProjectedCost    float64   `json:"projected_cost"`
	CostDelta        float64   `json:"cost_delta"`
	RiskLevel        string    `json:"risk_level"` // "low", "medium", "high"
	AnalysisTime     time.Time `json:"analysis_time"`
	ClaudeAssessment string    `json:"claude_assessment"`
}

// DeploymentCostRecord tracks actual vs predicted costs
type DeploymentCostRecord struct {
	UnitID        string    `json:"unit_id"`
	UnitName      string    `json:"unit_name"`
	DeployTime    time.Time `json:"deploy_time"`
	PredictedCost float64   `json:"predicted_cost"`
	ActualCost    float64   `json:"actual_cost"`
	Variance      float64   `json:"variance"`
	Accurate      bool      `json:"accurate"` // Within 10% of prediction
}

// CostTrend tracks cost direction over time
type CostTrend struct {
	Direction       string  `json:"direction"` // "increasing", "decreasing", "stable"
	WeeklyChange    float64 `json:"weekly_change_percent"`
	MonthlyChange   float64 `json:"monthly_change_percent"`
	ProjectedMonthly float64 `json:"projected_monthly_cost"`
}

// TriggerProcessor handles ConfigHub triggers
type TriggerProcessor struct {
	monitor         *CostImpactMonitor
	preApplyHooks   []PreApplyHook
	postApplyHooks  []PostApplyHook
	changeDetector  *ChangeDetector
	lastProcessed   map[string]time.Time
	mu              sync.Mutex
}

// PreApplyHook is called before unit deployment
type PreApplyHook func(unit *sdk.Unit, impact *CostImpact) error

// PostApplyHook is called after unit deployment
type PostApplyHook func(unit *sdk.Unit, actual *ActualUsage) error

// CostImpact represents predicted cost impact
type CostImpact struct {
	UnitID         string                 `json:"unit_id"`
	UnitName       string                 `json:"unit_name"`
	MonthlyCost    float64                `json:"monthly_cost"`
	CostDelta      float64                `json:"cost_delta"`
	ResourceChanges map[string]interface{} `json:"resource_changes"`
	RiskAssessment  RiskAssessment         `json:"risk_assessment"`
}

// RiskAssessment evaluates deployment risk
type RiskAssessment struct {
	Level       string   `json:"level"` // "low", "medium", "high", "critical"
	Factors     []string `json:"factors"`
	Recommendation string   `json:"recommendation"`
	AutoApprove bool     `json:"auto_approve"`
}

// ActualUsage represents real resource consumption
type ActualUsage struct {
	UnitID       string  `json:"unit_id"`
	UnitName     string  `json:"unit_name"`
	CPUCores     float64 `json:"cpu_cores"`
	MemoryGB     float64 `json:"memory_gb"`
	StorageGB    float64 `json:"storage_gb"`
	MonthlyCost  float64 `json:"monthly_cost"`
	MeasuredAt   time.Time `json:"measured_at"`
}

// ChangeDetector monitors ConfigHub for changes
type ChangeDetector struct {
	monitor       *CostImpactMonitor
	pollInterval  time.Duration
	unitCache     map[string]*sdk.Unit
	revisionCache map[string]int
}

func main() {
	monitor, err := NewCostImpactMonitor()
	if err != nil {
		log.Fatalf("Failed to initialize cost impact monitor: %v", err)
	}

	log.Println("🚀 Cost Impact Monitor started - Monitoring all ConfigHub spaces")

	// Start dashboard
	go monitor.dashboard.Start()

	// Start trigger processor
	go monitor.triggerProcessor.Start()

	// Run main monitoring loop with informers
	err = monitor.app.RunWithInformers(func() error {
		return monitor.monitorAllSpaces()
	})
	if err != nil {
		log.Fatalf("Monitoring failed: %v", err)
	}
}

// NewCostImpactMonitor creates a new cost impact monitor
func NewCostImpactMonitor() (*CostImpactMonitor, error) {
	app, err := sdk.NewDevOpsApp(sdk.DevOpsAppConfig{
		Name:        "cost-impact-monitor",
		Version:     "1.0.0",
		Description: "Monitor ConfigHub deployments for cost impact",
		RunInterval: 1 * time.Minute, // Check for changes every minute
		HealthPort:  8082,
	})
	if err != nil {
		return nil, fmt.Errorf("create DevOps app: %w", err)
	}

	monitor := &CostImpactMonitor{
		app:             app,
		monitoredSpaces: make(map[uuid.UUID]*SpaceMonitor),
	}

	// Initialize trigger processor
	monitor.triggerProcessor = &TriggerProcessor{
		monitor:       monitor,
		lastProcessed: make(map[string]time.Time),
		changeDetector: &ChangeDetector{
			monitor:       monitor,
			pollInterval:  30 * time.Second,
			unitCache:     make(map[string]*sdk.Unit),
			revisionCache: make(map[string]int),
		},
	}

	// Register default hooks
	monitor.registerDefaultHooks()

	// Initialize dashboard
	monitor.dashboard = NewMonitorDashboard(monitor)

	// Discover and register all ConfigHub spaces
	if err := monitor.discoverSpaces(); err != nil {
		return nil, fmt.Errorf("discover spaces: %w", err)
	}

	return monitor, nil
}

// discoverSpaces finds all ConfigHub spaces to monitor
func (m *CostImpactMonitor) discoverSpaces() error {
	if m.app.Cub == nil {
		m.app.Logger.Println("⚠️  ConfigHub not configured - running in demo mode")
		return nil
	}

	// List all spaces
	spaces, err := m.app.Cub.ListSpaces()
	if err != nil {
		return fmt.Errorf("list spaces: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, space := range spaces {
		m.monitoredSpaces[space.SpaceID] = &SpaceMonitor{
			SpaceID:          space.SpaceID,
			SpaceName:        space.Slug,
			LastAnalysis:     time.Now(),
			DeploymentHistory: make([]DeploymentCostRecord, 0),
		}
		m.app.Logger.Printf("📦 Monitoring space: %s (%s)", space.Slug, space.SpaceID)
	}

	m.app.Logger.Printf("🔍 Discovered %d ConfigHub spaces to monitor", len(spaces))
	return nil
}

// monitorAllSpaces analyzes costs across all ConfigHub spaces
func (m *CostImpactMonitor) monitorAllSpaces() error {
	m.mu.RLock()
	spaces := make([]*SpaceMonitor, 0, len(m.monitoredSpaces))
	for _, space := range m.monitoredSpaces {
		spaces = append(spaces, space)
	}
	m.mu.RUnlock()

	// Analyze each space in parallel
	var wg sync.WaitGroup
	for _, space := range spaces {
		wg.Add(1)
		go func(s *SpaceMonitor) {
			defer wg.Done()
			if err := m.analyzeSpace(s); err != nil {
				m.app.Logger.Printf("⚠️  Failed to analyze space %s: %v", s.SpaceName, err)
			}
		}(space)
	}
	wg.Wait()

	// Update dashboard with latest data
	m.dashboard.UpdateMonitoringData(m.getMonitoringSnapshot())

	return nil
}

// analyzeSpace analyzes cost for a specific space
func (m *CostImpactMonitor) analyzeSpace(space *SpaceMonitor) error {
	// Get all units in the space
	units, err := m.app.Cub.ListUnits(space.SpaceID)
	if err != nil {
		return fmt.Errorf("list units: %w", err)
	}

	totalCost := 0.0
	pendingChanges := []PendingChange{}

	// Analyze each unit
	for _, unit := range units {
		// Calculate current cost
		cost := m.calculateUnitCost(unit)
		totalCost += cost

		// Check for pending changes (units not yet applied)
		if unit.LiveState == nil || unit.LiveState.Status != "Applied" {
			change := m.analyzePendingChange(unit, cost)
			pendingChanges = append(pendingChanges, change)
		}
	}

	// Update space monitor
	space.CurrentCost = totalCost
	space.ProjectedCost = totalCost // Will be updated by pending changes
	space.PendingChanges = pendingChanges
	space.LastAnalysis = time.Now()

	// Calculate projected cost including pending changes
	for _, change := range pendingChanges {
		space.ProjectedCost += change.CostDelta
	}

	// Update cost trend
	space.CostTrend = m.calculateCostTrend(space)

	m.app.Logger.Printf("💰 Space %s: Current $%.2f/month, Projected $%.2f/month (%d pending changes)",
		space.SpaceName, space.CurrentCost, space.ProjectedCost, len(pendingChanges))

	return nil
}

// calculateUnitCost estimates monthly cost for a unit
func (m *CostImpactMonitor) calculateUnitCost(unit *sdk.Unit) float64 {
	// Parse unit data to extract resource requirements
	// This is simplified - real implementation would parse YAML/JSON manifests

	// Default cost calculation based on labels
	baseCost := 10.0 // Base cost per unit

	// Check for resource hints in labels
	if cpu, ok := unit.Labels["cpu"]; ok {
		// Parse CPU and calculate cost
		_ = cpu
		baseCost += 20.0 // Simplified
	}

	if memory, ok := unit.Labels["memory"]; ok {
		// Parse memory and calculate cost
		_ = memory
		baseCost += 15.0 // Simplified
	}

	return baseCost
}

// analyzePendingChange analyzes a unit that hasn't been applied yet
func (m *CostImpactMonitor) analyzePendingChange(unit *sdk.Unit, projectedCost float64) PendingChange {
	change := PendingChange{
		UnitID:        unit.UnitID.String(),
		UnitName:      unit.Slug,
		ChangeType:    "update",
		ProjectedCost: projectedCost,
		AnalysisTime:  time.Now(),
	}

	// Determine change type
	if unit.LiveState == nil {
		change.ChangeType = "create"
		change.CurrentCost = 0
		change.CostDelta = projectedCost
	} else {
		// For updates, calculate delta
		change.CurrentCost = m.calculateUnitCost(unit) // Should use previous version
		change.CostDelta = projectedCost - change.CurrentCost
	}

	// Risk assessment
	change.RiskLevel = m.assessRisk(change.CostDelta)

	// Get Claude assessment if available
	if m.app.Claude != nil {
		change.ClaudeAssessment = m.getClaudeAssessment(unit, change)
	}

	return change
}

// assessRisk determines risk level based on cost delta
func (m *CostImpactMonitor) assessRisk(costDelta float64) string {
	absDelta := costDelta
	if absDelta < 0 {
		absDelta = -absDelta
	}

	switch {
	case absDelta < 50:
		return "low"
	case absDelta < 200:
		return "medium"
	case absDelta < 500:
		return "high"
	default:
		return "critical"
	}
}

// getClaudeAssessment gets AI assessment of the change
func (m *CostImpactMonitor) getClaudeAssessment(unit *sdk.Unit, change PendingChange) string {
	prompt := fmt.Sprintf(`Assess this ConfigHub deployment cost change:
Unit: %s
Change Type: %s
Cost Delta: $%.2f/month
Risk Level: %s

Provide a brief risk assessment and recommendation.`,
		unit.Slug, change.ChangeType, change.CostDelta, change.RiskLevel)

	response, err := m.app.Claude.Complete(prompt)
	if err != nil {
		m.app.Logger.Printf("⚠️  Claude assessment failed: %v", err)
		return "AI assessment unavailable"
	}

	return response
}

// calculateCostTrend analyzes cost trends for a space
func (m *CostImpactMonitor) calculateCostTrend(space *SpaceMonitor) CostTrend {
	trend := CostTrend{
		Direction:        "stable",
		ProjectedMonthly: space.ProjectedCost,
	}

	// Calculate trend from deployment history
	if len(space.DeploymentHistory) >= 2 {
		recent := space.DeploymentHistory[len(space.DeploymentHistory)-1]
		previous := space.DeploymentHistory[len(space.DeploymentHistory)-2]

		change := (recent.ActualCost - previous.ActualCost) / previous.ActualCost * 100
		trend.WeeklyChange = change

		if change > 5 {
			trend.Direction = "increasing"
		} else if change < -5 {
			trend.Direction = "decreasing"
		}
	}

	return trend
}

// registerDefaultHooks sets up default trigger hooks
func (m *CostImpactMonitor) registerDefaultHooks() {
	// Pre-apply hook: Warn about high costs
	m.triggerProcessor.preApplyHooks = append(m.triggerProcessor.preApplyHooks,
		func(unit *sdk.Unit, impact *CostImpact) error {
			if impact.CostDelta > 100 {
				m.app.Logger.Printf("⚠️  HIGH COST WARNING: %s will increase costs by $%.2f/month",
					unit.Slug, impact.CostDelta)

				// Store warning in ConfigHub
				m.createCostWarning(unit, impact)
			}
			return nil
		})

	// Post-apply hook: Track accuracy
	m.triggerProcessor.postApplyHooks = append(m.triggerProcessor.postApplyHooks,
		func(unit *sdk.Unit, actual *ActualUsage) error {
			m.app.Logger.Printf("✅ Deployed %s - Actual cost: $%.2f/month",
				unit.Slug, actual.MonthlyCost)

			// Update deployment history
			m.updateDeploymentHistory(unit, actual)
			return nil
		})
}

// createCostWarning creates a warning unit in ConfigHub
func (m *CostImpactMonitor) createCostWarning(unit *sdk.Unit, impact *CostImpact) {
	if m.app.Cub == nil {
		return
	}

	warningData, _ := json.MarshalIndent(impact, "", "  ")

	_, err := m.app.Cub.CreateUnit(unit.SpaceID, sdk.CreateUnitRequest{
		Slug:        fmt.Sprintf("cost-warning-%s-%d", unit.Slug, time.Now().Unix()),
		DisplayName: fmt.Sprintf("Cost Warning: %s", unit.Slug),
		Data:        string(warningData),
		Labels: map[string]string{
			"type":        "cost-warning",
			"unit":        unit.Slug,
			"cost_delta":  fmt.Sprintf("%.2f", impact.CostDelta),
			"risk":        impact.RiskAssessment.Level,
		},
	})

	if err != nil {
		m.app.Logger.Printf("⚠️  Failed to create cost warning: %v", err)
	}
}

// updateDeploymentHistory records actual deployment costs
func (m *CostImpactMonitor) updateDeploymentHistory(unit *sdk.Unit, actual *ActualUsage) {
	m.mu.Lock()
	defer m.mu.Unlock()

	space, exists := m.monitoredSpaces[unit.SpaceID]
	if !exists {
		return
	}

	record := DeploymentCostRecord{
		UnitID:        unit.UnitID.String(),
		UnitName:      unit.Slug,
		DeployTime:    time.Now(),
		ActualCost:    actual.MonthlyCost,
		PredictedCost: m.calculateUnitCost(unit),
	}

	// Calculate variance
	if record.PredictedCost > 0 {
		record.Variance = ((record.ActualCost - record.PredictedCost) / record.PredictedCost) * 100
		record.Accurate = record.Variance >= -10 && record.Variance <= 10
	}

	space.DeploymentHistory = append(space.DeploymentHistory, record)

	// Keep only last 100 records
	if len(space.DeploymentHistory) > 100 {
		space.DeploymentHistory = space.DeploymentHistory[len(space.DeploymentHistory)-100:]
	}
}

// getMonitoringSnapshot returns current monitoring state
func (m *CostImpactMonitor) getMonitoringSnapshot() *MonitoringSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := &MonitoringSnapshot{
		Timestamp:       time.Now(),
		TotalSpaces:     len(m.monitoredSpaces),
		TotalCost:       0,
		ProjectedCost:   0,
		PendingChanges:  0,
		HighRiskChanges: 0,
	}

	for _, space := range m.monitoredSpaces {
		snapshot.TotalCost += space.CurrentCost
		snapshot.ProjectedCost += space.ProjectedCost
		snapshot.PendingChanges += len(space.PendingChanges)

		for _, change := range space.PendingChanges {
			if change.RiskLevel == "high" || change.RiskLevel == "critical" {
				snapshot.HighRiskChanges++
			}
		}

		snapshot.Spaces = append(snapshot.Spaces, space)
	}

	return snapshot
}

// MonitoringSnapshot represents current state of all monitoring
type MonitoringSnapshot struct {
	Timestamp       time.Time       `json:"timestamp"`
	TotalSpaces     int             `json:"total_spaces"`
	TotalCost       float64         `json:"total_cost"`
	ProjectedCost   float64         `json:"projected_cost"`
	PendingChanges  int             `json:"pending_changes"`
	HighRiskChanges int             `json:"high_risk_changes"`
	Spaces          []*SpaceMonitor `json:"spaces"`
}

// TriggerProcessor methods

// Start begins monitoring for ConfigHub changes
func (t *TriggerProcessor) Start() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		t.checkForChanges()
	}
}

// checkForChanges polls ConfigHub for unit changes
func (t *TriggerProcessor) checkForChanges() {
	if t.monitor.app.Cub == nil {
		return
	}

	t.monitor.mu.RLock()
	spaces := make([]uuid.UUID, 0, len(t.monitor.monitoredSpaces))
	for spaceID := range t.monitor.monitoredSpaces {
		spaces = append(spaces, spaceID)
	}
	t.monitor.mu.RUnlock()

	for _, spaceID := range spaces {
		units, err := t.monitor.app.Cub.ListUnits(spaceID)
		if err != nil {
			continue
		}

		for _, unit := range units {
			t.processUnitChange(unit)
		}
	}
}

// processUnitChange handles a unit that may have changed
func (t *TriggerProcessor) processUnitChange(unit *sdk.Unit) {
	unitKey := unit.UnitID.String()

	t.mu.Lock()
	lastProcessed, exists := t.lastProcessed[unitKey]
	t.mu.Unlock()

	// Check if unit has been modified since last processing
	if exists && unit.UpdatedAt.Before(lastProcessed) {
		return
	}

	// Check if unit is about to be applied
	if unit.LiveState == nil || unit.LiveState.Status == "Pending" {
		// Pre-apply trigger
		impact := t.analyzeImpact(unit)
		for _, hook := range t.preApplyHooks {
			if err := hook(unit, impact); err != nil {
				t.monitor.app.Logger.Printf("⚠️  Pre-apply hook error: %v", err)
			}
		}
	}

	// Check if unit was recently applied
	if unit.LiveState != nil && unit.LiveState.Status == "Applied" {
		// Post-apply trigger
		actual := t.measureActualUsage(unit)
		for _, hook := range t.postApplyHooks {
			if err := hook(unit, actual); err != nil {
				t.monitor.app.Logger.Printf("⚠️  Post-apply hook error: %v", err)
			}
		}
	}

	t.mu.Lock()
	t.lastProcessed[unitKey] = time.Now()
	t.mu.Unlock()
}

// analyzeImpact predicts cost impact of a unit deployment
func (t *TriggerProcessor) analyzeImpact(unit *sdk.Unit) *CostImpact {
	impact := &CostImpact{
		UnitID:      unit.UnitID.String(),
		UnitName:    unit.Slug,
		MonthlyCost: t.monitor.calculateUnitCost(unit),
	}

	// Calculate delta if unit exists
	if unit.LiveState != nil {
		currentCost := t.monitor.calculateUnitCost(unit) // Should use previous version
		impact.CostDelta = impact.MonthlyCost - currentCost
	} else {
		impact.CostDelta = impact.MonthlyCost
	}

	// Risk assessment
	impact.RiskAssessment = t.assessRisk(unit, impact.CostDelta)

	return impact
}

// measureActualUsage gets real resource usage for a deployed unit
func (t *TriggerProcessor) measureActualUsage(unit *sdk.Unit) *ActualUsage {
	actual := &ActualUsage{
		UnitID:     unit.UnitID.String(),
		UnitName:   unit.Slug,
		MeasuredAt: time.Now(),
	}

	// In real implementation, would query metrics-server or Prometheus
	// For now, use estimates
	actual.CPUCores = 0.5
	actual.MemoryGB = 1.0
	actual.StorageGB = 10.0
	actual.MonthlyCost = (actual.CPUCores * 24 * 30 * 0.024) +
	                   (actual.MemoryGB * 24 * 30 * 0.006)

	return actual
}

// assessRisk evaluates deployment risk
func (t *TriggerProcessor) assessRisk(unit *sdk.Unit, costDelta float64) RiskAssessment {
	assessment := RiskAssessment{
		Level:       "low",
		Factors:     []string{},
		AutoApprove: true,
	}

	// Check cost delta
	if costDelta > 500 {
		assessment.Level = "critical"
		assessment.Factors = append(assessment.Factors, "Very high cost increase")
		assessment.AutoApprove = false
	} else if costDelta > 200 {
		assessment.Level = "high"
		assessment.Factors = append(assessment.Factors, "Significant cost increase")
		assessment.AutoApprove = false
	} else if costDelta > 50 {
		assessment.Level = "medium"
		assessment.Factors = append(assessment.Factors, "Moderate cost increase")
	}

	// Check for production environment
	if unit.Labels["env"] == "production" {
		if assessment.Level == "low" {
			assessment.Level = "medium"
		}
		assessment.Factors = append(assessment.Factors, "Production environment")
		assessment.AutoApprove = false
	}

	// Generate recommendation
	switch assessment.Level {
	case "critical":
		assessment.Recommendation = "DO NOT DEPLOY without executive approval"
	case "high":
		assessment.Recommendation = "Review with team lead before deployment"
	case "medium":
		assessment.Recommendation = "Review cost optimization opportunities"
	default:
		assessment.Recommendation = "Safe to deploy"
	}

	return assessment
}