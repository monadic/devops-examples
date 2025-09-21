package main

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "time"

    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/resource"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/clientcmd"
    metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
    metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

type CostOptimizer struct {
    k8sClient     *kubernetes.Clientset
    metricsClient *metricsclientset.Clientset
    cubClient     *CubClient
    claudeClient  *ClaudeClient
    namespace     string
    space         string
}

type ResourceUsage struct {
    Name      string
    Namespace string
    Type      string // Deployment, StatefulSet, DaemonSet
    Replicas  int32
    CPU       CPUUsage
    Memory    MemoryUsage
    Storage   StorageUsage
}

type CPUUsage struct {
    Requested string // e.g., "100m"
    Limit     string // e.g., "500m"
    Actual    string // e.g., "50m" (from metrics)
}

type MemoryUsage struct {
    Requested string // e.g., "128Mi"
    Limit     string // e.g., "512Mi"
    Actual    string // e.g., "100Mi" (from metrics)
}

type StorageUsage struct {
    Size   string // e.g., "10Gi"
    Actual string // e.g., "2Gi" (actual usage)
}

type CostAnalysis struct {
    TotalMonthlyCost    float64                  `json:"total_monthly_cost"`
    PotentialSavings    float64                  `json:"potential_savings"`
    Recommendations     []CostRecommendation     `json:"recommendations"`
    ResourceBreakdown   map[string]float64       `json:"resource_breakdown"`
}

type CostRecommendation struct {
    Resource     string                 `json:"resource"`
    Type         string                 `json:"type"` // "rightsize", "scale", "remove", "reserved"
    Current      map[string]interface{} `json:"current"`
    Recommended  map[string]interface{} `json:"recommended"`
    MonthlySavings float64              `json:"monthly_savings"`
    Risk         string                 `json:"risk"` // "low", "medium", "high"
    Explanation  string                 `json:"explanation"`
}

// CubClient for ConfigHub integration
type CubClient struct {
    baseURL string
    token   string
    client  *http.Client
}

// ClaudeClient for AI analysis
type ClaudeClient struct {
    apiKey string
    client *http.Client
}

func main() {
    optimizer, err := NewCostOptimizer()
    if err != nil {
        log.Fatalf("Failed to initialize: %v", err)
    }

    log.Println("Cost optimizer started")
    log.Printf("Monitoring namespace: %s", optimizer.namespace)
    log.Printf("ConfigHub space: %s", optimizer.space)

    // Run optimization loop
    for {
        log.Println("Analyzing costs...")

        if err := optimizer.AnalyzeAndOptimize(); err != nil {
            log.Printf("Analysis error: %v", err)
        }

        // Run every hour
        time.Sleep(1 * time.Hour)
    }
}

func NewCostOptimizer() (*CostOptimizer, error) {
    config, err := getK8sConfig()
    if err != nil {
        return nil, fmt.Errorf("k8s config: %w", err)
    }

    k8sClient, err := kubernetes.NewForConfig(config)
    if err != nil {
        return nil, fmt.Errorf("k8s client: %w", err)
    }

    metricsClient, err := metricsclientset.NewForConfig(config)
    if err != nil {
        log.Printf("Warning: metrics client not available: %v", err)
        // Continue without metrics
    }

    cubClient := &CubClient{
        baseURL: getEnvOrDefault("CUB_API_URL", "https://hub.confighub.com/api/v1"),
        token:   os.Getenv("CUB_TOKEN"),
        client:  &http.Client{Timeout: 30 * time.Second},
    }

    claudeClient := &ClaudeClient{
        apiKey: os.Getenv("CLAUDE_API_KEY"),
        client: &http.Client{Timeout: 60 * time.Second},
    }

    return &CostOptimizer{
        k8sClient:     k8sClient,
        metricsClient: metricsClient,
        cubClient:     cubClient,
        claudeClient:  claudeClient,
        namespace:     getEnvOrDefault("NAMESPACE", "default"),
        space:         getEnvOrDefault("CUB_SPACE", "acorn-bear-qa"),
    }, nil
}

