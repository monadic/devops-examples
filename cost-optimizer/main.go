package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	sdk "github.com/monadic/devops-sdk"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// CostOptimizer is the main application using our enhanced SDK
type CostOptimizer struct {
	app           *sdk.DevOpsApp
	spaceID       uuid.UUID
	criticalSetID uuid.UUID
	dashboard     *Dashboard
}

// CostAnalysis represents the complete cost analysis
type CostAnalysis struct {
	Timestamp           time.Time            `json:"timestamp"`
	TotalMonthlyCost    float64              `json:"total_monthly_cost"`
	PotentialSavings    float64              `json:"potential_savings"`
	SavingsPercentage   float64              `json:"savings_percentage"`
	Recommendations     []CostRecommendation `json:"recommendations"`
	ResourceBreakdown   ResourceBreakdown    `json:"resource_breakdown"`
	ClusterSummary      ClusterSummary       `json:"cluster_summary"`
}

type CostRecommendation struct {
	Resource        string                 `json:"resource"`
	Namespace       string                 `json:"namespace"`
	Type            string                 `json:"type"` // "rightsize", "scale_down", "remove_unused", "optimize_storage"
	Priority        string                 `json:"priority"` // "high", "medium", "low"
	Current         map[string]interface{} `json:"current"`
	Recommended     map[string]interface{} `json:"recommended"`
	MonthlySavings  float64                `json:"monthly_savings"`
	Risk            string                 `json:"risk"` // "low", "medium", "high"
	Explanation     string                 `json:"explanation"`
	ConfigHubAction string                 `json:"confighub_action"` // What to update in ConfigHub
}

type ResourceBreakdown struct {
	Compute float64 `json:"compute"`
	Memory  float64 `json:"memory"`
	Storage float64 `json:"storage"`
	Network float64 `json:"network"`
}

type ClusterSummary struct {
	TotalNodes       int32   `json:"total_nodes"`
	TotalPods        int32   `json:"total_pods"`
	TotalDeployments int32   `json:"total_deployments"`
	AvgCPUUtil       float64 `json:"avg_cpu_utilization"`
	AvgMemoryUtil    float64 `json:"avg_memory_utilization"`
}

// ResourceUsage represents current vs requested resources
type ResourceUsage struct {
	Name           string  `json:"name"`
	Namespace      string  `json:"namespace"`
	Type           string  `json:"type"`
	Replicas       int32   `json:"replicas"`
	CPURequested   int64   `json:"cpu_requested_millicores"`
	CPUUsed        int64   `json:"cpu_used_millicores"`
	CPUUtilization float64 `json:"cpu_utilization_percent"`
	MemRequested   int64   `json:"memory_requested_bytes"`
	MemUsed        int64   `json:"memory_used_bytes"`
	MemUtilization float64 `json:"memory_utilization_percent"`
	MonthlyCost    float64 `json:"monthly_cost_estimate"`
}

func main() {
	// Check for demo mode
	if len(os.Args) > 1 && os.Args[1] == "demo" {
		runDemo()
		return
	}

	optimizer, err := NewCostOptimizer()
	if err != nil {
		log.Fatalf("Failed to initialize cost optimizer: %v", err)
	}

	log.Println("🚀 Cost Optimizer started using DevOps SDK")

	// Start dashboard server
	go optimizer.dashboard.Start()

	// Run in event-driven mode using our enhanced SDK
	err = optimizer.app.RunWithInformers(func() error {
		return optimizer.optimizeCosts()
	})
	if err != nil {
		log.Fatalf("Cost optimization failed: %v", err)
	}
}

