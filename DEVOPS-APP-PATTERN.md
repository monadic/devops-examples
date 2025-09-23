# The DevOps App Pattern

## Both Global-App and Drift-Detector Follow the Same Pattern

### 1. They're Both Kubernetes Deployments

```yaml
# Global-App Backend (business app)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend
  namespace: qa
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: backend
        image: ghcr.io/confighubai/cubbychat/backend:1.1.1
        env:
        - name: DATABASE_URL
          value: postgres://...
        ports:
        - containerPort: 8080

---

# Drift-Detector (DevOps app)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: drift-detector
  namespace: devops-apps  # Different namespace, same pattern!
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: drift-detector
        image: drift-detector:latest
        env:
        - name: CLAUDE_API_KEY      # REQUIRED by default
          value: sk-ant-...
        - name: CLAUDE_DEBUG_LOGGING # Default: true
          value: "true"
        - name: CUB_TOKEN           # Auto-obtained from cub CLI
          value: eyJ...
```

### 2. Both Run Continuously

```go
// Global-App Backend main.go
func main() {
    server := NewServer()
    for {
        server.HandleRequests()  // Runs forever
    }
}

// Drift-Detector main.go (SAME PATTERN!)
func main() {
    detector := NewDriftDetector()
    for {
        detector.DetectAndReport()  // Runs forever
        time.Sleep(5 * time.Minute)
    }
}
```

### 3. Both Integrate with External Services

```go
// Global-App Backend
type Backend struct {
    db       *sql.DB           // Connects to Postgres
    ollama   *OllamaClient     // Connects to AI service
    frontend *FrontendClient   // Connects to frontend
}

// Drift-Detector (SAME PATTERN!)
type DriftDetector struct {
    k8sClient    *kubernetes.Clientset  // Connects to Kubernetes
    cubClient    *CubClient             // Connects to ConfigHub
    claudeClient *ClaudeClient          // Connects to Claude AI
}
```

### 4. Both Use ConfigHub for Configuration

```go
// Global-App: ConfigHub manages its deployment config
// ConfigHub Unit: backend.yaml → Deployed to K8s

// Drift-Detector: ALSO managed by ConfigHub!
// ConfigHub Unit: drift-detector.yaml → Deployed to K8s
// PLUS: It reads ConfigHub to detect drift
```

### 5. Both Can Be Promoted Through Environments

```bash
# Global-App promotion
cub unit update --space acorn-bear-qa        # Deploy to QA
cub unit update --space acorn-bear-staging   # Promote to staging
cub unit update --space acorn-bear-prod      # Promote to prod

# Drift-Detector promotion (SAME!)
cub unit update --space drift-detector-dev   # Deploy to dev
cub unit update --space drift-detector-prod  # Promote to prod
```

## The Key Insight

**DevOps automation is not special** - it's just another application that happens to do DevOps tasks.

| Aspect | Global-App Backend | Drift-Detector |
|--------|-------------------|----------------|
| **Purpose** | Serve business logic | Detect configuration drift |
| **Architecture** | Kubernetes Deployment | Kubernetes Deployment |
| **Runtime** | Continuous | Continuous |
| **State** | Stateful (database) | Stateful (tracks drift) |
| **AI Integration** | Calls Ollama | Calls Claude |
| **Deployment** | Via ConfigHub | Via ConfigHub |
| **Monitoring** | Prometheus metrics | Prometheus metrics |
| **Logging** | stdout → log aggregator | stdout → log aggregator |
| **Versioning** | Git + container tags | Git + container tags |
| **Rollback** | kubectl rollout undo | kubectl rollout undo |

## Where Claude Gets Called (Detailed)

```go
// main.go - The complete Claude call chain:

func main() {
    // 1. Main loop starts
    for {
        detector.DetectAndReport()  // Line 74
        time.Sleep(5 * time.Minute)
    }
}

func (d *DriftDetector) DetectAndReport() {
    // 2. Detection flow
    basicDrift := d.detectBasicDrift(units, actualState)  // Line 149

    // 3. CLAUDE CALLED HERE!
    analysis, err := d.analyzeWithClaude(units, actualState, basicDrift)  // Line 157
}

func (d *DriftDetector) analyzeWithClaude(...) *DriftAnalysis {
    // 4. Build prompt for Claude
    prompt := fmt.Sprintf(`Analyze this drift...`)  // Line 269

    // 5. Call Claude API
    response, err := d.claudeClient.Complete(prompt)  // Line 297

    return analysis
}

func (c *ClaudeClient) Complete(prompt string) (string, error) {
    // 6. ACTUAL HTTP CALL TO CLAUDE
    req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", ...)
    req.Header.Set("x-api-key", c.apiKey)  // Line 407

    resp, err := c.client.Do(req)  // Line 411 - THE ACTUAL API CALL!

    // 7. Parse Claude's response
    return extractedText  // Line 428
}
```

## Why This Matters

This demonstrates that **DevOps automation should be built as applications**, not scripts or workflows:

1. **Reliability**: Apps can be monitored, scaled, rolled back
2. **State**: Apps can learn and improve over time
3. **Integration**: Apps can deeply integrate with proprietary systems
4. **Lifecycle**: Apps have versions, deployments, SLAs
5. **AI-Native**: Apps can maintain context for better AI decisions

## Comparison with Workflow Approach

| Workflow (Cased) | App (Our Approach) |
|-----------------|-------------------|
| Triggered by event | Runs continuously |
| Stateless | Stateful |
| No versioning | Full version control |
| Can't rollback | Can rollback |
| No monitoring | Full observability |
| Claude called per trigger | Claude has context |

The drift-detector is a **DevOps application**, deployed and managed exactly like global-app, just with a different purpose.