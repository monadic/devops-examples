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
    "strings"
    "time"

    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/rest"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/dynamic"
)

type DriftDetector struct {
    k8sClient     *kubernetes.Clientset
    dynamicClient dynamic.Interface
    cubClient     *CubClient
    claudeClient  *ClaudeClient
    namespace     string
    space         string
}

// CubClient - Simple ConfigHub API client
type CubClient struct {
    baseURL string
    token   string
    client  *http.Client
}

// ClaudeClient - Simple Claude API client
type ClaudeClient struct {
    apiKey string
    client *http.Client
}

type Unit struct {
    Name   string                 `json:"name"`
    Space  string                 `json:"space"`
    Data   map[string]interface{} `json:"data"`
    Labels map[string]string      `json:"labels"`
}

type DriftAnalysis struct {
    HasDrift bool             `json:"has_drift"`
    Items    []DriftItem      `json:"items"`
    Summary  string           `json:"summary"`
    Fixes    []ProposedFix    `json:"fixes"`
}

type DriftItem struct {
    Resource string `json:"resource"`
    Field    string `json:"field"`
    Expected string `json:"expected"`
    Actual   string `json:"actual"`
}

type ProposedFix struct {
    Unit        string                 `json:"unit"`
    PatchPath   string                 `json:"patch_path"`
    PatchValue  interface{}            `json:"patch_value"`
    Explanation string                 `json:"explanation"`
}

func main() {
    detector, err := NewDriftDetector()
    if err != nil {
        log.Fatalf("Failed to initialize: %v", err)
    }

    log.Println("Drift detector started")
    log.Printf("Monitoring namespace: %s", detector.namespace)
    log.Printf("ConfigHub space: %s", detector.space)

    // Run detection loop
    for {
        log.Println("Checking for drift...")
        if err := detector.DetectAndReport(); err != nil {
            log.Printf("Detection error: %v", err)
        }

        time.Sleep(5 * time.Minute)
    }
}

func NewDriftDetector() (*DriftDetector, error) {
    // Initialize Kubernetes client
    config, err := getK8sConfig()
    if err != nil {
        return nil, fmt.Errorf("k8s config: %w", err)
    }

    k8sClient, err := kubernetes.NewForConfig(config)
    if err != nil {
        return nil, fmt.Errorf("k8s client: %w", err)
    }

    dynamicClient, err := dynamic.NewForConfig(config)
    if err != nil {
        return nil, fmt.Errorf("dynamic client: %w", err)
    }

    // Initialize CUB client
    cubClient := &CubClient{
        baseURL: getEnvOrDefault("CUB_API_URL", "https://hub.confighub.com/api/v1"),
        token:   os.Getenv("CUB_TOKEN"),
        client:  &http.Client{Timeout: 30 * time.Second},
    }

    // Initialize Claude client
    claudeClient := &ClaudeClient{
        apiKey: os.Getenv("CLAUDE_API_KEY"),
        client: &http.Client{Timeout: 60 * time.Second},
    }

    return &DriftDetector{
        k8sClient:     k8sClient,
        dynamicClient: dynamicClient,
        cubClient:     cubClient,
        claudeClient:  claudeClient,
        namespace:     getEnvOrDefault("NAMESPACE", "qa"),
        space:         getEnvOrDefault("CUB_SPACE", "acorn-bear-qa"),
    }, nil
}

func (d *DriftDetector) DetectAndReport() error {
    // 1. Get desired state from ConfigHub
    units, err := d.cubClient.GetUnits(d.space)
    if err != nil {
        return fmt.Errorf("get units: %w", err)
    }
    log.Printf("Found %d units in ConfigHub space %s", len(units), d.space)

    // 2. Get actual state from Kubernetes
    actualState, err := d.getActualState()
    if err != nil {
        return fmt.Errorf("get actual state: %w", err)
    }
    log.Printf("Found %d resources in Kubernetes namespace %s", len(actualState), d.namespace)

    // 3. Perform basic drift detection
    basicDrift := d.detectBasicDrift(units, actualState)
    if len(basicDrift) == 0 {
        log.Println("No drift detected")
        return nil
    }
    log.Printf("Detected %d drift items", len(basicDrift))

    // 4. Ask Claude for deeper analysis
    analysis, err := d.analyzeWithClaude(units, actualState, basicDrift)
    if err != nil {
        log.Printf("Claude analysis failed (will use basic drift): %v", err)
        // Continue with basic drift even if Claude fails
        analysis = &DriftAnalysis{
            HasDrift: true,
            Items:    basicDrift,
            Summary:  fmt.Sprintf("Found %d configuration differences", len(basicDrift)),
        }
    }

    // 5. Report findings
    d.reportDrift(analysis)

    // 6. Optionally create fix (controlled by env var)
    if os.Getenv("AUTO_FIX") == "true" && len(analysis.Fixes) > 0 {
        if err := d.createFix(analysis); err != nil {
            log.Printf("Failed to create fix: %v", err)
        }
    }

    return nil
}