func (o *CostOptimizer) AnalyzeAndOptimize() error {
    // 1. Collect resource usage data
    usage, err := o.collectResourceUsage()
    if err != nil {
        return fmt.Errorf("collect usage: %w", err)
    }
    log.Printf("Collected data for %d resources", len(usage))

    // 2. Get actual metrics if available
    if o.metricsClient != nil {
        o.enrichWithMetrics(usage)
    }

    // 3. Calculate current costs
    currentCosts := o.calculateCosts(usage)
    log.Printf("Current monthly cost: $%.2f", currentCosts)

    // 4. Use Claude to analyze and recommend optimizations
    analysis, err := o.analyzeWithClaude(usage, currentCosts)
    if err != nil {
        log.Printf("Claude analysis failed: %v", err)
        // Fall back to basic analysis
        analysis = o.basicAnalysis(usage, currentCosts)
    }

    // 5. Report findings
    o.reportAnalysis(analysis)

    // 6. Optionally create optimization in ConfigHub
    if os.Getenv("AUTO_OPTIMIZE") == "true" && len(analysis.Recommendations) > 0 {
        if err := o.createOptimization(analysis); err != nil {
            log.Printf("Failed to create optimization: %v", err)
        }
    }

    return nil
}

func (o *CostOptimizer) collectResourceUsage() ([]ResourceUsage, error) {
    var usage []ResourceUsage

    // Get Deployments
    deployments, err := o.k8sClient.AppsV1().Deployments(o.namespace).List(
        context.Background(),
        metav1.ListOptions{},
    )
    if err != nil {
        return nil, err
    }

    for _, dep := range deployments.Items {
        usage = append(usage, o.extractDeploymentUsage(&dep))
    }

    // Get StatefulSets
    statefulsets, err := o.k8sClient.AppsV1().StatefulSets(o.namespace).List(
        context.Background(),
        metav1.ListOptions{},
    )
    if err != nil {
        return nil, err
    }

    for _, sts := range statefulsets.Items {
        usage = append(usage, o.extractStatefulSetUsage(&sts))
    }

    // Get PVCs for storage costs
    pvcs, err := o.k8sClient.CoreV1().PersistentVolumeClaims(o.namespace).List(
        context.Background(),
        metav1.ListOptions{},
    )
    if err != nil {
        return nil, err
    }

    // Add storage to relevant resources
    for _, pvc := range pvcs.Items {
        o.enrichWithStorage(&usage, &pvc)
    }

    return usage, nil
}

func (o *CostOptimizer) extractDeploymentUsage(dep *appsv1.Deployment) ResourceUsage {
    usage := ResourceUsage{
        Name:      dep.Name,
        Namespace: dep.Namespace,
        Type:      "Deployment",
        Replicas:  *dep.Spec.Replicas,
    }

    // Get resource requests/limits from first container
    if len(dep.Spec.Template.Spec.Containers) > 0 {
        container := dep.Spec.Template.Spec.Containers[0]

        if container.Resources.Requests != nil {
            if cpu := container.Resources.Requests.Cpu(); cpu != nil {
                usage.CPU.Requested = cpu.String()
            }
            if mem := container.Resources.Requests.Memory(); mem != nil {
                usage.Memory.Requested = mem.String()
            }
        }

        if container.Resources.Limits != nil {
            if cpu := container.Resources.Limits.Cpu(); cpu != nil {
                usage.CPU.Limit = cpu.String()
            }
            if mem := container.Resources.Limits.Memory(); mem != nil {
                usage.Memory.Limit = mem.String()
            }
        }
    }

    return usage
}

