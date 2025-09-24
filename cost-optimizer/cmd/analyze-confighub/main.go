package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	sdk "github.com/monadic/devops-sdk"
)

func main() {
	var (
		spaceID    = flag.String("space", "", "ConfigHub space ID to analyze")
		hierarchy  = flag.Bool("hierarchy", false, "Analyze full environment hierarchy")
		storeBack  = flag.Bool("store", false, "Store cost annotations back to ConfigHub")
		outputJSON = flag.Bool("json", false, "Output as JSON")
	)
	flag.Parse()

	// Initialize SDK app
	app, err := sdk.NewApp("confighub-cost-analyzer", "1.0.0")
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// Get space ID from environment or flag
	if *spaceID == "" {
		*spaceID = os.Getenv("CONFIGHUB_SPACE_ID")
	}
	if *spaceID == "" {
		// Try to read from .cub-project file
		if data, err := os.ReadFile(".cub-project"); err == nil {
			*spaceID = string(data)
		}
	}
	if *spaceID == "" {
		fmt.Println("Usage: analyze-confighub -space <space-id>")
		fmt.Println("   or: export CONFIGHUB_SPACE_ID=<space-id>")
		fmt.Println("   or: create .cub-project file with space ID")
		os.Exit(1)
	}

	// Parse space ID
	spaceUUID, err := uuid.Parse(*spaceID)
	if err != nil {
		log.Fatalf("Invalid space ID format: %v", err)
	}

	// Create SDK cost analyzer
	analyzer := sdk.NewCostAnalyzer(app, spaceUUID)

	fmt.Println("🔍 ConfigHub Cost Analyzer")
	fmt.Println("══════════════════════════")
	fmt.Printf("Space: %s\n", *spaceID)
	fmt.Printf("Time: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// Perform analysis
	var analysis *sdk.SpaceCostAnalysis
	if *hierarchy {
		fmt.Println("Analyzing full environment hierarchy...")
		analysis, err = analyzer.AnalyzeHierarchy(*spaceID)
	} else {
		fmt.Println("Analyzing single space...")
		analysis, err = analyzer.AnalyzeSpace()
	}

	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	// Store annotations if requested
	if *storeBack {
		fmt.Println("\n📝 Storing cost annotations in ConfigHub...")
		if err := analyzer.StoreAnalysisInConfigHub(analysis); err != nil {
			log.Printf("Warning: Failed to store some annotations: %v", err)
		}
	}

	// Output results
	if *outputJSON {
		// JSON output for programmatic use
		outputJSON(analysis)
	} else {
		// Human-readable report
		report := analyzer.GenerateReport(analysis)
		fmt.Println(report)

		// Summary
		fmt.Println("\n💡 Key Insights:")
		fmt.Printf("• Total estimated monthly cost: $%.2f\n", analysis.TotalMonthlyCost)
		fmt.Printf("• Number of workloads analyzed: %d\n", len(analysis.Units))

		if len(analysis.Units) > 0 {
			// Find most expensive unit
			maxCost := 0.0
			var maxUnit sdk.UnitCostEstimate
			for _, unit := range analysis.Units {
				if unit.MonthlyCost > maxCost {
					maxCost = unit.MonthlyCost
					maxUnit = unit
				}
			}
			fmt.Printf("• Most expensive: %s ($%.2f/month)\n", maxUnit.UnitName, maxUnit.MonthlyCost)
		}

		fmt.Println("\n🚀 Next Steps:")
		fmt.Println("1. Deploy these configs to see actual usage")
		fmt.Println("2. Run cost-optimizer with OpenCost for real metrics")
		fmt.Println("3. Use Claude AI for optimization recommendations")
	}
}

func outputJSON(analysis *sdk.SpaceCostAnalysis) {
	// Use proper JSON marshaling
	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		log.Printf("Error marshaling JSON: %v", err)
		return
	}
	fmt.Println(string(data))
}