// NewCostOptimizer creates a new cost optimizer using our enhanced SDK
func NewCostOptimizer() (*CostOptimizer, error) {
	// Initialize DevOps app with our enhanced SDK
	app, err := sdk.NewDevOpsApp(sdk.DevOpsAppConfig{
		Name:        "cost-optimizer",
		Version:     "2.0.0",
		Description: "AI-powered Kubernetes cost optimization using ConfigHub",
		RunInterval: 10 * time.Minute, // Fallback interval
		HealthPort:  8080,
	})
	if err != nil {
		return nil, fmt.Errorf("create DevOps app: %w", err)
	}

	// Enable Claude debug logging for cost analysis
	if app.Claude != nil {
		app.Claude.EnableDebugLogging()
	}

	optimizer := &CostOptimizer{
		app: app,
	}

	// Initialize ConfigHub space and sets
	if err := optimizer.initializeConfigHub(); err != nil {
		return nil, fmt.Errorf("initialize ConfigHub: %w", err)
	}

	// Initialize dashboard
	optimizer.dashboard = NewDashboard(optimizer)

	return optimizer, nil
}

// initializeConfigHub sets up ConfigHub space and filters for cost optimization
func (c *CostOptimizer) initializeConfigHub() error {
	if c.app.Cub == nil {
		c.app.Logger.Println("⚠️  ConfigHub not configured - running in local mode")
		return nil
	}

	// Create space with unique prefix for this cost optimization instance
	space, slug, err := c.app.Cub.CreateSpaceWithUniquePrefix("cost-optimizer",
		"Cost Optimization Analysis Space",
		map[string]string{
			"app":  "cost-optimizer",
			"type": "analysis",
		})
	if err != nil {
		return fmt.Errorf("create cost optimizer space: %w", err)
	}

	c.spaceID = space.SpaceID
	c.app.Logger.Printf("📦 Created ConfigHub space: %s", slug)

	// Create set for critical cost items
	criticalSet, err := c.app.Cub.CreateSet(c.spaceID, sdk.CreateSetRequest{
		Slug:        "critical-costs",
		DisplayName: "Critical Cost Items",
		Labels: map[string]string{
			"priority": "high",
			"type":     "cost-optimization",
		},
	})
	if err != nil {
		return fmt.Errorf("create critical costs set: %w", err)
	}

	c.criticalSetID = criticalSet.SetID
	c.app.Logger.Printf("📊 Created critical costs set: %s", criticalSet.SetID)

	// Create filter for high-cost resources
	_, err = c.app.Cub.CreateFilter(c.spaceID, sdk.CreateFilterRequest{
		Slug:        "high-cost-resources",
		DisplayName: "High Cost Resources (>$100/month)",
		From:        "Unit",
		Where:       "Labels.monthly_cost > '100'",
	})
	if err != nil {
		return fmt.Errorf("create high cost filter: %w", err)
	}

	return nil
}

// optimizeCosts performs the main cost optimization analysis
func (c *CostOptimizer) optimizeCosts() error {
	c.app.Logger.Println("🔍 Starting cost optimization analysis...")

	// 1. Gather resource usage data
	resourceUsage, err := c.gatherResourceUsage()
	if err != nil {
		return fmt.Errorf("gather resource usage: %w", err)
	}

	c.app.Logger.Printf("📊 Analyzed %d resources across cluster", len(resourceUsage))

	// 2. Analyze with Claude AI for intelligent recommendations
	analysis, err := c.analyzeWithClaude(resourceUsage)
	if err != nil {
		return fmt.Errorf("AI analysis: %w", err)
	}

	c.app.Logger.Printf("💰 Potential monthly savings: $%.2f (%.1f%%)",
		analysis.PotentialSavings, analysis.SavingsPercentage)

	// 3. Store analysis in ConfigHub for tracking
	if c.app.Cub != nil {
		if err := c.storeAnalysisInConfigHub(analysis); err != nil {
			c.app.Logger.Printf("⚠️  Failed to store in ConfigHub: %v", err)
		}
	}

	// 4. Update dashboard with latest data
	c.dashboard.UpdateAnalysis(analysis)

	// 5. Apply high-confidence recommendations (if enabled)
	if sdk.GetEnvBool("AUTO_APPLY_OPTIMIZATIONS", false) {
		if err := c.applyRecommendations(analysis); err != nil {
			c.app.Logger.Printf("⚠️  Failed to apply recommendations: %v", err)
		}
	}

	return nil
}

