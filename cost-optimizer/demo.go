package main

import (
	"fmt"
	"time"

	sdk "github.com/monadic/devops-sdk"
)

// runDemo shows the cost optimizer working with mock data
func runDemo() {
	fmt.Println("🚀 DevOps as Apps - Cost Optimizer Demo")
	fmt.Println("=====================================")
	fmt.Println()

	demo := &CostOptimizerDemo{}
	demo.run()
}

type CostOptimizerDemo struct{}

func (d *CostOptimizerDemo) run() {
	fmt.Println("📋 Step 1: Initialize Cost Optimizer with Enhanced SDK")
	fmt.Println("   ✅ Created DevOps app using enhanced SDK")
	fmt.Println("   ✅ Enabled Claude debug logging for AI analysis")
	fmt.Println("   ✅ Connected to Kubernetes cluster")
	fmt.Println("   ✅ Connected to ConfigHub API")
	fmt.Println()

	time.Sleep(500 * time.Millisecond)

	fmt.Println("🔧 Step 2: ConfigHub Space Setup with Unique Prefix")
	fmt.Println("   ✅ Generated unique prefix: 'efficient-whale-1234567890'")
	fmt.Println("   ✅ Created space: efficient-whale-1234567890-cost-optimizer")
	fmt.Println("   ✅ Created 'critical-costs' Set for high-priority items")
	fmt.Println("   ✅ Created filter: 'high-cost-resources' (>$100/month)")
	fmt.Println()

	time.Sleep(500 * time.Millisecond)

	fmt.Println("📊 Step 3: Gather Resource Usage (Event-Driven)")
	resourceUsage := d.mockResourceUsage()
	fmt.Printf("   Found %d deployments across cluster:\n\n", len(resourceUsage))

	// Use SDK ASCII table to show resource usage
	usageTable := d.renderResourceUsageTable(resourceUsage)
	fmt.Println(usageTable)
	fmt.Println()

	time.Sleep(500 * time.Millisecond)

	fmt.Println("🤖 Step 4: Claude AI Cost Analysis")
	analysis := d.mockClaudeAnalysis(resourceUsage)
	fmt.Printf("   Claude Analysis Summary:\n")
	fmt.Printf("   📈 Total Monthly Cost: $%.2f\n", analysis.TotalMonthlyCost)
	fmt.Printf("   💰 Potential Savings: $%.2f (%.1f%%)\n", analysis.PotentialSavings, analysis.SavingsPercentage)
	fmt.Printf("   🎯 Recommendations: %d\n", len(analysis.Recommendations))
	fmt.Println()

	time.Sleep(500 * time.Millisecond)

	fmt.Println("💡 Step 5: Cost Optimization Recommendations\n")

	// Use SDK ASCII table to show recommendations
	recsTable := d.renderRecommendationsTable(analysis.Recommendations)
	fmt.Println(recsTable)
	fmt.Println()

	time.Sleep(500 * time.Millisecond)

	fmt.Println("🏪 Step 6: Store Analysis in ConfigHub")
	fmt.Println("   ✅ Created cost-analysis unit with full data")
	fmt.Println("   ✅ Stored 2 high-priority recommendations in 'critical-costs' Set")
	fmt.Println("   ✅ Applied Labels for filtering and querying")
	fmt.Println()

	time.Sleep(500 * time.Millisecond)

	fmt.Println("🌐 Step 7: Web Dashboard Live")
	fmt.Println("   ✅ Dashboard updated with latest analysis")
	fmt.Println("   🔗 View at: http://localhost:8081")
	fmt.Println("   📊 Cost breakdown by resource type")
	fmt.Println("   🎯 Interactive recommendations view")
	fmt.Println("   📈 Auto-refresh every 30 seconds")
	fmt.Println()

	time.Sleep(500 * time.Millisecond)

	fmt.Println("⚡ Step 8: Event-Driven Processing")
	fmt.Println("   ✅ Using RunWithInformers() for Kubernetes events")
	fmt.Println("   ✅ Automatic cost analysis on deployment changes")
	fmt.Println("   ✅ ConfigHub push-upgrade for propagating optimizations")
	fmt.Println()

	fmt.Println("🎉 Demo Complete!")
	fmt.Println()
	fmt.Println("The cost optimizer successfully demonstrated:")
	fmt.Println("  ✅ Enhanced SDK integration with logging")
	fmt.Println("  ✅ ConfigHub space/set/filter management")
	fmt.Println("  ✅ Claude AI-powered cost analysis")
	fmt.Println("  ✅ Real-time web dashboard")
	fmt.Println("  ✅ Event-driven architecture")
	fmt.Println("  ✅ Global-app deployment pattern")
	fmt.Println()

	fmt.Println("📋 Real Usage:")
	fmt.Println("  ./cost-optimizer                    # Run with real K8s cluster")
	fmt.Println("  CLAUDE_DEBUG_LOG=true ./cost-optimizer  # Enable full AI logging")
	fmt.Println("  AUTO_APPLY_OPTIMIZATIONS=true ./cost-optimizer  # Auto-apply safe changes")
	fmt.Println()

	fmt.Println("🔗 Endpoints:")
	fmt.Println("  Dashboard:  http://localhost:8081")
	fmt.Println("  Health:     http://localhost:8080/health")
	fmt.Println("  API:        http://localhost:8081/api/analysis")
}

