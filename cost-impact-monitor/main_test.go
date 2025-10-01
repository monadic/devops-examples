// Cost Impact Monitor Test Suite
// Copy to: /Users/alexis/Public/github-repos/devops-examples/cost-impact-monitor/main_test.go

package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	sdk "github.com/monadic/devops-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// UNIT TESTS
// ============================================================================

func TestCostImpactAnalysis(t *testing.T) {
	t.Run("CalculateCostDelta", func(t *testing.T) {
		before := &sdk.SpaceCostAnalysis{
			TotalMonthlyCost: 1000.0,
		}

		after := &sdk.SpaceCostAnalysis{
			TotalMonthlyCost: 1200.0,
		}

		delta := calculateCostDelta(before, after)

		assert.Equal(t, 200.0, delta.AbsoluteIncrease, "Absolute increase should be $200")
		assert.Equal(t, 20.0, delta.PercentageIncrease, "Percentage increase should be 20%")
	})

	t.Run("CostDecreaseScenario", func(t *testing.T) {
		before := &sdk.SpaceCostAnalysis{
			TotalMonthlyCost: 1000.0,
		}

		after := &sdk.SpaceCostAnalysis{
			TotalMonthlyCost: 800.0,
		}

		delta := calculateCostDelta(before, after)

		assert.Equal(t, -200.0, delta.AbsoluteIncrease, "Should show decrease")
		assert.Equal(t, -20.0, delta.PercentageIncrease, "Should show 20% decrease")
	})

	t.Run("NoCostChange", func(t *testing.T) {
		before := &sdk.SpaceCostAnalysis{
			TotalMonthlyCost: 1000.0,
		}

		after := &sdk.SpaceCostAnalysis{
			TotalMonthlyCost: 1000.0,
		}

		delta := calculateCostDelta(before, after)

		assert.Equal(t, 0.0, delta.AbsoluteIncrease, "No change")
		assert.Equal(t, 0.0, delta.PercentageIncrease, "0% change")
	})
}

func TestTriggerDetection(t *testing.T) {
	t.Run("DetectHelmInstall", func(t *testing.T) {
		event := &K8sEvent{
			Type:      "ADDED",
			Kind:      "Release",
			Name:      "prometheus",
			Namespace: "monitoring",
		}

		isTrigger := isTriggerEvent(event)
		assert.True(t, isTrigger, "Helm release should be a trigger")
	})

	t.Run("DetectDeploymentScale", func(t *testing.T) {
		event := &K8sEvent{
			Type:      "MODIFIED",
			Kind:      "Deployment",
			Name:      "frontend",
			Namespace: "production",
			Changes: map[string]interface{}{
				"spec.replicas": map[string]int{
					"old": 3,
					"new": 10,
				},
			},
		}

		isTrigger := isTriggerEvent(event)
		assert.True(t, isTrigger, "Significant scale-up should be a trigger")
	})

	t.Run("IgnoreMinorChanges", func(t *testing.T) {
		event := &K8sEvent{
			Type:      "MODIFIED",
			Kind:      "Deployment",
			Name:      "frontend",
			Namespace: "production",
			Changes: map[string]interface{}{
				"metadata.annotations": map[string]string{
					"old": "v1",
					"new": "v2",
				},
			},
		}

		isTrigger := isTriggerEvent(event)
		assert.False(t, isTrigger, "Annotation changes should not trigger")
	})
}