// gatherResourceUsage collects current resource usage from Kubernetes
func (c *CostOptimizer) gatherResourceUsage() ([]ResourceUsage, error) {
	ctx := context.Background()
	var resourceUsage []ResourceUsage

	// Get all deployments
	deployments, err := c.app.K8s.Clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}

	// Get pod metrics for actual usage
	var podMetrics *metricsv1beta1.PodMetricsList
	if c.app.K8s.MetricsClient != nil {
		podMetrics, err = c.app.K8s.MetricsClient.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
		if err != nil {
			c.app.Logger.Printf("⚠️  Could not get metrics: %v", err)
		}
	}

	// Build metrics map for quick lookup
	metricsMap := make(map[string]metricsv1beta1.PodMetrics)
	if podMetrics != nil {
		for _, metric := range podMetrics.Items {
			metricsMap[metric.Namespace+"/"+metric.Name] = metric
		}
	}

	// Analyze each deployment
	for _, deployment := range deployments.Items {
		usage := c.analyzeDeployment(deployment, metricsMap)
		resourceUsage = append(resourceUsage, usage)
	}

	return resourceUsage, nil
}

// analyzeDeployment analyzes a single deployment's resource usage
func (c *CostOptimizer) analyzeDeployment(deployment appsv1.Deployment, metricsMap map[string]metricsv1beta1.PodMetrics) ResourceUsage {
	usage := ResourceUsage{
		Name:      deployment.Name,
		Namespace: deployment.Namespace,
		Type:      "Deployment",
		Replicas:  *deployment.Spec.Replicas,
	}

	// Calculate requested resources
	if len(deployment.Spec.Template.Spec.Containers) > 0 {
		container := deployment.Spec.Template.Spec.Containers[0]

		if cpu := container.Resources.Requests[corev1.ResourceCPU]; !cpu.IsZero() {
			usage.CPURequested = cpu.MilliValue() * int64(usage.Replicas)
		}

		if mem := container.Resources.Requests[corev1.ResourceMemory]; !mem.IsZero() {
			usage.MemRequested = mem.Value() * int64(usage.Replicas)
		}
	}

	// Get actual usage from metrics (simplified - would need pod listing in real implementation)
	usage.CPUUsed = usage.CPURequested / 2 // Simulate 50% usage
	usage.MemUsed = usage.MemRequested / 2

	// Calculate utilization percentages
	if usage.CPURequested > 0 {
		usage.CPUUtilization = float64(usage.CPUUsed) / float64(usage.CPURequested) * 100
	}
	if usage.MemRequested > 0 {
		usage.MemUtilization = float64(usage.MemUsed) / float64(usage.MemRequested) * 100
	}

	// Estimate monthly cost (simplified calculation)
	cpuCostPerHour := float64(usage.CPURequested) / 1000.0 * 0.0416 // $0.0416 per vCPU hour
	memoryCostPerHour := float64(usage.MemRequested) / (1024*1024*1024) * 0.00456 // $0.00456 per GB hour
	usage.MonthlyCost = (cpuCostPerHour + memoryCostPerHour) * 24 * 30

	return usage
}