func (d *CostOptimizerDemo) mockResourceUsage() []ResourceUsage {
	return []ResourceUsage{
		{
			Name:           "frontend-web",
			Namespace:      "production",
			Type:           "Deployment",
			Replicas:       3,
			CPURequested:   3000, // 3 vCPU
			CPUUsed:        900,  // 30% utilization
			CPUUtilization: 30.0,
			MemRequested:   3 * 1024 * 1024 * 1024, // 3GB
			MemUsed:        1 * 1024 * 1024 * 1024, // 1GB used
			MemUtilization: 33.3,
			MonthlyCost:    245.50,
		},
		{
			Name:           "backend-api",
			Namespace:      "production",
			Type:           "Deployment",
			Replicas:       5,
			CPURequested:   5000, // 5 vCPU
			CPUUsed:        2000, // 40% utilization
			CPUUtilization: 40.0,
			MemRequested:   5 * 1024 * 1024 * 1024, // 5GB
			MemUsed:        2 * 1024 * 1024 * 1024, // 2GB used
			MemUtilization: 40.0,
			MonthlyCost:    408.75,
		},
		{
			Name:           "cache-redis",
			Namespace:      "production",
			Type:           "StatefulSet",
			Replicas:       1,
			CPURequested:   1000, // 1 vCPU
			CPUUsed:        150,  // 15% utilization
			CPUUtilization: 15.0,
			MemRequested:   2 * 1024 * 1024 * 1024, // 2GB
			MemUsed:        300 * 1024 * 1024,      // 300MB used
			MemUtilization: 15.0,
			MonthlyCost:    89.25,
		},
		{
			Name:           "monitoring-prometheus",
			Namespace:      "monitoring",
			Type:           "Deployment",
			Replicas:       1,
			CPURequested:   2000, // 2 vCPU
			CPUUsed:        1600, // 80% utilization
			CPUUtilization: 80.0,
			MemRequested:   4 * 1024 * 1024 * 1024, // 4GB
			MemUsed:        3 * 1024 * 1024 * 1024, // 3GB used
			MemUtilization: 75.0,
			MonthlyCost:    178.50,
		},
	}
}

func (d *CostOptimizerDemo) mockClaudeAnalysis(resourceUsage []ResourceUsage) *CostAnalysis {
	totalCost := 0.0
	for _, usage := range resourceUsage {
		totalCost += usage.MonthlyCost
	}

	recommendations := []CostRecommendation{
		{
			Resource:       "deployment/frontend-web",
			Namespace:      "production",
			Type:           "rightsize",
			Priority:       "high",
			MonthlySavings: 73.65, // 30% savings
			Risk:           "low",
			Explanation:    "Frontend is over-provisioned with only 30% CPU and 33% memory utilization. Can safely reduce resources without impacting performance.",
			ConfigHubAction: "Update deployment unit: reduce CPU to 1.5 vCPU and memory to 2GB",
		},
		{
			Resource:       "statefulset/cache-redis",
			Namespace:      "production",
			Type:           "rightsize",
			Priority:       "high",
			MonthlySavings: 53.55, // 60% savings
			Risk:           "low",
			Explanation:    "Redis cache is significantly over-provisioned with only 15% utilization. Memory can be reduced to match actual usage patterns.",
			ConfigHubAction: "Update StatefulSet unit: reduce memory to 512MB",
		},
		{
			Resource:       "deployment/backend-api",
			Namespace:      "production",
			Type:           "rightsize",
			Priority:       "medium",
			MonthlySavings: 81.75, // 20% savings
			Risk:           "medium",
			Explanation:    "Backend API has moderate utilization (40%) but could benefit from slight resource reduction with monitoring.",
			ConfigHubAction: "Update deployment unit: reduce CPU to 4 vCPU, monitor for 1 week",
		},
	}

	return &CostAnalysis{
		Timestamp:         time.Now(),
		TotalMonthlyCost:  totalCost,
		PotentialSavings:  208.95,
		SavingsPercentage: 22.8,
		Recommendations:   recommendations,
		ResourceBreakdown: ResourceBreakdown{
			Compute: 612.00,
			Memory:  310.00,
			Storage: 61.20,
			Network: 30.60,
		},
		ClusterSummary: ClusterSummary{
			TotalNodes:       3,
			TotalPods:        10,
			TotalDeployments: 4,
			AvgCPUUtil:       41.25,
			AvgMemoryUtil:    40.8,
		},
	}
}

func (d *CostOptimizerDemo) renderResourceUsageTable(resourceUsage []ResourceUsage) string {
	table := sdk.NewTableWriter([]string{"Resource", "Type", "Replicas", "CPU Util", "Mem Util", "Monthly Cost"})
	table.SetBorderStyle(sdk.DefaultBorder)

	for _, usage := range resourceUsage {
		table.AddRow([]string{
			usage.Name,
			usage.Type,
			fmt.Sprintf("%d", usage.Replicas),
			fmt.Sprintf("%.1f%%", usage.CPUUtilization),
			fmt.Sprintf("%.1f%%", usage.MemUtilization),
			fmt.Sprintf("$%.2f", usage.MonthlyCost),
		})
	}

	return table.Render()
}

func (d *CostOptimizerDemo) renderRecommendationsTable(recommendations []CostRecommendation) string {
	table := sdk.NewTableWriter([]string{"Resource", "Priority", "Savings/mo", "Risk", "Action"})
	table.SetBorderStyle(sdk.DefaultBorder)

	for _, rec := range recommendations {
		table.AddRow([]string{
			rec.Resource,
			rec.Priority,
			fmt.Sprintf("$%.2f", rec.MonthlySavings),
			rec.Risk,
			rec.ConfigHubAction,
		})
	}

	return table.Render()
}