func TestCrossSpaceMonitoring(t *testing.T) {
	t.Run("MonitorMultipleSpaces", func(t *testing.T) {
		spaces := []uuid.UUID{
			uuid.New(),
			uuid.New(),
			uuid.New(),
		}

		monitor := NewCostImpactMonitor(spaces)

		assert.Equal(t, 3, len(monitor.spaces), "Should monitor 3 spaces")
		assert.NotNil(t, monitor.costHistory, "Cost history should be initialized")
	})

	t.Run("TrackCostHistory", func(t *testing.T) {
		monitor := NewCostImpactMonitor([]uuid.UUID{uuid.New()})

		// Record cost snapshots
		monitor.RecordCostSnapshot("2024-01-01T10:00:00Z", 1000.0)
		monitor.RecordCostSnapshot("2024-01-01T11:00:00Z", 1200.0)
		monitor.RecordCostSnapshot("2024-01-01T12:00:00Z", 1500.0)

		history := monitor.GetCostHistory()

		assert.Equal(t, 3, len(history), "Should have 3 snapshots")
		assert.Equal(t, 1000.0, history[0].Cost, "First snapshot")
		assert.Equal(t, 1500.0, history[2].Cost, "Latest snapshot")
	})

	t.Run("CalculateCostTrend", func(t *testing.T) {
		monitor := NewCostImpactMonitor([]uuid.UUID{uuid.New()})

		// Simulate increasing trend
		times := []string{
			"2024-01-01T10:00:00Z",
			"2024-01-01T11:00:00Z",
			"2024-01-01T12:00:00Z",
			"2024-01-01T13:00:00Z",
		}

		costs := []float64{1000.0, 1100.0, 1250.0, 1400.0}

		for i := range times {
			monitor.RecordCostSnapshot(times[i], costs[i])
		}

		trend := monitor.CalculateTrend()

		assert.Equal(t, "INCREASING", trend.Direction, "Trend should be increasing")
		assert.Greater(t, trend.AverageChangePercent, 0.0, "Average change should be positive")
	})
}

func TestImpactReport(t *testing.T) {
	t.Run("GenerateImpactReport", func(t *testing.T) {
		trigger := &TriggerEvent{
			Type:      "helm_install",
			Chart:     "prometheus",
			Namespace: "monitoring",
			Timestamp: time.Now(),
		}

		costBefore := &sdk.SpaceCostAnalysis{
			TotalMonthlyCost: 1000.0,
		}

		costAfter := &sdk.SpaceCostAnalysis{
			TotalMonthlyCost: 1500.0,
		}

		report := GenerateImpactReport(trigger, costBefore, costAfter)

		require.NotNil(t, report)
		assert.Equal(t, "helm_install", report.TriggerType)
		assert.Equal(t, 500.0, report.CostIncrease)
		assert.Equal(t, 50.0, report.PercentageIncrease)
		assert.Contains(t, report.Recommendations, "prometheus", "Should mention chart name")
	})

	t.Run("ReportWithResourceBreakdown", func(t *testing.T) {
		trigger := &TriggerEvent{
			Type: "deployment_scale",
		}

		costBefore := &sdk.SpaceCostAnalysis{
			TotalMonthlyCost: 1000.0,
			Units: []sdk.UnitCostEstimate{
				{
					UnitName:    "frontend",
					MonthlyCost: 300.0,
					Breakdown: sdk.CostBreakdown{
						CPUCost:    200.0,
						MemoryCost: 100.0,
					},
				},
			},
		}

		costAfter := &sdk.SpaceCostAnalysis{
			TotalMonthlyCost: 1500.0,
			Units: []sdk.UnitCostEstimate{
				{
					UnitName:    "frontend",
					MonthlyCost: 900.0,
					Breakdown: sdk.CostBreakdown{
						CPUCost:    600.0,
						MemoryCost: 300.0,
					},
				},
			},
		}

		report := GenerateImpactReport(trigger, costBefore, costAfter)

		assert.NotNil(t, report.ResourceBreakdown)
		assert.Equal(t, 400.0, report.ResourceBreakdown.CPUIncrease)
		assert.Equal(t, 200.0, report.ResourceBreakdown.MemoryIncrease)
	})
}