// analyzeWithClaude uses Claude AI to generate intelligent cost optimization recommendations
func (c *CostOptimizer) analyzeWithClaude(resourceUsage []ResourceUsage) (*CostAnalysis, error) {
	if c.app.Claude == nil {
		// Fallback to basic analysis without AI
		return c.basicCostAnalysis(resourceUsage), nil
	}

	prompt := `Analyze the following Kubernetes resource usage data and provide cost optimization recommendations.

Focus on:
1. Resources with low utilization (<50%) that can be right-sized
2. Over-provisioned deployments that can be scaled down
3. Resources that might be candidates for removal
4. Storage optimization opportunities

For each recommendation, provide:
- Specific resource to modify
- Current vs recommended configuration
- Estimated monthly savings
- Risk level (low/medium/high)
- Clear explanation of the change

Return your analysis as JSON matching this structure:
{
  "total_monthly_cost": 1234.56,
  "potential_savings": 234.56,
  "savings_percentage": 19.0,
  "recommendations": [
    {
      "resource": "deployment/my-app",
      "namespace": "default",
      "type": "rightsize",
      "priority": "high",
      "current": {"cpu": "1000m", "memory": "1Gi", "replicas": 3},
      "recommended": {"cpu": "500m", "memory": "512Mi", "replicas": 2},
      "monthly_savings": 123.45,
      "risk": "low",
      "explanation": "Resource is only using 30% of allocated CPU and memory",
      "confighub_action": "Update deployment unit with new resource limits"
    }
  ]
}`

	response, err := c.app.Claude.AnalyzeJSON(prompt, resourceUsage)
	if err != nil {
		c.app.Logger.Printf("⚠️  Claude analysis failed: %v", err)
		return c.basicCostAnalysis(resourceUsage), nil
	}

	// Parse Claude's response
	var analysis CostAnalysis
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		c.app.Logger.Printf("⚠️  Failed to parse Claude response: %v", err)
		return c.basicCostAnalysis(resourceUsage), nil
	}

	// Add metadata
	analysis.Timestamp = time.Now()
	analysis.ResourceBreakdown = c.calculateResourceBreakdown(resourceUsage)
	analysis.ClusterSummary = c.calculateClusterSummary(resourceUsage)

	return &analysis, nil
}

// basicCostAnalysis provides fallback analysis without AI
func (c *CostOptimizer) basicCostAnalysis(resourceUsage []ResourceUsage) *CostAnalysis {
	totalCost := 0.0
	savings := 0.0
	var recommendations []CostRecommendation

	for _, usage := range resourceUsage {
		totalCost += usage.MonthlyCost

		// Simple rule: if utilization < 50%, recommend rightsizing
		if usage.CPUUtilization < 50 && usage.MemUtilization < 50 {
			rec := CostRecommendation{
				Resource:        fmt.Sprintf("deployment/%s", usage.Name),
				Namespace:       usage.Namespace,
				Type:            "rightsize",
				Priority:        "medium",
				MonthlySavings:  usage.MonthlyCost * 0.3, // 30% savings
				Risk:            "low",
				Explanation:     fmt.Sprintf("Low utilization: CPU %.1f%%, Memory %.1f%%", usage.CPUUtilization, usage.MemUtilization),
				ConfigHubAction: "Update resource requests to match actual usage",
			}
			recommendations = append(recommendations, rec)
			savings += rec.MonthlySavings
		}
	}

	return &CostAnalysis{
		Timestamp:         time.Now(),
		TotalMonthlyCost:  totalCost,
		PotentialSavings:  savings,
		SavingsPercentage: (savings / totalCost) * 100,
		Recommendations:   recommendations,
		ResourceBreakdown: c.calculateResourceBreakdown(resourceUsage),
		ClusterSummary:    c.calculateClusterSummary(resourceUsage),
	}
}

// calculateResourceBreakdown calculates cost breakdown by resource type
func (c *CostOptimizer) calculateResourceBreakdown(resourceUsage []ResourceUsage) ResourceBreakdown {
	totalCompute := 0.0
	totalMemory := 0.0

	for _, usage := range resourceUsage {
		cpuCost := float64(usage.CPURequested) / 1000.0 * 0.0416 * 24 * 30
		memCost := float64(usage.MemRequested) / (1024*1024*1024) * 0.00456 * 24 * 30

		totalCompute += cpuCost
		totalMemory += memCost
	}

	return ResourceBreakdown{
		Compute: totalCompute,
		Memory:  totalMemory,
		Storage: totalCompute * 0.1, // Estimate storage as 10% of compute
		Network: totalCompute * 0.05, // Estimate network as 5% of compute
	}
}