func (o *CostOptimizer) extractStatefulSetUsage(sts *appsv1.StatefulSet) ResourceUsage {
    usage := ResourceUsage{
        Name:      sts.Name,
        Namespace: sts.Namespace,
        Type:      "StatefulSet",
        Replicas:  *sts.Spec.Replicas,
    }

    // Similar extraction as deployment
    if len(sts.Spec.Template.Spec.Containers) > 0 {
        container := sts.Spec.Template.Spec.Containers[0]

        if container.Resources.Requests != nil {
            if cpu := container.Resources.Requests.Cpu(); cpu != nil {
                usage.CPU.Requested = cpu.String()
            }
            if mem := container.Resources.Requests.Memory(); mem != nil {
                usage.Memory.Requested = mem.String()
            }
        }

        if container.Resources.Limits != nil {
            if cpu := container.Resources.Limits.Cpu(); cpu != nil {
                usage.CPU.Limit = cpu.String()
            }
            if mem := container.Resources.Limits.Memory(); mem != nil {
                usage.Memory.Limit = mem.String()
            }
        }
    }

    return usage
}

func (o *CostOptimizer) enrichWithMetrics(usage []ResourceUsage) {
    if o.metricsClient == nil {
        return
    }

    // Get pod metrics
    podMetrics, err := o.metricsClient.MetricsV1beta1().PodMetricses(o.namespace).List(
        context.Background(),
        metav1.ListOptions{},
    )
    if err != nil {
        log.Printf("Failed to get pod metrics: %v", err)
        return
    }

    // Map metrics to resources
    for i := range usage {
        o.addActualUsage(&usage[i], podMetrics)
    }
}

func (o *CostOptimizer) addActualUsage(usage *ResourceUsage, podMetrics *metricsv1beta1.PodMetricsList) {
    totalCPU := resource.NewQuantity(0, resource.DecimalSI)
    totalMemory := resource.NewQuantity(0, resource.BinarySI)
    podCount := 0

    for _, pm := range podMetrics.Items {
        // Check if pod belongs to this resource
        if o.podBelongsToResource(pm.Name, usage.Name) {
            for _, container := range pm.Containers {
                totalCPU.Add(*container.Usage.Cpu())
                totalMemory.Add(*container.Usage.Memory())
            }
            podCount++
        }
    }

    if podCount > 0 {
        // Average per pod
        avgCPU := totalCPU.DeepCopy()
        avgCPU.Set(avgCPU.Value() / int64(podCount))
        usage.CPU.Actual = avgCPU.String()

        avgMem := totalMemory.DeepCopy()
        avgMem.Set(avgMem.Value() / int64(podCount))
        usage.Memory.Actual = avgMem.String()
    }
}

func (o *CostOptimizer) podBelongsToResource(podName, resourceName string) bool {
    // Simple check - in reality would use labels/selectors
    return len(podName) > len(resourceName) && podName[:len(resourceName)] == resourceName
}

func (o *CostOptimizer) enrichWithStorage(usage *[]ResourceUsage, pvc *corev1.PersistentVolumeClaim) {
    // Match PVC to resource based on labels or naming convention
    for i := range *usage {
        if o.pvcBelongsToResource(pvc, &(*usage)[i]) {
            if storage := pvc.Spec.Resources.Requests.Storage(); storage != nil {
                (*usage)[i].Storage.Size = storage.String()
            }
        }
    }
}

func (o *CostOptimizer) pvcBelongsToResource(pvc *corev1.PersistentVolumeClaim, usage *ResourceUsage) bool {
    // Check if PVC name contains resource name (simplified)
    return usage.Type == "StatefulSet" && pvc.Name == usage.Name
}

func (o *CostOptimizer) calculateCosts(usage []ResourceUsage) float64 {
    totalCost := 0.0

    // Simple cost model (per month)
    // CPU: $25 per vCPU
    // Memory: $3 per GB
    // Storage: $0.10 per GB

    for _, u := range usage {
        // CPU cost
        if u.CPU.Requested != "" {
            cpuQuantity, _ := resource.ParseQuantity(u.CPU.Requested)
            cpuCores := float64(cpuQuantity.MilliValue()) / 1000.0
            totalCost += cpuCores * 25.0 * float64(u.Replicas)
        }

        // Memory cost
        if u.Memory.Requested != "" {
            memQuantity, _ := resource.ParseQuantity(u.Memory.Requested)
            memGB := float64(memQuantity.Value()) / (1024 * 1024 * 1024)
            totalCost += memGB * 3.0 * float64(u.Replicas)
        }

        // Storage cost
        if u.Storage.Size != "" {
            storageQuantity, _ := resource.ParseQuantity(u.Storage.Size)
            storageGB := float64(storageQuantity.Value()) / (1024 * 1024 * 1024)
            totalCost += storageGB * 0.10
        }
    }

    return totalCost
}

