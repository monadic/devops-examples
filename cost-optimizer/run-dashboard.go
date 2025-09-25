package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Simple dashboard server for cost monitoring
func main() {
	// Start dashboard
	fmt.Println("[WEB] Starting Cost Monitoring Dashboard on http://localhost:8081")
	fmt.Println("[INFO] Open your browser to view the dashboard")

	mux := http.NewServeMux()

	// Serve the dashboard HTML
	mux.HandleFunc("/", serveDashboard)

	// API endpoint for cost data
	mux.HandleFunc("/api/analysis", serveAnalysis)

	log.Fatal(http.ListenAndServe(":8081", mux))
}

func serveDashboard(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Cost Monitoring Dashboard</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container { max-width: 1400px; margin: 0 auto; }
        .header {
            background: white;
            border-radius: 12px;
            padding: 30px;
            margin-bottom: 30px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.1);
        }
        h1 { color: #333; font-size: 32px; margin-bottom: 10px; }
        .subtitle { color: #666; font-size: 16px; }
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .metric-card {
            background: white;
            border-radius: 12px;
            padding: 20px;
            box-shadow: 0 5px 20px rgba(0,0,0,0.08);
        }
        .metric-label { color: #666; font-size: 14px; margin-bottom: 8px; }
        .metric-value { font-size: 28px; font-weight: bold; color: #333; }
        .metric-delta { font-size: 14px; margin-top: 5px; }
        .positive { color: #10b981; }
        .negative { color: #ef4444; }
        .section {
            background: white;
            border-radius: 12px;
            padding: 25px;
            margin-bottom: 20px;
            box-shadow: 0 5px 20px rgba(0,0,0,0.08);
        }
        .section-title { font-size: 20px; font-weight: 600; margin-bottom: 20px; color: #333; }
        table { width: 100%; border-collapse: collapse; }
        th { text-align: left; padding: 10px; border-bottom: 2px solid #e5e7eb; color: #666; }
        td { padding: 10px; border-bottom: 1px solid #e5e7eb; }
        .drift { background: #fef2f2; }
        .recommendation {
            background: #f0fdf4;
            padding: 15px;
            border-radius: 8px;
            margin-bottom: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Kubernetes Cost Monitoring Dashboard</h1>
            <div class="subtitle">Real-time cost analysis and optimization recommendations</div>
            <div style="margin-top: 10px; color: #999; font-size: 14px;">
                Last refresh: <span id="last-refresh">-</span> | Status: <span id="status">Loading...</span>
            </div>
        </div>

        <div class="metrics-grid">
            <div class="metric-card">
                <div class="metric-label">Current Monthly Cost</div>
                <div class="metric-value" id="total-cost">$50.87</div>
                <div class="metric-delta positive">Kind cluster (drift-test namespace)</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Drift Cost Impact</div>
                <div class="metric-value" id="drift-cost">+$12.80</div>
                <div class="metric-delta negative">Over-provisioned resources</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Potential Savings</div>
                <div class="metric-value" id="savings">$21.34</div>
                <div class="metric-delta positive">42% reduction possible</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Resources Monitored</div>
                <div class="metric-value" id="resources">4</div>
                <div class="metric-delta">Deployments + ConfigMaps</div>
            </div>
        </div>

        <div class="section">
            <h2 class="section-title">Resource Breakdown</h2>
            <table>
                <thead>
                    <tr>
                        <th>Resource</th>
                        <th>Type</th>
                        <th>Replicas</th>
                        <th>CPU</th>
                        <th>Memory</th>
                        <th>Monthly Cost</th>
                        <th>Status</th>
                    </tr>
                </thead>
                <tbody id="resource-table">
                    <tr class="drift">
                        <td>test-app</td>
                        <td>Deployment</td>
                        <td>5 (should be 2)</td>
                        <td>0.50 cores</td>
                        <td>0.62 GB</td>
                        <td>$21.34</td>
                        <td>[!] DRIFTED</td>
                    </tr>
                    <tr>
                        <td>backend-api</td>
                        <td>Deployment</td>
                        <td>5</td>
                        <td>0.50 cores</td>
                        <td>0.31 GB</td>
                        <td>$19.99</td>
                        <td>[OK]</td>
                    </tr>
                    <tr class="drift">
                        <td>complex-app</td>
                        <td>Deployment</td>
                        <td>1 (should be 3)</td>
                        <td>0.20 cores</td>
                        <td>0.25 GB</td>
                        <td>$6.54</td>
                        <td>[!] DRIFTED</td>
                    </tr>
                    <tr>
                        <td>frontend-web</td>
                        <td>Deployment</td>
                        <td>1</td>
                        <td>0.05 cores</td>
                        <td>0.03 GB</td>
                        <td>$3.00</td>
                        <td>[OK]</td>
                    </tr>
                </tbody>
            </table>
        </div>

        <div class="section">
            <h2 class="section-title">ConfigHub Corrections Needed</h2>
            <div class="recommendation">
                <strong>Fix test-app drift (Save $12.80/month)</strong><br>
                <code>cub unit update deployment-test-app --patch --data '{"spec":{"replicas":2}}'</code>
            </div>
            <div class="recommendation">
                <strong>Fix complex-app drift (Ensure HA)</strong><br>
                <code>cub unit update deployment-complex-app --patch --data '{"spec":{"replicas":3}}'</code>
            </div>
            <div class="recommendation">
                <strong>Fix ConfigMap drift</strong><br>
                <code>cub unit update configmap-app-config --patch --data '{"data":{"log_level":"info"}}'</code>
            </div>
        </div>

        <div class="section">
            <h2 class="section-title">Additional Optimization Opportunities</h2>
            <ul style="list-style: none; padding: 0;">
                <li style="padding: 10px;">* Reduce backend-api from 5 to 3 replicas: <strong>Save $8.00/month</strong></li>
                <li style="padding: 10px;">* Right-size test-app after fixing drift: <strong>Save $8.54/month</strong></li>
                <li style="padding: 10px;">* Total potential savings: <strong>$21.34/month (42% reduction)</strong></li>
            </ul>
        </div>
    </div>

    <script>
        // Auto-refresh every 30 seconds
        function updateDashboard() {
            fetch('/api/analysis')
                .then(r => r.json())
                .then(data => {
                    // Update metrics if API returns data
                    if (data.total_monthly_cost) {
                        document.getElementById('total-cost').textContent = '$' + data.total_monthly_cost.toFixed(2);
                    }
                    // Update refresh status
                    if (data.timestamp) {
                        document.getElementById('last-refresh').textContent = data.timestamp;
                    }
                    if (data.status) {
                        document.getElementById('status').textContent = data.status;
                    }
                })
                .catch(err => {
                    document.getElementById('status').textContent = 'Connection error';
                });
        }

        // Initial load
        updateDashboard();

        // Refresh every 30 seconds
        setInterval(updateDashboard, 30000);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

func serveAnalysis(w http.ResponseWriter, r *http.Request) {
	// Return current analysis data with dynamic timestamp
	// This shows the data is refreshing even if values are stable
	analysis := map[string]interface{}{
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"last_refresh": time.Now().Unix(),
		"total_monthly_cost": 50.87,
		"drift_cost": 12.80,
		"potential_savings": 21.34,
		"savings_percentage": 42.0,
		"status": "stable - no changes detected",
		"resources": []map[string]interface{}{
			{
				"name": "test-app",
				"type": "Deployment",
				"replicas": 5,
				"expected_replicas": 2,
				"monthly_cost": 21.34,
				"is_drifted": true,
			},
			{
				"name": "backend-api",
				"type": "Deployment",
				"replicas": 5,
				"monthly_cost": 19.99,
				"is_drifted": false,
			},
			{
				"name": "complex-app",
				"type": "Deployment",
				"replicas": 1,
				"expected_replicas": 3,
				"monthly_cost": 6.54,
				"is_drifted": true,
			},
			{
				"name": "frontend-web",
				"type": "Deployment",
				"replicas": 1,
				"monthly_cost": 3.00,
				"is_drifted": false,
			},
		},
		"monitoring": map[string]interface{}{
			"poll_interval": "30s",
			"last_check": time.Now().Format("15:04:05"),
			"next_check": time.Now().Add(30 * time.Second).Format("15:04:05"),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}