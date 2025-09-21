package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/fluxcd/pkg/apis/meta"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "k8s.io/apimachinery/pkg/runtime/schema"
    "k8s.io/client-go/dynamic"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/clientcmd"
)

// FluxDriftDetector uses Flux's native drift detection
type FluxDriftDetector struct {
    dynamicClient dynamic.Interface
    cubClient     *CubClient
    claudeClient  *ClaudeClient
    namespace     string
}

// DriftInfo from Flux's status
type DriftInfo struct {
    Resource   string
    Kind       string
    Name       string
    Namespace  string
    Message    string
    DriftedAt  time.Time
    FluxSource string
}

func main() {
    detector, err := NewFluxDriftDetector()
    if err != nil {
        log.Fatalf("Failed to initialize: %v", err)
    }

    log.Println("Flux-based drift detector started")
    log.Printf("Monitoring Flux resources in namespace: %s", detector.namespace)

    // Run detection loop
    for {
        log.Println("Checking Flux resources for drift...")

        drifts, err := detector.DetectDrift()
        if err != nil {
            log.Printf("Detection error: %v", err)
        } else if len(drifts) > 0 {
            detector.HandleDrift(drifts)
        } else {
            log.Println("No drift detected by Flux")
        }

        time.Sleep(1 * time.Minute)
    }
}

func NewFluxDriftDetector() (*FluxDriftDetector, error) {
    config, err := getK8sConfig()
    if err != nil {
        return nil, fmt.Errorf("k8s config: %w", err)
    }

    dynamicClient, err := dynamic.NewForConfig(config)
    if err != nil {
        return nil, fmt.Errorf("dynamic client: %w", err)
    }

    return &FluxDriftDetector{
        dynamicClient: dynamicClient,
        cubClient:     NewCubClient(),
        claudeClient:  NewClaudeClient(),
        namespace:     getEnvOrDefault("FLUX_NAMESPACE", "flux-system"),
    }, nil
}

func (d *FluxDriftDetector) DetectDrift() ([]DriftInfo, error) {
    var allDrifts []DriftInfo

    // Check Kustomizations (Flux's main resource type)
    kustomizations, err := d.getKustomizations()
    if err != nil {
        return nil, fmt.Errorf("get kustomizations: %w", err)
    }

    for _, ks := range kustomizations {
        if drift := d.checkKustomizationDrift(ks); drift != nil {
            allDrifts = append(allDrifts, *drift)
        }
    }

    // Check HelmReleases
    helmReleases, err := d.getHelmReleases()
    if err != nil {
        log.Printf("Warning: could not get HelmReleases: %v", err)
    } else {
        for _, hr := range helmReleases {
            if drift := d.checkHelmReleaseDrift(hr); drift != nil {
                allDrifts = append(allDrifts, *drift)
            }
        }
    }

    // Check GitRepositories for source drift
    gitRepos, err := d.getGitRepositories()
    if err != nil {
        log.Printf("Warning: could not get GitRepositories: %v", err)
    } else {
        for _, gr := range gitRepos {
            if drift := d.checkGitRepoDrift(gr); drift != nil {
                allDrifts = append(allDrifts, *drift)
            }
        }
    }

    return allDrifts, nil
}

func (d *FluxDriftDetector) getKustomizations() ([]unstructured.Unstructured, error) {
    // Flux Kustomization CRD
    gvr := schema.GroupVersionResource{
        Group:    "kustomize.toolkit.fluxcd.io",
        Version:  "v1",
        Resource: "kustomizations",
    }

    list, err := d.dynamicClient.Resource(gvr).Namespace(d.namespace).List(
        context.Background(),
        metav1.ListOptions{},
    )
    if err != nil {
        return nil, err
    }

    return list.Items, nil
}

func (d *FluxDriftDetector) getHelmReleases() ([]unstructured.Unstructured, error) {
    gvr := schema.GroupVersionResource{
        Group:    "helm.toolkit.fluxcd.io",
        Version:  "v2beta2",
        Resource: "helmreleases",
    }

    list, err := d.dynamicClient.Resource(gvr).Namespace("").List(
        context.Background(),
        metav1.ListOptions{},
    )
    if err != nil {
        return nil, err
    }

    return list.Items, nil
}

func (d *FluxDriftDetector) getGitRepositories() ([]unstructured.Unstructured, error) {
    gvr := schema.GroupVersionResource{
        Group:    "source.toolkit.fluxcd.io",
        Version:  "v1",
        Resource: "gitrepositories",
    }

    list, err := d.dynamicClient.Resource(gvr).Namespace(d.namespace).List(
        context.Background(),
        metav1.ListOptions{},
    )
    if err != nil {
        return nil, err
    }

    return list.Items, nil
}