func (d *DriftDetector) getActualState() (map[string]interface{}, error) {
    state := make(map[string]interface{})

    // Get Deployments
    deployments, err := d.k8sClient.AppsV1().Deployments(d.namespace).List(context.Background(), metav1.ListOptions{})
    if err != nil {
        return nil, err
    }
    for _, dep := range deployments.Items {
        state[fmt.Sprintf("deployment/%s", dep.Name)] = map[string]interface{}{
            "replicas": *dep.Spec.Replicas,
            "image":    dep.Spec.Template.Spec.Containers[0].Image,
            "resources": dep.Spec.Template.Spec.Containers[0].Resources,
        }
    }

    // Get Services
    services, err := d.k8sClient.CoreV1().Services(d.namespace).List(context.Background(), metav1.ListOptions{})
    if err != nil {
        return nil, err
    }
    for _, svc := range services.Items {
        ports := make([]interface{}, 0)
        for _, p := range svc.Spec.Ports {
            ports = append(ports, map[string]interface{}{
                "port":       p.Port,
                "targetPort": p.TargetPort.IntVal,
            })
        }
        state[fmt.Sprintf("service/%s", svc.Name)] = map[string]interface{}{
            "type":  svc.Spec.Type,
            "ports": ports,
        }
    }

    // Get ConfigMaps
    configMaps, err := d.k8sClient.CoreV1().ConfigMaps(d.namespace).List(context.Background(), metav1.ListOptions{})
    if err != nil {
        return nil, err
    }
    for _, cm := range configMaps.Items {
        state[fmt.Sprintf("configmap/%s", cm.Name)] = cm.Data
    }

    return state, nil
}

func (d *DriftDetector) detectBasicDrift(units []Unit, actualState map[string]interface{}) []DriftItem {
    var driftItems []DriftItem

    for _, unit := range units {
        // Simple comparison - check if key resources match
        if strings.Contains(unit.Name, "backend") {
            actualKey := "deployment/backend"
            if actual, ok := actualState[actualKey]; ok {
                // Check replicas
                if expectedReplicas := d.extractReplicas(unit.Data); expectedReplicas > 0 {
                    if actualMap, ok := actual.(map[string]interface{}); ok {
                        if actualReplicas, ok := actualMap["replicas"].(int32); ok {
                            if int32(expectedReplicas) != actualReplicas {
                                driftItems = append(driftItems, DriftItem{
                                    Resource: actualKey,
                                    Field:    "replicas",
                                    Expected: fmt.Sprintf("%d", expectedReplicas),
                                    Actual:   fmt.Sprintf("%d", actualReplicas),
                                })
                            }
                        }
                    }
                }
            }
        }
    }

    return driftItems
}

func (d *DriftDetector) extractReplicas(unitData map[string]interface{}) int {
    // Try to find replicas in the unit data structure
    if spec, ok := unitData["spec"].(map[string]interface{}); ok {
        if replicas, ok := spec["replicas"].(float64); ok {
            return int(replicas)
        }
    }
    return 1 // default
}