func (o *CostOptimizer) analyzeWithClaude(usage []ResourceUsage, currentCosts float64) (*CostAnalysis, error) {
    if o.claudeClient.apiKey == "" {
        return nil, fmt.Errorf("Claude API key not configured")
    }

    prompt := fmt.Sprintf(`Analyze these Kubernetes resources for cost optimization opportunities:

Current Monthly Cost: $%.2f

Resources:
%s

Please analyze and provide cost optimization recommendations. Consider:
1. Right-sizing based on actual vs requested resources
2. Replica count optimization
3. Idle or underutilized resources
4. Storage optimization
5. Reserved instance opportunities

Return JSON with this structure:
{
  "total_monthly_cost": %.2f,
  "potential_savings": 0.0,
  "recommendations": [
    {
      "resource": "resource-name",
      "type": "rightsize|scale|remove|reserved",
      "current": {"replicas": 3, "cpu": "500m", "memory": "1Gi"},
      "recommended": {"replicas": 2, "cpu": "200m", "memory": "512Mi"},
      "monthly_savings": 50.0,
      "risk": "low|medium|high",
      "explanation": "why this optimization makes sense"
    }
  ],
  "resource_breakdown": {
    "compute": 0.0,
    "memory": 0.0,
    "storage": 0.0
  }
}`, currentCosts, o.formatUsageForClaude(usage), currentCosts)

    response, err := o.claudeClient.Complete(prompt)
    if err != nil {
        return nil, err
    }

    var analysis CostAnalysis
    if err := json.Unmarshal([]byte(response), &analysis); err != nil {
        return nil, fmt.Errorf("parse Claude response: %w", err)
    }

    return &analysis, nil
}

func (o *CostOptimizer) formatUsageForClaude(usage []ResourceUsage) string {
    b, _ := json.MarshalIndent(usage, "", "  ")
    return string(b)
}

func (o *CostOptimizer) basicAnalysis(usage []ResourceUsage, currentCosts float64) *CostAnalysis {
    analysis := &CostAnalysis{
        TotalMonthlyCost:  currentCosts,
        PotentialSavings:  0,
        Recommendations:   []CostRecommendation{},
        ResourceBreakdown: make(map[string]float64),
    }

    // Simple heuristics for optimization
    for _, u := range usage {
        // Check for oversized resources
        if u.CPU.Actual != "" && u.CPU.Requested != "" {
            actualCPU, _ := resource.ParseQuantity(u.CPU.Actual)
            requestedCPU, _ := resource.ParseQuantity(u.CPU.Requested)

            // If using less than 50% of requested
            if actualCPU.MilliValue() < requestedCPU.MilliValue()/2 {
                savings := o.calculateSavingsForRightsize(u, 0.5)
                analysis.Recommendations = append(analysis.Recommendations, CostRecommendation{
                    Resource: u.Name,
                    Type:     "rightsize",
                    Current: map[string]interface{}{
                        "cpu":    u.CPU.Requested,
                        "memory": u.Memory.Requested,
                    },
                    Recommended: map[string]interface{}{
                        "cpu":    u.CPU.Actual,
                        "memory": u.Memory.Actual,
                    },
                    MonthlySavings: savings,
                    Risk:           "low",
                    Explanation:    "Resource is using less than 50% of requested capacity",
                })
                analysis.PotentialSavings += savings
            }
        }
    }

    return analysis
}