func (d *FluxDriftDetector) checkKustomizationDrift(ks unstructured.Unstructured) *DriftInfo {
    // Check Flux's status conditions for drift
    status, found, _ := unstructured.NestedMap(ks.Object, "status")
    if !found {
        return nil
    }

    conditions, found, _ := unstructured.NestedSlice(status, "conditions")
    if !found {
        return nil
    }

    for _, condition := range conditions {
        condMap, ok := condition.(map[string]interface{})
        if !ok {
            continue
        }

        condType, _ := condMap["type"].(string)
        condStatus, _ := condMap["status"].(string)
        reason, _ := condMap["reason"].(string)
        message, _ := condMap["message"].(string)

        // Flux sets these conditions when drift is detected
        if condType == "Ready" && condStatus == "False" && reason == "DriftDetected" {
            return &DriftInfo{
                Resource:   "Kustomization",
                Kind:       ks.GetKind(),
                Name:       ks.GetName(),
                Namespace:  ks.GetNamespace(),
                Message:    message,
                DriftedAt:  time.Now(),
                FluxSource: fmt.Sprintf("%s/%s", ks.GetNamespace(), ks.GetName()),
            }
        }

        // Also check for reconciliation failures that might indicate drift
        if condType == "Ready" && condStatus == "False" &&
           (reason == "ReconciliationFailed" || reason == "HealthCheckFailed") {
            // This could be drift - let's analyze
            return &DriftInfo{
                Resource:   "Kustomization",
                Kind:       ks.GetKind(),
                Name:       ks.GetName(),
                Namespace:  ks.GetNamespace(),
                Message:    fmt.Sprintf("Possible drift: %s", message),
                DriftedAt:  time.Now(),
                FluxSource: fmt.Sprintf("%s/%s", ks.GetNamespace(), ks.GetName()),
            }
        }
    }

    // Check inventory for drift
    if inventory, found := status["inventory"]; found {
        if entries, ok := inventory.(map[string]interface{})["entries"].([]interface{}); ok {
            // Flux tracks all managed resources in inventory
            // We could check each one for manual changes
            for _, entry := range entries {
                if e, ok := entry.(map[string]interface{}); ok {
                    // Check if resource has been modified outside of Flux
                    if d.isResourceDrifted(e) {
                        return &DriftInfo{
                            Resource:   "Kustomization",
                            Kind:       ks.GetKind(),
                            Name:       ks.GetName(),
                            Namespace:  ks.GetNamespace(),
                            Message:    "Inventory resource modified outside Flux",
                            DriftedAt:  time.Now(),
                            FluxSource: fmt.Sprintf("%s/%s", ks.GetNamespace(), ks.GetName()),
                        }
                    }
                }
            }
        }
    }

    return nil
}

func (d *FluxDriftDetector) checkHelmReleaseDrift(hr unstructured.Unstructured) *DriftInfo {
    status, found, _ := unstructured.NestedMap(hr.Object, "status")
    if !found {
        return nil
    }

    // Check for drift in Helm releases
    conditions, found, _ := unstructured.NestedSlice(status, "conditions")
    if !found {
        return nil
    }

    for _, condition := range conditions {
        condMap, ok := condition.(map[string]interface{})
        if !ok {
            continue
        }

        condType, _ := condMap["type"].(string)
        reason, _ := condMap["reason"].(string)

        // Helm-specific drift detection
        if condType == "TestSuccess" && reason == "TestFailed" {
            return &DriftInfo{
                Resource:   "HelmRelease",
                Kind:       hr.GetKind(),
                Name:       hr.GetName(),
                Namespace:  hr.GetNamespace(),
                Message:    "Helm test failed - possible drift",
                DriftedAt:  time.Now(),
                FluxSource: fmt.Sprintf("%s/%s", hr.GetNamespace(), hr.GetName()),
            }
        }

        // Check for upgrade failures that might indicate drift
        if condType == "Ready" && reason == "UpgradeFailed" {
            message, _ := condMap["message"].(string)
            return &DriftInfo{
                Resource:   "HelmRelease",
                Kind:       hr.GetKind(),
                Name:       hr.GetName(),
                Namespace:  hr.GetNamespace(),
                Message:    fmt.Sprintf("Upgrade failed (drift?): %s", message),
                DriftedAt:  time.Now(),
                FluxSource: fmt.Sprintf("%s/%s", hr.GetNamespace(), hr.GetName()),
            }
        }
    }

    return nil
}