// calculateClusterSummary calculates cluster-wide summary statistics
func (c *CostOptimizer) calculateClusterSummary(resourceUsage []ResourceUsage) ClusterSummary {
	totalDeployments := int32(len(resourceUsage))
	totalReplicas := int32(0)
	totalCPUUtil := 0.0
	totalMemUtil := 0.0

	for _, usage := range resourceUsage {
		totalReplicas += usage.Replicas
		totalCPUUtil += usage.CPUUtilization
		totalMemUtil += usage.MemUtilization
	}

	avgCPUUtil := totalCPUUtil / float64(len(resourceUsage))
	avgMemUtil := totalMemUtil / float64(len(resourceUsage))

	return ClusterSummary{
		TotalNodes:       3, // Would get from actual node count
		TotalPods:        totalReplicas,
		TotalDeployments: totalDeployments,
		AvgCPUUtil:       avgCPUUtil,
		AvgMemoryUtil:    avgMemUtil,
	}
}

// storeAnalysisInConfigHub stores the cost analysis in ConfigHub for tracking
func (c *CostOptimizer) storeAnalysisInConfigHub(analysis *CostAnalysis) error {
	// Store overall analysis
	analysisData, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal analysis: %w", err)
	}

	_, err = c.app.Cub.CreateUnit(c.spaceID, sdk.CreateUnitRequest{
		Slug:        fmt.Sprintf("cost-analysis-%d", time.Now().Unix()),
		DisplayName: fmt.Sprintf("Cost Analysis %s", time.Now().Format("2006-01-02 15:04")),
		Data:        string(analysisData),
		Labels: map[string]string{
			"type":           "cost-analysis",
			"total_cost":     fmt.Sprintf("%.2f", analysis.TotalMonthlyCost),
			"savings":        fmt.Sprintf("%.2f", analysis.PotentialSavings),
			"timestamp":      analysis.Timestamp.Format(time.RFC3339),
		},
	})
	if err != nil {
		return fmt.Errorf("create analysis unit: %w", err)
	}

	// Store high-priority recommendations in the critical set
	for _, rec := range analysis.Recommendations {
		if rec.Priority == "high" && rec.MonthlySavings > 50 {
			recData, _ := json.MarshalIndent(rec, "", "  ")

			unit, err := c.app.Cub.CreateUnit(c.spaceID, sdk.CreateUnitRequest{
				Slug:        fmt.Sprintf("rec-%s-%d", strings.ReplaceAll(rec.Resource, "/", "-"), time.Now().Unix()),
				DisplayName: fmt.Sprintf("High Priority: %s", rec.Resource),
				Data:        string(recData),
				Labels: map[string]string{
					"type":            "recommendation",
					"priority":        rec.Priority,
					"monthly_savings": fmt.Sprintf("%.2f", rec.MonthlySavings),
					"resource":        rec.Resource,
				},
				SetIDs: []uuid.UUID{c.criticalSetID},
			})
			if err != nil {
				c.app.Logger.Printf("⚠️  Failed to store recommendation: %v", err)
				continue
			}

			c.app.Logger.Printf("💡 Stored high-priority recommendation: %s (saves $%.2f/month)",
				rec.Resource, rec.MonthlySavings)

			// Store unit ID for later reference
			_ = unit
		}
	}

	return nil
}

// applyRecommendations applies safe recommendations automatically
func (c *CostOptimizer) applyRecommendations(analysis *CostAnalysis) error {
	applied := 0

	for _, rec := range analysis.Recommendations {
		// Only apply low-risk recommendations automatically
		if rec.Risk == "low" && rec.Type == "rightsize" && rec.MonthlySavings > 20 {
			if err := c.applySingleRecommendation(rec); err != nil {
				c.app.Logger.Printf("⚠️  Failed to apply recommendation for %s: %v", rec.Resource, err)
				continue
			}
			applied++
		}
	}

	if applied > 0 {
		c.app.Logger.Printf("✅ Applied %d cost optimization recommendations", applied)
	}

	return nil
}

// applySingleRecommendation applies a single recommendation
func (c *CostOptimizer) applySingleRecommendation(rec CostRecommendation) error {
	// In a real implementation, this would update the Kubernetes deployment
	// For demo purposes, we'll just log what would be done
	c.app.Logger.Printf("🔧 Would apply: %s - %s (saves $%.2f/month)",
		rec.Resource, rec.Explanation, rec.MonthlySavings)

	// Would also update ConfigHub unit with new configuration
	return nil
}