func (o *CostOptimizer) calculateSavingsForRightsize(usage ResourceUsage, factor float64) float64 {
    savings := 0.0

    if usage.CPU.Requested != "" {
        cpuQuantity, _ := resource.ParseQuantity(usage.CPU.Requested)
        cpuCores := float64(cpuQuantity.MilliValue()) / 1000.0
        savings += cpuCores * 25.0 * float64(usage.Replicas) * (1 - factor)
    }

    if usage.Memory.Requested != "" {
        memQuantity, _ := resource.ParseQuantity(usage.Memory.Requested)
        memGB := float64(memQuantity.Value()) / (1024 * 1024 * 1024)
        savings += memGB * 3.0 * float64(usage.Replicas) * (1 - factor)
    }

    return savings
}

func (o *CostOptimizer) reportAnalysis(analysis *CostAnalysis) {
    log.Println("=== COST OPTIMIZATION REPORT ===")
    log.Printf("Current Monthly Cost: $%.2f", analysis.TotalMonthlyCost)
    log.Printf("Potential Savings: $%.2f (%.1f%%)",
        analysis.PotentialSavings,
        (analysis.PotentialSavings/analysis.TotalMonthlyCost)*100)

    log.Printf("Found %d optimization opportunities:", len(analysis.Recommendations))

    for _, rec := range analysis.Recommendations {
        log.Printf("  %s (%s):", rec.Resource, rec.Type)
        log.Printf("    Savings: $%.2f/month", rec.MonthlySavings)
        log.Printf("    Risk: %s", rec.Risk)
        log.Printf("    Action: %s", rec.Explanation)
    }

    if len(analysis.ResourceBreakdown) > 0 {
        log.Println("Cost Breakdown:")
        for category, cost := range analysis.ResourceBreakdown {
            log.Printf("  %s: $%.2f", category, cost)
        }
    }
}

func (o *CostOptimizer) createOptimization(analysis *CostAnalysis) error {
    // Create a new ConfigHub space for optimizations
    optimizationSpace := fmt.Sprintf("%s-cost-opt-%d", o.space, time.Now().Unix())

    log.Printf("Creating optimization space: %s", optimizationSpace)

    // In real implementation, would:
    // 1. Create ConfigHub space
    // 2. Apply recommended changes
    // 3. Create gradual rollout plan

    return nil
}

// ClaudeClient implementation
func (c *ClaudeClient) Complete(prompt string) (string, error) {
    if c.apiKey == "" {
        return "", fmt.Errorf("Claude API key not set")
    }

    payload := map[string]interface{}{
        "model":       "claude-3-opus-20240229",
        "max_tokens":  2000,
        "temperature": 0,
        "messages": []map[string]string{
            {
                "role":    "user",
                "content": prompt,
            },
        },
    }

    jsonData, err := json.Marshal(payload)
    if err != nil {
        return "", err
    }

    req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
    if err != nil {
        return "", err
    }

    req.Header.Set("x-api-key", c.apiKey)
    req.Header.Set("anthropic-version", "2023-06-01")
    req.Header.Set("content-type", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    var claudeResp map[string]interface{}
    if err := json.Unmarshal(body, &claudeResp); err != nil {
        return "", err
    }

    if content, ok := claudeResp["content"].([]interface{}); ok && len(content) > 0 {
        if text, ok := content[0].(map[string]interface{})["text"].(string); ok {
            return text, nil
        }
    }

    return "", fmt.Errorf("unexpected Claude response format")
}

// Helper functions
func getK8sConfig() (*rest.Config, error) {
    config, err := rest.InClusterConfig()
    if err != nil {
        kubeconfig := os.Getenv("KUBECONFIG")
        if kubeconfig == "" {
            kubeconfig = os.Getenv("HOME") + "/.kube/config"
        }
        config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
        if err != nil {
            // Try Kind cluster config
            kubeconfig = "../../global-app/var/acorn-bear-infra.kubeconfig"
            config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
        }
    }
    return config, err
}

func getEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}