func (d *DriftDetector) analyzeWithClaude(units []Unit, actualState map[string]interface{}, basicDrift []DriftItem) (*DriftAnalysis, error) {
    prompt := fmt.Sprintf(`Analyze this Kubernetes configuration drift and suggest fixes.

Desired State (from ConfigHub):
%s

Actual State (from Kubernetes):
%s

Basic Drift Detected:
%s

Please analyze and return JSON with this structure:
{
  "has_drift": true/false,
  "items": [{"resource": "...", "field": "...", "expected": "...", "actual": "..."}],
  "summary": "Brief explanation of the drift",
  "fixes": [
    {
      "unit": "unit-name",
      "patch_path": "/spec/replicas",
      "patch_value": 3,
      "explanation": "Why this fix is needed"
    }
  ]
}`,
        d.jsonPretty(units),
        d.jsonPretty(actualState),
        d.jsonPretty(basicDrift))

    response, err := d.claudeClient.Complete(prompt)
    if err != nil {
        return nil, err
    }

    var analysis DriftAnalysis
    if err := json.Unmarshal([]byte(response), &analysis); err != nil {
        // If Claude doesn't return valid JSON, create basic analysis
        return &DriftAnalysis{
            HasDrift: len(basicDrift) > 0,
            Items:    basicDrift,
            Summary:  "Configuration drift detected",
        }, nil
    }

    return &analysis, nil
}

func (d *DriftDetector) reportDrift(analysis *DriftAnalysis) {
    log.Println("=== DRIFT REPORT ===")
    log.Printf("Summary: %s", analysis.Summary)
    log.Printf("Drift Items: %d", len(analysis.Items))

    for _, item := range analysis.Items {
        log.Printf("  - %s.%s: expected=%s, actual=%s",
            item.Resource, item.Field, item.Expected, item.Actual)
    }

    if len(analysis.Fixes) > 0 {
        log.Println("Proposed Fixes:")
        for _, fix := range analysis.Fixes {
            log.Printf("  - %s: %s", fix.Unit, fix.Explanation)
        }
    }
}

func (d *DriftDetector) createFix(analysis *DriftAnalysis) error {
    // Create a new ConfigHub space for the fix
    fixSpace := fmt.Sprintf("%s-drift-fix-%d", d.space, time.Now().Unix())

    log.Printf("Creating fix space: %s", fixSpace)

    // In real implementation, would call ConfigHub API to:
    // 1. Create new space
    // 2. Apply patches from analysis.Fixes
    // 3. Create PR or notification

    return nil
}

// CubClient methods
func (c *CubClient) GetUnits(space string) ([]Unit, error) {
    // For now, return mock data
    // Real implementation would call ConfigHub API
    return []Unit{
        {
            Name:  "backend",
            Space: space,
            Data: map[string]interface{}{
                "spec": map[string]interface{}{
                    "replicas": float64(2),
                },
            },
        },
        {
            Name:  "frontend",
            Space: space,
            Data: map[string]interface{}{
                "spec": map[string]interface{}{
                    "replicas": float64(3),
                },
            },
        },
    }, nil
}

// ClaudeClient methods
func (c *ClaudeClient) Complete(prompt string) (string, error) {
    // Real implementation would call Claude API
    // For now, return a mock response
    if c.apiKey == "" {
        // Return basic analysis without Claude
        return `{
            "has_drift": true,
            "summary": "Drift detected (Claude API not configured)",
            "items": [],
            "fixes": []
        }`, nil
    }

    // Actual Claude API call would go here
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

    jsonData, _ := json.Marshal(payload)
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

    body, _ := io.ReadAll(resp.Body)

    // Parse Claude response and extract content
    var claudeResp map[string]interface{}
    if err := json.Unmarshal(body, &claudeResp); err != nil {
        return "", err
    }

    // Extract the actual response text
    if content, ok := claudeResp["content"].([]interface{}); ok && len(content) > 0 {
        if text, ok := content[0].(map[string]interface{})["text"].(string); ok {
            return text, nil
        }
    }

    return "", fmt.Errorf("unexpected Claude response format")
}

// Helper functions
func getK8sConfig() (*rest.Config, error) {
    // Try in-cluster config first
    config, err := rest.InClusterConfig()
    if err != nil {
        // Fall back to kubeconfig
        kubeconfig := os.Getenv("KUBECONFIG")
        if kubeconfig == "" {
            kubeconfig = os.Getenv("HOME") + "/.kube/config"
        }
        config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
        if err != nil {
            // Last resort: use the Kind cluster config we created
            kubeconfig = "var/acorn-bear-infra.kubeconfig"
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

func (d *DriftDetector) jsonPretty(v interface{}) string {
    b, _ := json.MarshalIndent(v, "", "  ")
    return string(b)
}