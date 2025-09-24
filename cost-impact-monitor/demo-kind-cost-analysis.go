package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type ResourceCost struct {
	Name         string
	Type         string
	Namespace    string
	CPURequested int64    // millicores
	MemRequested int64    // bytes
	Replicas     int32
	MonthlyCost  float64
	Status       string
}

func main() {
	fmt.Println("🔍 Cost Impact Analysis for Kind Cluster")
	fmt.Println("==========================================")
	fmt.Println()

	// Connect to Kind cluster
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.TODO()
	var allResources []ResourceCost
	totalCost := 0.0

	// Analyze drift-test namespace
	namespace := "drift-test"
	fmt.Printf("📦 Analyzing namespace: %s\n\n", namespace)

	// Get all deployments
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Error listing deployments: %v", err)
	} else {
		for _, dep := range deployments.Items {
			resource := analyzeDeployment(dep)
			allResources = append(allResources, resource)
			totalCost += resource.MonthlyCost
		}
	}

	// Get all statefulsets
	statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err == nil && len(statefulsets.Items) > 0 {
		for _, sts := range statefulsets.Items {
			fmt.Printf("StatefulSet: %s (%d replicas)\n", sts.Name, *sts.Spec.Replicas)
		}
	}

	// Display cost analysis
	fmt.Println("💰 Cost Analysis Results:")
	fmt.Println("-------------------------")
	fmt.Printf("%-25s %-15s %-10s %-15s %-15s %s\n",
		"Resource", "Type", "Replicas", "CPU (cores)", "Memory (GB)", "Monthly Cost")
	fmt.Println("--------------------------------------------------------------------------------")

	for _, r := range allResources {
		cpuCores := float64(r.CPURequested) / 1000.0
		memGB := float64(r.MemRequested) / (1024 * 1024 * 1024)
		fmt.Printf("%-25s %-15s %-10d %-15.2f %-15.2f $%.2f\n",
			r.Name, r.Type, r.Replicas, cpuCores, memGB, r.MonthlyCost)
	}

	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("TOTAL MONTHLY COST: $%.2f\n\n", totalCost)

	// Show drift impact
	fmt.Println("⚠️  Drift Impact Analysis:")
	fmt.Println("--------------------------")

	// Check for drift (compare actual vs expected)
	for _, r := range allResources {
		switch r.Name {
		case "test-app":
			if r.Replicas != 2 {
				costDiff := calculateCostDifference(r, 2)
				fmt.Printf("❌ %s: Running %d replicas (expected: 2)\n", r.Name, r.Replicas)
				fmt.Printf("   Cost impact: %+.2f/month\n", costDiff)
				fmt.Printf("   Fix: cub unit update deployment-test-app --patch --data '{\"spec\":{\"replicas\":2}}'\n")
			}
		case "complex-app":
			if r.Replicas != 3 {
				costDiff := calculateCostDifference(r, 3)
				fmt.Printf("❌ %s: Running %d replicas (expected: 3)\n", r.Name, r.Replicas)
				fmt.Printf("   Cost impact: %+.2f/month\n", costDiff)
				fmt.Printf("   Fix: cub unit update deployment-complex-app --patch --data '{\"spec\":{\"replicas\":3}}'\n")
			}
		}
	}

	// Check ConfigMaps for drift
	configmaps, _ := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	for _, cm := range configmaps.Items {
		if cm.Name == "app-config" {
			if logLevel, ok := cm.Data["log_level"]; ok && logLevel != "info" {
				fmt.Printf("❌ ConfigMap app-config: log_level is '%s' (expected: 'info')\n", logLevel)
				fmt.Printf("   Fix: cub unit update configmap-app-config --patch --data '{\"data\":{\"log_level\":\"info\"}}'\n")
			}
		}
	}

	fmt.Println()
	fmt.Println("📊 Cost Optimization Recommendations:")
	fmt.Println("-------------------------------------")

	// Provide recommendations
	for _, r := range allResources {
		if r.Replicas > 3 {
			savings := (float64(r.Replicas - 3) / float64(r.Replicas)) * r.MonthlyCost
			fmt.Printf("💡 %s: Consider reducing from %d to 3 replicas (save $%.2f/month)\n",
				r.Name, r.Replicas, savings)
		}

		cpuCores := float64(r.CPURequested) / 1000.0
		if cpuCores < 0.2 && r.Replicas > 1 {
			fmt.Printf("💡 %s: Low CPU request (%.2f cores), consider consolidating\n",
				r.Name, cpuCores)
		}
	}

	fmt.Println()
	fmt.Println("🎯 ConfigHub Integration:")
	fmt.Println("-------------------------")
	fmt.Println("To maintain these resources via ConfigHub:")
	fmt.Println("1. Create units: cub unit create deployment-[name] --data @deployment.yaml")
	fmt.Println("2. Apply units: cub unit apply deployment-[name]")
	fmt.Println("3. Monitor drift: Use drift-detector to compare ConfigHub vs Kubernetes")
	fmt.Println("4. Fix drift: Update ConfigHub units (not kubectl!)")
}

func analyzeDeployment(dep appsv1.Deployment) ResourceCost {
	resource := ResourceCost{
		Name:      dep.Name,
		Type:      "Deployment",
		Namespace: dep.Namespace,
		Replicas:  *dep.Spec.Replicas,
	}

	// Calculate resource requests
	if len(dep.Spec.Template.Spec.Containers) > 0 {
		for _, container := range dep.Spec.Template.Spec.Containers {
			if cpu := container.Resources.Requests.Cpu(); cpu != nil {
				resource.CPURequested += cpu.MilliValue() * int64(resource.Replicas)
			}
			if mem := container.Resources.Requests.Memory(); mem != nil {
				resource.MemRequested += mem.Value() * int64(resource.Replicas)
			}
		}
	}

	// Calculate monthly cost (AWS pricing estimates)
	cpuCores := float64(resource.CPURequested) / 1000.0
	memGB := float64(resource.MemRequested) / (1024 * 1024 * 1024)

	// AWS m5.large equivalent pricing
	cpuCostPerCore := 0.024 * 24 * 30  // $0.024 per vCPU-hour
	memCostPerGB := 0.006 * 24 * 30     // $0.006 per GB-hour

	resource.MonthlyCost = (cpuCores * cpuCostPerCore) + (memGB * memCostPerGB)

	// Add minimum cost for pod overhead
	resource.MonthlyCost += float64(resource.Replicas) * 2.0 // $2 per pod/month overhead

	return resource
}

func calculateCostDifference(current ResourceCost, expectedReplicas int32) float64 {
	costPerReplica := current.MonthlyCost / float64(current.Replicas)
	expectedCost := costPerReplica * float64(expectedReplicas)
	return current.MonthlyCost - expectedCost
}