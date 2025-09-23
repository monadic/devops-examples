package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	sdk "github.com/monadic/devops-sdk"
)

// OpenCostClient provides integration with OpenCost API
type OpenCostClient struct {
	baseURL string
	client  *http.Client
}

// NewOpenCostClient creates a new OpenCost client
func NewOpenCostClient(baseURL string) *OpenCostClient {
	if baseURL == "" {
		// Default to in-cluster service
		baseURL = "http://opencost.opencost.svc.cluster.local:9003"
	}
	
	return &OpenCostClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// OpenCostAllocation represents cost data from OpenCost
type OpenCostAllocation struct {
	Name      string                 `json:"name"`
	Start     string                 `json:"start"`
	End       string                 `json:"end"`
	CPUCost   float64                `json:"cpuCost"`
	GPUCost   float64                `json:"gpuCost"`
	RAMCost   float64                `json:"ramCost"`
	PVCost    float64                `json:"pvCost"`
	TotalCost float64                `json:"totalCost"`
	Properties map[string]interface{} `json:"properties"`
}

// OpenCostResponse represents the API response
type OpenCostResponse struct {
	Code int                              `json:"code"`
	Data []map[string]OpenCostAllocation `json:"data"`
	Message string                        `json:"message,omitempty"`
}

// GetAllocationData fetches real cost data from OpenCost
func (oc *OpenCostClient) GetAllocationData(window string, aggregate string) (*OpenCostResponse, error) {
	// Construct API URL
	// Example: /allocation/compute?window=1d&aggregate=namespace
	url := fmt.Sprintf("%s/allocation/compute?window=%s&aggregate=%s",
		oc.baseURL, window, aggregate)
	
	fmt.Printf("[OpenCost] Fetching allocation data from: %s\n", url)
	
	resp, err := oc.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OpenCost data: %v", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenCost API error (status %d): %s",
			resp.StatusCode, string(body))
	}
	
	var result OpenCostResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse OpenCost response: %v", err)
	}
	
	fmt.Printf("[OpenCost] Retrieved %d allocation entries\n", len(result.Data))
	
	return &result, nil
}

// ConvertToResourceUsage converts OpenCost data to our ResourceUsage format
func (oc *OpenCostClient) ConvertToResourceUsage(allocations *OpenCostResponse) []ResourceUsage {
	var resources []ResourceUsage
	
	for _, dayData := range allocations.Data {
		for name, allocation := range dayData {
			// Extract namespace and deployment from name
			namespace := "default"
			if props, ok := allocation.Properties["namespace"].(string); ok {
				namespace = props
			}
			
			// Convert OpenCost allocation to ResourceUsage
			resource := ResourceUsage{
				Name:        name,
				Type:        "deployment",
				Namespace:   namespace,
				MonthlyCost: allocation.TotalCost * 30, // Convert daily to monthly
				CPUCost:     allocation.CPUCost * 30,
				MemoryCost:  allocation.RAMCost * 30,
				StorageCost: allocation.PVCost * 30,
				GPUCost:     allocation.GPUCost * 30,
				
				// Extract utilization if available
				CPUUtilization: extractUtilization(allocation.Properties, "cpuUtilization"),
				MemUtilization: extractUtilization(allocation.Properties, "ramUtilization"),
			}
			
			resources = append(resources, resource)
		}
	}
	
	return resources
}

// extractUtilization safely extracts utilization from properties
func extractUtilization(props map[string]interface{}, key string) float64 {
	if val, ok := props[key].(float64); ok {
		return val * 100 // Convert to percentage
	}
	return 0.0
}

