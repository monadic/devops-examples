package main

// PricingProvider defines cloud provider pricing
type PricingProvider struct {
	Name            string
	Region          string
	CPUHourly       float64 // Per vCPU per hour
	MemoryHourly    float64 // Per GB per hour
	StorageMonthly  float64 // Per GB per month
	EgressGB        float64 // Per GB
	IngressGB       float64 // Per GB (usually free)
	ControlPlaneHourly float64 // EKS/GKE/AKS cluster cost
}

// GetAWSPricing returns real AWS EKS pricing for different regions
func GetAWSPricing(region string) PricingProvider {
	// Based on m5.large instances (most common for EKS)
	// 2 vCPU, 8GB RAM = $0.096/hour
	switch region {
	case "us-east-1":
		return PricingProvider{
			Name:            "AWS EKS",
			Region:          region,
			CPUHourly:       0.024,    // $0.048/hour for 2 vCPU
			MemoryHourly:    0.006,    // $0.048/hour for 8GB
			StorageMonthly:  0.10,     // EBS gp3
			EgressGB:        0.09,     // Data transfer out
			IngressGB:       0.00,     // Free
			ControlPlaneHourly: 0.10,  // EKS control plane
		}
	case "us-west-2":
		return PricingProvider{
			Name:            "AWS EKS",
			Region:          region,
			CPUHourly:       0.024,
			MemoryHourly:    0.006,
			StorageMonthly:  0.10,
			EgressGB:        0.09,
			IngressGB:       0.00,
			ControlPlaneHourly: 0.10,
		}
	case "eu-west-1":
		return PricingProvider{
			Name:            "AWS EKS",
			Region:          region,
			CPUHourly:       0.026,    // ~10% more expensive
			MemoryHourly:    0.0065,
			StorageMonthly:  0.11,
			EgressGB:        0.09,
			IngressGB:       0.00,
			ControlPlaneHourly: 0.10,
		}
	default:
		// Default to us-east-1 pricing
		return GetAWSPricing("us-east-1")
	}
}

// GetGCPPricing returns real GKE pricing
func GetGCPPricing(region string) PricingProvider {
	// Based on e2-standard-2 (2 vCPU, 8GB)
	switch region {
	case "us-central1":
		return PricingProvider{
			Name:            "GCP GKE",
			Region:          region,
			CPUHourly:       0.021,    // Slightly cheaper than AWS
			MemoryHourly:    0.0055,
			StorageMonthly:  0.04,     // Persistent disk
			EgressGB:        0.12,     // More expensive egress
			IngressGB:       0.00,
			ControlPlaneHourly: 0.00,  // Free for zonal clusters
		}
	default:
		return GetGCPPricing("us-central1")
	}
}

// GetAzurePricing returns real AKS pricing
func GetAzurePricing(region string) PricingProvider {
	// Based on D2v3 (2 vCPU, 8GB)
	switch region {
	case "eastus":
		return PricingProvider{
			Name:            "Azure AKS",
			Region:          region,
			CPUHourly:       0.025,
			MemoryHourly:    0.006,
			StorageMonthly:  0.05,     // Managed disk
			EgressGB:        0.087,
			IngressGB:       0.00,
			ControlPlaneHourly: 0.00,  // Free
		}
	default:
		return GetAzurePricing("eastus")
	}
}

// CalculateRealCost calculates cost using actual cloud pricing
func CalculateRealCost(cpuCores float64, memoryGB float64, storageGB float64, provider PricingProvider) float64 {
	hoursPerMonth := 24.0 * 30.0 // 720 hours

	// Compute cost
	computeCost := (cpuCores * provider.CPUHourly + memoryGB * provider.MemoryHourly) * hoursPerMonth

	// Storage cost
	storageCost := storageGB * provider.StorageMonthly

	// Add 15% overhead for networking, monitoring, etc.
	totalCost := (computeCost + storageCost) * 1.15

	return totalCost
}

// EstimateSavings estimates potential savings from optimization
func EstimateSavings(current, optimized float64) (savings float64, percentage float64) {
	savings = current - optimized
	if current > 0 {
		percentage = (savings / current) * 100
	}
	return savings, percentage
}