func TestDashboardData(t *testing.T) {
	t.Run("SerializeDashboardData", func(t *testing.T) {
		monitor := NewCostImpactMonitor([]uuid.UUID{uuid.New()})

		// Add some data
		monitor.RecordCostSnapshot("2024-01-01T10:00:00Z", 1000.0)
		monitor.RecordCostSnapshot("2024-01-01T11:00:00Z", 1200.0)

		dashboardData := monitor.GetDashboardData()

		require.NotNil(t, dashboardData)
		assert.Equal(t, 2, len(dashboardData.CostHistory))
		assert.Equal(t, 1200.0, dashboardData.CurrentCost)
		assert.Equal(t, 200.0, dashboardData.LastHourIncrease)
	})

	t.Run("DashboardAlertsGeneration", func(t *testing.T) {
		monitor := NewCostImpactMonitor([]uuid.UUID{uuid.New()})

		// Simulate large cost increase
		monitor.RecordCostSnapshot("2024-01-01T10:00:00Z", 1000.0)
		monitor.RecordCostSnapshot("2024-01-01T11:00:00Z", 2500.0) // 150% increase

		dashboardData := monitor.GetDashboardData()

		assert.Greater(t, len(dashboardData.Alerts), 0, "Should generate alerts")
		assert.Contains(t, dashboardData.Alerts[0].Message, "cost increase", "Alert should mention cost increase")
		assert.Equal(t, "HIGH", dashboardData.Alerts[0].Severity)
	})
}

// ============================================================================
// INTEGRATION TESTS
// ============================================================================

func TestCostImpactMonitorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("FullWorkflow", func(t *testing.T) {
		// Initialize monitor
		spaceIDs := []uuid.UUID{uuid.New()}
		monitor := NewCostImpactMonitor(spaceIDs)

		// Step 1: Record baseline
		baseline, err := monitor.RecordBaseline()
		require.NoError(t, err)
		assert.Greater(t, baseline.TotalMonthlyCost, 0.0)

		// Step 2: Simulate trigger event
		trigger := &TriggerEvent{
			Type:      "helm_install",
			Chart:     "prometheus",
			Namespace: "monitoring",
			Timestamp: time.Now(),
		}

		err = monitor.RegisterTrigger(trigger)
		require.NoError(t, err)

		// Step 3: Wait for deployment
		time.Sleep(30 * time.Second)

		// Step 4: Measure impact
		impact, err := monitor.MeasureImpact(trigger.ID)
		require.NoError(t, err)
		assert.NotNil(t, impact)

		// Step 5: Verify dashboard updated
		dashboardData := monitor.GetDashboardData()
		assert.NotNil(t, dashboardData)
		assert.Greater(t, len(dashboardData.CostHistory), 1)
	})
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func calculateCostDelta(before, after *sdk.SpaceCostAnalysis) *CostDelta {
	return &CostDelta{
		AbsoluteIncrease:   after.TotalMonthlyCost - before.TotalMonthlyCost,
		PercentageIncrease: ((after.TotalMonthlyCost - before.TotalMonthlyCost) / before.TotalMonthlyCost) * 100,
	}
}

func isTriggerEvent(event *K8sEvent) bool {
	// Simplified trigger detection
	if event.Kind == "Release" {
		return true
	}

	if event.Kind == "Deployment" && event.Type == "MODIFIED" {
		if changes, ok := event.Changes["spec.replicas"].(map[string]int); ok {
			if changes["new"] > changes["old"]*2 {
				return true // Significant scale-up
			}
		}
	}

	return false
}

// ============================================================================
// PLACEHOLDER TYPES
// ============================================================================

type CostDelta struct {
	AbsoluteIncrease   float64
	PercentageIncrease float64
}

type K8sEvent struct {
	Type      string
	Kind      string
	Name      string
	Namespace string
	Changes   map[string]interface{}
}

type TriggerEvent struct {
	ID        string
	Type      string
	Chart     string
	Namespace string
	Timestamp time.Time
}

type CostImpactMonitor struct {
	spaces      []uuid.UUID
	costHistory []CostSnapshot
}

type CostSnapshot struct {
	Timestamp string
	Cost      float64
}

type ImpactReport struct {
	TriggerType        string
	CostIncrease       float64
	PercentageIncrease float64
	Recommendations    []string
	ResourceBreakdown  *ResourceBreakdown
}

type ResourceBreakdown struct {
	CPUIncrease    float64
	MemoryIncrease float64
}