// IntegrateWithOpenCost enhances cost optimizer with real OpenCost data
func (c *CostOptimizer) IntegrateWithOpenCost() error {
	fmt.Println("\n🔌 Integrating with OpenCost for real cost data...")
	
	// Check if OpenCost is available
	opencostURL := os.Getenv("OPENCOST_URL")
	if opencostURL == "" {
		// Try to detect OpenCost service in cluster
		opencostURL = "http://opencost.opencost.svc.cluster.local:9003"
	}
	
	oc := NewOpenCostClient(opencostURL)
	
	// Test OpenCost connectivity
	testURL := fmt.Sprintf("%s/healthz", opencostURL)
	resp, err := oc.client.Get(testURL)
	if err != nil {
		fmt.Printf("[OpenCost] Not available at %s, using estimated costs\n", opencostURL)
		return nil // Fallback to estimates
	}
	resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[OpenCost] Health check failed (status %d), using estimates\n",
			resp.StatusCode)
		return nil
	}
	
	fmt.Printf("[OpenCost] ✓ Connected to OpenCost at %s\n", opencostURL)
	
	// Fetch real cost data from OpenCost
	allocations, err := oc.GetAllocationData("1d", "namespace")
	if err != nil {
		fmt.Printf("[OpenCost] Error fetching data: %v\n", err)
		return err
	}
	
	// Convert OpenCost data to our format
	opencostResources := oc.ConvertToResourceUsage(allocations)
	
	if len(opencostResources) > 0 {
		fmt.Printf("[OpenCost] ✓ Using real cost data for %d resources\n",
			len(opencostResources))
		
		// Merge with existing resource data
		mergedResources := c.mergeResourceData(c.resources, opencostResources)
		c.resources = mergedResources
		
		// Store OpenCost data in ConfigHub
		c.storeOpenCostData(allocations)
	}
	
	return nil
}

// mergeResourceData merges Kubernetes metrics with OpenCost cost data
func (c *CostOptimizer) mergeResourceData(k8sResources, opencostResources []ResourceUsage) []ResourceUsage {
	// Create map for quick lookup
	opencostMap := make(map[string]ResourceUsage)
	for _, res := range opencostResources {
		key := fmt.Sprintf("%s/%s", res.Namespace, res.Name)
		opencostMap[key] = res
	}
	
	// Merge data
	var merged []ResourceUsage
	for _, k8sRes := range k8sResources {
		key := fmt.Sprintf("%s/%s", k8sRes.Namespace, k8sRes.Name)
		
		if ocRes, found := opencostMap[key]; found {
			// Use real costs from OpenCost
			k8sRes.MonthlyCost = ocRes.MonthlyCost
			k8sRes.CPUCost = ocRes.CPUCost
			k8sRes.MemoryCost = ocRes.MemoryCost
			k8sRes.StorageCost = ocRes.StorageCost
			k8sRes.GPUCost = ocRes.GPUCost
			
			// Use utilization from OpenCost if available
			if ocRes.CPUUtilization > 0 {
				k8sRes.CPUUtilization = ocRes.CPUUtilization
			}
			if ocRes.MemUtilization > 0 {
				k8sRes.MemUtilization = ocRes.MemUtilization
			}
			
			fmt.Printf("[OpenCost] Updated %s with real costs: $%.2f/month\n",
				key, k8sRes.MonthlyCost)
		}
		
		merged = append(merged, k8sRes)
	}
	
	return merged
}

// storeOpenCostData stores OpenCost data in ConfigHub for audit trail
func (c *CostOptimizer) storeOpenCostData(data *OpenCostResponse) error {
	fmt.Println("[ConfigHub] Storing OpenCost data for audit trail...")
	
	// Create unit with OpenCost data
	unitName := fmt.Sprintf("opencost-data-%d", time.Now().Unix())
	unitData := map[string]interface{}{
		"source":    "opencost",
		"timestamp": time.Now().Format(time.RFC3339),
		"data":      data,
	}
	
	unitJSON, err := json.Marshal(unitData)
	if err != nil {
		return fmt.Errorf("failed to marshal OpenCost data: %v", err)
	}
	
	_, err = c.app.Cub.CreateUnit(c.spaceID, sdk.CreateUnitRequest{
		Slug:        unitName,
		DisplayName: fmt.Sprintf("OpenCost Data - %s", time.Now().Format("2006-01-02")),
		Data:        string(unitJSON),
		Labels: map[string]string{
			"type":   "opencost-data",
			"source": "opencost",
		},
	})
	
	if err != nil {
		fmt.Printf("[ConfigHub] Warning: Could not store OpenCost data: %v\n", err)
		return nil // Non-critical error
	}
	
	fmt.Printf("[ConfigHub] ✓ Stored OpenCost data as unit: %s\n", unitName)
	return nil
}

