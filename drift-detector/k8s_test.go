package main

import (
	"context"
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// TestKindClusterDrift tests drift detection against local Kind cluster
func TestKindClusterDrift(t *testing.T) {
	// Get kubeconfig for Kind cluster
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{CurrentContext: "kind-devops-test"},
	)

	config, err := kubeconfig.ClientConfig()
	if err != nil {
		t.Skipf("Skipping Kind cluster test - cluster not available: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Check deployments in drift-test namespace
	deployments, err := clientset.AppsV1().Deployments("drift-test").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Skipf("Skipping - namespace not found: %v", err)
	}

	t.Logf("Found %d deployments in drift-test namespace", len(deployments.Items))

	// Check for drift
	driftFound := false
	for _, deployment := range deployments.Items {
		replicas := *deployment.Spec.Replicas
		expected := getExpectedReplicas(deployment.Name)

		if replicas != expected {
			driftFound = true
			t.Logf("⚠️  DRIFT DETECTED: %s has %d replicas, expected %d",
				deployment.Name, replicas, expected)
		} else {
			t.Logf("✅ No drift: %s has correct replica count (%d)",
				deployment.Name, replicas)
		}
	}

	if driftFound {
		t.Logf("\n🔍 Drift Summary:")
		t.Logf("   - backend-api: over-scaled (cost impact)")
		t.Logf("   - frontend-web: under-scaled (availability risk)")
		t.Logf("\n🔧 Recommended Actions:")
		t.Logf("   - Scale backend-api to 3 replicas")
		t.Logf("   - Scale frontend-web to 2 replicas")

		// In real implementation, would call ConfigHub to fix this
		t.Logf("\n📡 Would use ConfigHub APIs:")
		t.Logf("   - BulkPatchUnits with Upgrade=true")
		t.Logf("   - BulkApplyUnits to fix drift")
	}
}

func getExpectedReplicas(deploymentName string) int32 {
	// Expected replica counts
	expected := map[string]int32{
		"backend-api":  3,
		"frontend-web": 2,
	}

	if val, ok := expected[deploymentName]; ok {
		return val
	}
	return 1
}

// TestDriftDetectorWithKind demonstrates full workflow with Kind cluster
func TestDriftDetectorWithKind(t *testing.T) {
	// This test shows what would happen with real ConfigHub integration

	t.Log("🚀 Drift Detector - Kind Cluster Integration Test")
	t.Log("=" + fmt.Sprintf("%*s", 45, ""))

	t.Log("\n📋 Step 1: ConfigHub Setup (simulated)")
	t.Log("   Would create:")
	t.Log("   - Space: drift-detector")
	t.Log("   - Set: critical-services")
	t.Log("   - Filter: Labels['tier'] = 'critical'")

	t.Log("\n🔍 Step 2: Kubernetes State Check")

	// Connect to Kind cluster
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{CurrentContext: "kind-devops-test"},
	)

	config, err := kubeconfig.ClientConfig()
	if err != nil {
		t.Skipf("Kind cluster not available: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Get actual state
	deployments, err := clientset.AppsV1().Deployments("drift-test").List(
		context.Background(),
		metav1.ListOptions{
			LabelSelector: "tier=critical",
		},
	)
	if err != nil {
		t.Skipf("Test namespace not found: %v", err)
	}

	t.Logf("   Found %d critical deployments", len(deployments.Items))

	// Analyze drift
	var driftItems []string
	for _, dep := range deployments.Items {
		actual := *dep.Spec.Replicas
		expected := getExpectedReplicas(dep.Name)

		if actual != expected {
			driftItems = append(driftItems, fmt.Sprintf(
				"   - %s: %d replicas (expected %d)",
				dep.Name, actual, expected,
			))
		}
	}

	if len(driftItems) > 0 {
		t.Log("\n⚠️  Step 3: Drift Detected!")
		for _, item := range driftItems {
			t.Log(item)
		}

		t.Log("\n🤖 Step 4: AI Analysis (simulated)")
		t.Log("   Claude would analyze:")
		t.Log("   - Performance impact of over/under-scaling")
		t.Log("   - Cost implications")
		t.Log("   - Availability risks")

		t.Log("\n🔧 Step 5: Fix Application (simulated)")
		t.Log("   Would execute:")
		t.Log("   - BulkPatchUnits(Where: \"SetID='critical-services'\", Upgrade: true)")
		t.Log("   - Push changes downstream to all environments")
		t.Log("   - Apply fixes to Kubernetes cluster")

		t.Log("\n✅ Result: Drift would be corrected automatically!")
	} else {
		t.Log("\n✅ No drift detected - all deployments match expected state")
	}
}

// BenchmarkDriftDetection measures performance
func BenchmarkDriftDetection(b *testing.B) {
	// Connect to cluster once
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{CurrentContext: "kind-devops-test"},
	)

	config, err := kubeconfig.ClientConfig()
	if err != nil {
		b.Skipf("Kind cluster not available: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Measure drift detection performance
		deployments, _ := clientset.AppsV1().Deployments("drift-test").List(
			context.Background(),
			metav1.ListOptions{LabelSelector: "tier=critical"},
		)

		for _, dep := range deployments.Items {
			actual := *dep.Spec.Replicas
			expected := getExpectedReplicas(dep.Name)
			_ = (actual != expected)
		}
	}
}