type DashboardData struct {
	CostHistory       []CostSnapshot
	CurrentCost       float64
	LastHourIncrease  float64
	Alerts            []Alert
}

type Alert struct {
	Message  string
	Severity string
}

type Trend struct {
	Direction            string
	AverageChangePercent float64
}

func NewCostImpactMonitor(spaces []uuid.UUID) *CostImpactMonitor {
	return &CostImpactMonitor{
		spaces:      spaces,
		costHistory: []CostSnapshot{},
	}
}

func (m *CostImpactMonitor) RecordCostSnapshot(timestamp string, cost float64) {
	m.costHistory = append(m.costHistory, CostSnapshot{Timestamp: timestamp, Cost: cost})
}

func (m *CostImpactMonitor) GetCostHistory() []CostSnapshot {
	return m.costHistory
}

func (m *CostImpactMonitor) CalculateTrend() *Trend {
	if len(m.costHistory) < 2 {
		return &Trend{Direction: "STABLE", AverageChangePercent: 0}
	}

	totalChange := 0.0
	for i := 1; i < len(m.costHistory); i++ {
		change := ((m.costHistory[i].Cost - m.costHistory[i-1].Cost) / m.costHistory[i-1].Cost) * 100
		totalChange += change
	}

	avgChange := totalChange / float64(len(m.costHistory)-1)
	direction := "STABLE"
	if avgChange > 5 {
		direction = "INCREASING"
	} else if avgChange < -5 {
		direction = "DECREASING"
	}

	return &Trend{
		Direction:            direction,
		AverageChangePercent: avgChange,
	}
}

func (m *CostImpactMonitor) GetDashboardData() *DashboardData {
	currentCost := 0.0
	lastHourIncrease := 0.0

	if len(m.costHistory) > 0 {
		currentCost = m.costHistory[len(m.costHistory)-1].Cost
	}

	if len(m.costHistory) > 1 {
		lastHourIncrease = currentCost - m.costHistory[len(m.costHistory)-2].Cost
	}

	alerts := []Alert{}
	if lastHourIncrease > currentCost*0.5 { // 50% increase
		alerts = append(alerts, Alert{
			Message:  "Significant cost increase detected in last hour",
			Severity: "HIGH",
		})
	}

	return &DashboardData{
		CostHistory:      m.costHistory,
		CurrentCost:      currentCost,
		LastHourIncrease: lastHourIncrease,
		Alerts:           alerts,
	}
}

func (m *CostImpactMonitor) RecordBaseline() (*sdk.SpaceCostAnalysis, error) {
	return &sdk.SpaceCostAnalysis{TotalMonthlyCost: 1000.0}, nil
}

func (m *CostImpactMonitor) RegisterTrigger(trigger *TriggerEvent) error {
	return nil
}

func (m *CostImpactMonitor) MeasureImpact(triggerID string) (*ImpactReport, error) {
	return &ImpactReport{}, nil
}

func GenerateImpactReport(trigger *TriggerEvent, before, after *sdk.SpaceCostAnalysis) *ImpactReport {
	costIncrease := after.TotalMonthlyCost - before.TotalMonthlyCost
	percentageIncrease := (costIncrease / before.TotalMonthlyCost) * 100

	recommendations := []string{
		"Review resource requests for " + trigger.Chart,
		"Consider enabling autoscaling",
		"Monitor cost trends over next 24 hours",
	}

	var resourceBreakdown *ResourceBreakdown
	if len(before.Units) > 0 && len(after.Units) > 0 {
		resourceBreakdown = &ResourceBreakdown{
			CPUIncrease:    after.Units[0].Breakdown.CPUCost - before.Units[0].Breakdown.CPUCost,
			MemoryIncrease: after.Units[0].Breakdown.MemoryCost - before.Units[0].Breakdown.MemoryCost,
		}
	}

	return &ImpactReport{
		TriggerType:        trigger.Type,
		CostIncrease:       costIncrease,
		PercentageIncrease: percentageIncrease,
		Recommendations:    recommendations,
		ResourceBreakdown:  resourceBreakdown,
	}
}
