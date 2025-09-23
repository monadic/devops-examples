package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sync"
)

// Dashboard provides a web interface for cost optimization results
type Dashboard struct {
	optimizer   *CostOptimizer
	latestAnalysis *CostAnalysis
	mutex       sync.RWMutex
	port        int
}

// NewDashboard creates a new dashboard instance
func NewDashboard(optimizer *CostOptimizer) *Dashboard {
	return &Dashboard{
		optimizer: optimizer,
		port:      8081, // Different from health port
	}
}

// Start starts the dashboard web server
func (d *Dashboard) Start() {
	d.optimizer.app.Logger.Printf("🌐 Starting cost optimization dashboard on port %d", d.port)

	http.HandleFunc("/", d.handleDashboard)
	http.HandleFunc("/api/analysis", d.handleAPIAnalysis)
	http.HandleFunc("/api/recommendations", d.handleAPIRecommendations)
	http.HandleFunc("/static/", d.handleStatic)

	addr := fmt.Sprintf(":%d", d.port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		d.optimizer.app.Logger.Printf("⚠️  Dashboard server failed: %v", err)
	}
}

// UpdateAnalysis updates the dashboard with new analysis data
func (d *Dashboard) UpdateAnalysis(analysis *CostAnalysis) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.latestAnalysis = analysis
	d.optimizer.app.Logger.Printf("📊 Dashboard updated with analysis from %s", analysis.Timestamp.Format("15:04:05"))
}