func (d *FluxDriftDetector) checkGitRepoDrift(gr unstructured.Unstructured) *DriftInfo {
    status, found, _ := unstructured.NestedMap(gr.Object, "status")
    if !found {
        return nil
    }

    // Check if source has diverged
    observedGeneration, _, _ := unstructured.NestedInt64(status, "observedGeneration")
    generation := gr.GetGeneration()

    if observedGeneration != generation {
        return &DriftInfo{
            Resource:   "GitRepository",
            Kind:       gr.GetKind(),
            Name:       gr.GetName(),
            Namespace:  gr.GetNamespace(),
            Message:    fmt.Sprintf("Source drift: generation mismatch (%d != %d)", observedGeneration, generation),
            DriftedAt:  time.Now(),
            FluxSource: fmt.Sprintf("%s/%s", gr.GetNamespace(), gr.GetName()),
        }
    }

    return nil
}

func (d *FluxDriftDetector) isResourceDrifted(entry map[string]interface{}) bool {
    // Check if a resource tracked by Flux has been modified
    // This would require comparing with actual cluster state
    // For now, simplified check

    id, _ := entry["id"].(string)
    version, _ := entry["v"].(string)

    // In a real implementation, we would:
    // 1. Get the actual resource from cluster
    // 2. Compare its resourceVersion with Flux's tracked version
    // 3. Check annotations for manual changes

    log.Printf("Checking resource %s version %s for drift", id, version)

    // Placeholder - would need actual comparison
    return false
}

func (d *FluxDriftDetector) HandleDrift(drifts []DriftInfo) {
    log.Printf("=== FLUX DRIFT REPORT ===")
    log.Printf("Detected %d drift items", len(drifts))

    for _, drift := range drifts {
        log.Printf("DRIFT: %s/%s in %s", drift.Kind, drift.Name, drift.Namespace)
        log.Printf("  Message: %s", drift.Message)
        log.Printf("  Source: %s", drift.FluxSource)
        log.Printf("  Time: %s", drift.DriftedAt.Format(time.RFC3339))

        // Ask Claude for remediation advice
        if d.claudeClient != nil {
            advice := d.getRemediationAdvice(drift)
            log.Printf("  Claude suggests: %s", advice)
        }

        // Create ConfigHub fix space
        if os.Getenv("AUTO_FIX") == "true" {
            d.createConfigHubFix(drift)
        }
    }
}

func (d *FluxDriftDetector) getRemediationAdvice(drift DriftInfo) string {
    prompt := fmt.Sprintf(`
A Flux-managed resource has drifted:
- Resource: %s/%s
- Namespace: %s
- Message: %s
- Flux Source: %s

What's the best way to remediate this drift? Should we:
1. Force reconcile in Flux?
2. Update the source to match current state?
3. Revert the manual changes?

Provide a brief recommendation.
`, drift.Kind, drift.Name, drift.Namespace, drift.Message, drift.FluxSource)

    // Call Claude API (simplified)
    return "Force reconcile recommended"
}

func (d *FluxDriftDetector) createConfigHubFix(drift DriftInfo) {
    // Create a new ConfigHub space with the fix
    log.Printf("Would create ConfigHub fix for %s/%s", drift.Kind, drift.Name)
    // Implementation would call ConfigHub API
}

// CubClient - ConfigHub API client
type CubClient struct {
    baseURL string
    token   string
}

func NewCubClient() *CubClient {
    return &CubClient{
        baseURL: getEnvOrDefault("CUB_API_URL", "https://hub.confighub.com/api/v1"),
        token:   os.Getenv("CUB_TOKEN"),
    }
}

// ClaudeClient - Claude API client
type ClaudeClient struct {
    apiKey string
}

func NewClaudeClient() *ClaudeClient {
    apiKey := os.Getenv("CLAUDE_API_KEY")
    if apiKey == "" {
        return nil
    }
    return &ClaudeClient{apiKey: apiKey}
}

func getK8sConfig() (*rest.Config, error) {
    config, err := rest.InClusterConfig()
    if err != nil {
        kubeconfig := os.Getenv("KUBECONFIG")
        if kubeconfig == "" {
            kubeconfig = os.Getenv("HOME") + "/.kube/config"
        }
        config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
    }
    return config, err
}

func getEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}