// handleDashboard serves the main dashboard HTML
func (d *Dashboard) handleDashboard(w http.ResponseWriter, r *http.Request) {
	d.mutex.RLock()
	analysis := d.latestAnalysis
	d.mutex.RUnlock()

	// Create dashboard HTML template
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Cost Optimization Dashboard</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #f5f5f7; color: #1d1d1f; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; margin-bottom: 40px; }
        .header h1 { font-size: 2.5rem; font-weight: 600; margin-bottom: 10px; }
        .header p { font-size: 1.1rem; color: #666; }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 40px; }
        .stat-card { background: white; border-radius: 12px; padding: 24px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); border: 1px solid #e5e5e7; }
        .stat-value { font-size: 2rem; font-weight: 700; margin-bottom: 8px; }
        .stat-label { font-size: 0.9rem; color: #666; text-transform: uppercase; letter-spacing: 0.5px; }
        .savings { color: #30a14e; }
        .cost { color: #d73a49; }
        .utilization { color: #0366d6; }
        .section { background: white; border-radius: 12px; padding: 24px; margin-bottom: 20px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); border: 1px solid #e5e5e7; }
        .section h2 { font-size: 1.5rem; margin-bottom: 20px; }
        .recommendations { margin-bottom: 20px; }
        .recommendation { background: #f8f9fa; border-radius: 8px; padding: 16px; margin-bottom: 12px; border-left: 4px solid #0366d6; }
        .recommendation.high { border-left-color: #d73a49; }
        .recommendation.medium { border-left-color: #fb8500; }
        .recommendation.low { border-left-color: #30a14e; }
        .rec-header { display: flex; justify-content: between; align-items: center; margin-bottom: 8px; }
        .rec-resource { font-weight: 600; }
        .rec-savings { color: #30a14e; font-weight: 600; }
        .rec-explanation { color: #666; font-size: 0.9rem; margin-bottom: 8px; }
        .rec-details { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; font-size: 0.8rem; }
        .detail-group { }
        .detail-label { font-weight: 600; color: #666; margin-bottom: 4px; }
        .breakdown-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; }
        .breakdown-item { text-align: center; }
        .breakdown-value { font-size: 1.5rem; font-weight: 600; color: #0366d6; }
        .breakdown-label { font-size: 0.9rem; color: #666; margin-top: 4px; }
        .status { padding: 8px 12px; border-radius: 6px; font-size: 0.8rem; font-weight: 600; }
        .status.running { background: #d4edda; color: #155724; }
        .status.error { background: #f8d7da; color: #721c24; }
        .refresh-info { text-align: center; color: #666; font-size: 0.9rem; margin-top: 20px; }
        .no-data { text-align: center; color: #666; padding: 40px; }
    </style>
    <script>
        // Auto-refresh every 30 seconds
        setInterval(() => {
            window.location.reload();
        }, 30000);
    </script>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>💰 Cost Optimization Dashboard</h1>
            {{if .Analysis}}
            <p>Cluster: <strong>{{.Analysis.ClusterSummary.ClusterName}}</strong> | Context: <strong>{{.Analysis.ClusterSummary.ClusterContext}}</strong> | ConfigHub Space: <strong>{{.Analysis.ConfigHubSpace}}</strong></p>
            <div class="status running">✅ Active - Last updated: {{.Analysis.Timestamp.Format "2006-01-02 15:04:05"}}</div>
            {{else}}
            <p>AI-powered Kubernetes cost analysis and recommendations</p>
            <div class="status error">⏳ Waiting for first analysis...</div>
            {{end}}
        </div>

        {{if .Analysis}}
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-value cost">${{printf "%.2f" .Analysis.TotalMonthlyCost}}</div>
                <div class="stat-label">Total Monthly Cost</div>
            </div>
            <div class="stat-card">
                <div class="stat-value savings">${{printf "%.2f" .Analysis.PotentialSavings}}</div>
                <div class="stat-label">Potential Savings</div>
            </div>
            <div class="stat-card">
                <div class="stat-value savings">{{printf "%.1f" .Analysis.SavingsPercentage}}%</div>
                <div class="stat-label">Savings Percentage</div>
            </div>
            <div class="stat-card">
                <div class="stat-value utilization">{{len .Analysis.Recommendations}}</div>
                <div class="stat-label">Recommendations</div>
            </div>
        </div>

        <div class="section">
            <h2>📊 Cost Breakdown</h2>
            <div class="breakdown-grid">
                <div class="breakdown-item">
                    <div class="breakdown-value">${{printf "%.2f" .Analysis.ResourceBreakdown.Compute}}</div>
                    <div class="breakdown-label">Compute</div>
                </div>
                <div class="breakdown-item">
                    <div class="breakdown-value">${{printf "%.2f" .Analysis.ResourceBreakdown.Memory}}</div>
                    <div class="breakdown-label">Memory</div>
                </div>
                <div class="breakdown-item">
                    <div class="breakdown-value">${{printf "%.2f" .Analysis.ResourceBreakdown.Storage}}</div>
                    <div class="breakdown-label">Storage</div>
                </div>
                <div class="breakdown-item">
                    <div class="breakdown-value">${{printf "%.2f" .Analysis.ResourceBreakdown.Network}}</div>
                    <div class="breakdown-label">Network</div>
                </div>
            </div>
        </div>

        <div class="section">
            <h2>🎯 Optimization Recommendations</h2>
            {{if .Analysis.Recommendations}}
            <div class="recommendations">
                {{range .Analysis.Recommendations}}
                <div class="recommendation {{.Priority}}">
                    <div class="rec-header">
                        <div class="rec-resource">{{.Resource}}</div>
                        <div class="rec-savings">Save ${{printf "%.2f" .MonthlySavings}}/month</div>
                    </div>
                    <div class="rec-explanation">{{.Explanation}}</div>
                    <div class="rec-details">
                        <div class="detail-group">
                            <div class="detail-label">Type:</div>
                            <div>{{.Type}} ({{.Priority}} priority)</div>
                        </div>
                        <div class="detail-group">
                            <div class="detail-label">Risk Level:</div>
                            <div>{{.Risk}}</div>
                        </div>
                    </div>
                </div>
                {{end}}
            </div>
            {{else}}
            <div class="no-data">No recommendations available yet.</div>
            {{end}}
        </div>

        <div class="section">
            <h2>📊 Resource Details & Metrics</h2>
            {{if .Analysis.ResourceDetails}}
            <table style="width: 100%; border-collapse: collapse;">
                <thead>
                    <tr style="background: #f0f0f0;">
                        <th style="padding: 8px; text-align: left;">Resource</th>
                        <th style="padding: 8px; text-align: left;">Namespace</th>
                        <th style="padding: 8px; text-align: center;">Replicas</th>
                        <th style="padding: 8px; text-align: center;">CPU Requested</th>
                        <th style="padding: 8px; text-align: center;">CPU Used</th>
                        <th style="padding: 8px; text-align: center;">CPU Util %</th>
                        <th style="padding: 8px; text-align: center;">Memory Requested</th>
                        <th style="padding: 8px; text-align: center;">Memory Used</th>
                        <th style="padding: 8px; text-align: center;">Mem Util %</th>
                        <th style="padding: 8px; text-align: right;">Monthly Cost</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Analysis.ResourceDetails}}
                    <tr style="border-bottom: 1px solid #e0e0e0;">
                        <td style="padding: 8px;">{{.Name}}</td>
                        <td style="padding: 8px;">{{.Namespace}}</td>
                        <td style="padding: 8px; text-align: center;">{{.Replicas}}</td>
                        <td style="padding: 8px; text-align: center;">{{.CPURequested}}m</td>
                        <td style="padding: 8px; text-align: center;">{{.CPUUsed}}m</td>
                        <td style="padding: 8px; text-align: center; color: {{if lt .CPUUtilization 30.0}}#d73a49{{else if lt .CPUUtilization 70.0}}#fb8500{{else}}#30a14e{{end}}">{{printf "%.1f" .CPUUtilization}}%</td>
                        <td style="padding: 8px; text-align: center;">{{.MemRequested}}B</td>
                        <td style="padding: 8px; text-align: center;">{{.MemUsed}}B</td>
                        <td style="padding: 8px; text-align: center; color: {{if lt .MemUtilization 30.0}}#d73a49{{else if lt .MemUtilization 70.0}}#fb8500{{else}}#30a14e{{end}}">{{printf "%.1f" .MemUtilization}}%</td>
                        <td style="padding: 8px; text-align: right; font-weight: 600;">${{printf "%.2f" .MonthlyCost}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
            {{else}}
            <div class="no-data">No resource details available.</div>
            {{end}}
        </div>

        <div class="section">
            <h2>🏗️ Cluster Summary</h2>
            <div class="breakdown-grid">
                <div class="breakdown-item">
                    <div class="breakdown-value">{{.Analysis.ClusterSummary.TotalNodes}}</div>
                    <div class="breakdown-label">Nodes</div>
                </div>
                <div class="breakdown-item">
                    <div class="breakdown-value">{{.Analysis.ClusterSummary.TotalPods}}</div>
                    <div class="breakdown-label">Pods</div>
                </div>
                <div class="breakdown-item">
                    <div class="breakdown-value">{{.Analysis.ClusterSummary.TotalDeployments}}</div>
                    <div class="breakdown-label">Deployments</div>
                </div>
                <div class="breakdown-item">
                    <div class="breakdown-value">{{printf "%.1f" .Analysis.ClusterSummary.AvgCPUUtil}}%</div>
                    <div class="breakdown-label">Avg CPU Util</div>
                </div>
            </div>
        </div>

        <div class="section">
            <h2>🔍 ConfigHub Verification</h2>
            <p><strong>Space ID:</strong> {{.Analysis.ConfigHubSpace}}</p>
            <p><strong>Cluster Type:</strong> {{.Analysis.ClusterSummary.ClusterType}} | <strong>Version:</strong> {{.Analysis.ClusterSummary.KubernetesVersion}}</p>
            {{if .Analysis.DataSource}}
            <div style="margin-top: 15px;">
                <h3>Data Sources:</h3>
                <ul style="list-style: none; padding: 0;">
                    <li>✅ Kubernetes API: <strong>{{.Analysis.DataSource.KubernetesAPI}}</strong></li>
                    <li>{{if .Analysis.DataSource.MetricsServer}}✅{{else}}⚠️{{end}} Metrics Server: <strong>{{.Analysis.DataSource.MetricsServer}}</strong> {{if not .Analysis.DataSource.MetricsServer}}(Using 50% simulated utilization){{end}}</li>
                    <li>✅ ConfigHub Sets: <strong>{{.Analysis.DataSource.ConfigHubSets}}</strong></li>
                    <li>✅ Claude AI: <strong>{{.Analysis.DataSource.ClaudeAI}}</strong></li>
                </ul>
                <div style="margin-top: 10px; padding: 10px; background: #e7f3ff; border-left: 4px solid #0066cc; border-radius: 4px;">
                    <p style="margin: 0; font-size: 0.9rem; color: #003d7a;">
                        <strong>📝 Claude API Logs:</strong> If you are using Claude API then review prompt and response session history here: <code style="background: #f1f3f5; padding: 2px 4px; border-radius: 3px;">logs/claude-analysis-latest.log</code>
                    </p>
                </div>
            </div>
            {{end}}
            <div style="margin-top: 15px; padding: 10px; background: #f8f9fa; border-radius: 6px;">
                <p style="font-size: 0.9rem; color: #666;">
                    <strong>Note:</strong> {{if eq .Analysis.ClusterSummary.ClusterType "kind"}}Running on local Kind cluster. Metrics are simulated at 50% utilization. Deploy metrics-server for real metrics.{{else}}Production {{.Analysis.ClusterSummary.ClusterType}} cluster with real metrics.{{end}}
                </p>
            </div>
        </div>

        <div class="section">
            <h2>📦 Namespace Mapping</h2>
            {{if .Analysis.ClusterSummary.Namespaces}}
            <table style="width: 100%; border-collapse: collapse;">
                <thead>
                    <tr style="background: #f0f0f0;">
                        <th style="padding: 8px; text-align: left;">Namespace</th>
                        <th style="padding: 8px; text-align: left;">Description</th>
                        <th style="padding: 8px; text-align: center;">Resources</th>
                        <th style="padding: 8px; text-align: left;">ConfigHub Unit</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Analysis.ClusterSummary.Namespaces}}
                    <tr style="border-bottom: 1px solid #e0e0e0;">
                        <td style="padding: 8px;">{{.Name}}</td>
                        <td style="padding: 8px; color: #666;">{{.Description}}</td>
                        <td style="padding: 8px; text-align: center;">{{.ResourceCount}}</td>
                        <td style="padding: 8px;">{{if .ConfigHubUnit}}{{.ConfigHubUnit}}{{else}}-{{end}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
            {{else}}
            <p style="color: #666;">No namespace information available.</p>
            {{end}}
        </div>
        {{else}}
        <div class="no-data">
            <h2>⏳ Initializing Cost Analysis...</h2>
            <p>The cost optimizer is starting up and will begin analysis shortly.</p>
            <p>This page will refresh automatically when data is available.</p>
        </div>
        {{end}}

        <div class="refresh-info">
            Dashboard auto-refreshes every 30 seconds |
            <a href="/api/analysis" target="_blank">Raw JSON API</a> |
            Health: <a href=":8080/health" target="_blank">:8080/health</a>
        </div>
    </div>
</body>
</html>`

	// Parse and execute template
	t, err := template.New("dashboard").Parse(tmpl)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Analysis *CostAnalysis
	}{
		Analysis: analysis,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := t.Execute(w, data); err != nil {
		d.optimizer.app.Logger.Printf("Template execution error: %v", err)
		http.Error(w, "Template execution error", http.StatusInternalServerError)
	}
}

// handleAPIAnalysis serves the analysis data as JSON
func (d *Dashboard) handleAPIAnalysis(w http.ResponseWriter, r *http.Request) {
	d.mutex.RLock()
	analysis := d.latestAnalysis
	d.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if analysis == nil {
		json.NewEncoder(w).Encode(map[string]string{
			"status": "waiting",
			"message": "No analysis data available yet",
		})
		return
	}

	json.NewEncoder(w).Encode(analysis)
}

// handleAPIRecommendations serves just the recommendations as JSON
func (d *Dashboard) handleAPIRecommendations(w http.ResponseWriter, r *http.Request) {
	d.mutex.RLock()
	analysis := d.latestAnalysis
	d.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if analysis == nil {
		json.NewEncoder(w).Encode([]CostRecommendation{})
		return
	}

	json.NewEncoder(w).Encode(analysis.Recommendations)
}

// handleStatic serves static files (placeholder for future CSS/JS)
func (d *Dashboard) handleStatic(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}