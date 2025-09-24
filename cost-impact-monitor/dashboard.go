package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"time"
)

// MonitorDashboard provides web interface for cost impact monitoring
type MonitorDashboard struct {
	monitor      *CostImpactMonitor
	currentData  *MonitoringSnapshot
	lastUpdate   time.Time
}

// NewMonitorDashboard creates a new dashboard
func NewMonitorDashboard(monitor *CostImpactMonitor) *MonitorDashboard {
	return &MonitorDashboard{
		monitor:    monitor,
		lastUpdate: time.Now(),
	}
}

// Start begins serving the dashboard
func (d *MonitorDashboard) Start() {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/snapshot", d.handleSnapshot)
	mux.HandleFunc("/api/spaces", d.handleSpaces)
	mux.HandleFunc("/api/pending", d.handlePendingChanges)
	mux.HandleFunc("/api/triggers", d.handleTriggers)
	mux.HandleFunc("/api/history", d.handleHistory)

	// Main dashboard
	mux.HandleFunc("/", d.handleDashboard)

	// Static resources
	mux.HandleFunc("/static/", d.handleStatic)

	port := ":8083"
	log.Printf("📊 Cost Impact Monitor Dashboard: http://localhost%s", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Printf("Dashboard server error: %v", err)
	}
}

// UpdateMonitoringData updates the dashboard data
func (d *MonitorDashboard) UpdateMonitoringData(snapshot *MonitoringSnapshot) {
	d.currentData = snapshot
	d.lastUpdate = time.Now()
}

// handleSnapshot returns current monitoring snapshot
func (d *MonitorDashboard) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if d.currentData == nil {
		d.currentData = d.monitor.getMonitoringSnapshot()
	}

	if err := json.NewEncoder(w).Encode(d.currentData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleSpaces returns detailed space information
func (d *MonitorDashboard) handleSpaces(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	d.monitor.mu.RLock()
	spaces := make([]*SpaceMonitor, 0, len(d.monitor.monitoredSpaces))
	for _, space := range d.monitor.monitoredSpaces {
		spaces = append(spaces, space)
	}
	d.monitor.mu.RUnlock()

	// Sort by projected cost (highest first)
	sort.Slice(spaces, func(i, j int) bool {
		return spaces[i].ProjectedCost > spaces[j].ProjectedCost
	})

	response := map[string]interface{}{
		"spaces":      spaces,
		"total":       len(spaces),
		"last_update": d.lastUpdate,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handlePendingChanges returns all pending changes across spaces
func (d *MonitorDashboard) handlePendingChanges(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var allChanges []map[string]interface{}

	d.monitor.mu.RLock()
	for _, space := range d.monitor.monitoredSpaces {
		for _, change := range space.PendingChanges {
			changeData := map[string]interface{}{
				"space_name":        space.SpaceName,
				"space_id":          space.SpaceID,
				"unit_name":         change.UnitName,
				"change_type":       change.ChangeType,
				"current_cost":      change.CurrentCost,
				"projected_cost":    change.ProjectedCost,
				"cost_delta":        change.CostDelta,
				"risk_level":        change.RiskLevel,
				"analysis_time":     change.AnalysisTime,
				"claude_assessment": change.ClaudeAssessment,
			}
			allChanges = append(allChanges, changeData)
		}
	}
	d.monitor.mu.RUnlock()

	// Sort by risk level and cost delta
	sort.Slice(allChanges, func(i, j int) bool {
		riskOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
		if riskOrder[allChanges[i]["risk_level"].(string)] != riskOrder[allChanges[j]["risk_level"].(string)] {
			return riskOrder[allChanges[i]["risk_level"].(string)] < riskOrder[allChanges[j]["risk_level"].(string)]
		}
		return allChanges[i]["cost_delta"].(float64) > allChanges[j]["cost_delta"].(float64)
	})

	response := map[string]interface{}{
		"pending_changes": allChanges,
		"total":          len(allChanges),
		"last_update":    d.lastUpdate,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleTriggers returns trigger activity
func (d *MonitorDashboard) handleTriggers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get recent trigger activity
	d.monitor.triggerProcessor.mu.Lock()
	recentTriggers := make([]map[string]interface{}, 0)
	for unitID, lastProcessed := range d.monitor.triggerProcessor.lastProcessed {
		if time.Since(lastProcessed) < 1*time.Hour {
			trigger := map[string]interface{}{
				"unit_id":        unitID,
				"last_processed": lastProcessed,
				"age":           time.Since(lastProcessed).String(),
			}
			recentTriggers = append(recentTriggers, trigger)
		}
	}
	d.monitor.triggerProcessor.mu.Unlock()

	response := map[string]interface{}{
		"recent_triggers": recentTriggers,
		"total":          len(recentTriggers),
		"last_update":    d.lastUpdate,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleHistory returns deployment history with accuracy tracking
func (d *MonitorDashboard) handleHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var allHistory []DeploymentCostRecord

	d.monitor.mu.RLock()
	for _, space := range d.monitor.monitoredSpaces {
		allHistory = append(allHistory, space.DeploymentHistory...)
	}
	d.monitor.mu.RUnlock()

	// Sort by deploy time (newest first)
	sort.Slice(allHistory, func(i, j int) bool {
		return allHistory[i].DeployTime.After(allHistory[j].DeployTime)
	})

	// Calculate accuracy metrics
	totalRecords := len(allHistory)
	accurateCount := 0
	for _, record := range allHistory {
		if record.Accurate {
			accurateCount++
		}
	}

	accuracyRate := 0.0
	if totalRecords > 0 {
		accuracyRate = (float64(accurateCount) / float64(totalRecords)) * 100
	}

	response := map[string]interface{}{
		"history":       allHistory,
		"total":         totalRecords,
		"accuracy_rate": accuracyRate,
		"last_update":   d.lastUpdate,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleDashboard serves the main dashboard HTML
func (d *MonitorDashboard) handleDashboard(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>ConfigHub Cost Impact Monitor</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
        }
        .header {
            background: white;
            border-radius: 12px;
            padding: 30px;
            margin-bottom: 30px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            font-size: 32px;
            margin-bottom: 10px;
        }
        .subtitle {
            color: #666;
            font-size: 16px;
        }
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
        .metric-label {
            color: #666;
            font-size: 14px;
            margin-bottom: 8px;
        }
        .metric-value {
            font-size: 28px;
            font-weight: bold;
            color: #333;
        }
        .metric-delta {
            font-size: 14px;
            margin-top: 5px;
        }
        .positive { color: #10b981; }
        .negative { color: #ef4444; }
        .neutral { color: #6b7280; }
        .section {
            background: white;
            border-radius: 12px;
            padding: 25px;
            margin-bottom: 20px;
            box-shadow: 0 5px 20px rgba(0,0,0,0.08);
        }
        .section-title {
            font-size: 20px;
            font-weight: 600;
            margin-bottom: 20px;
            color: #333;
        }
        .pending-changes {
            display: grid;
            gap: 15px;
        }
        .change-card {
            border: 1px solid #e5e7eb;
            border-radius: 8px;
            padding: 15px;
            display: grid;
            grid-template-columns: 1fr auto;
            gap: 10px;
        }
        .change-info {
            display: flex;
            flex-direction: column;
            gap: 5px;
        }
        .change-name {
            font-weight: 600;
            color: #333;
        }
        .change-details {
            color: #666;
            font-size: 14px;
        }
        .risk-badge {
            padding: 4px 12px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: 600;
            text-transform: uppercase;
            align-self: start;
        }
        .risk-critical { background: #fef2f2; color: #991b1b; }
        .risk-high { background: #fef3c7; color: #92400e; }
        .risk-medium { background: #fef9c3; color: #713f12; }
        .risk-low { background: #f0fdf4; color: #166534; }
        .space-list {
            display: grid;
            gap: 12px;
        }
        .space-row {
            display: grid;
            grid-template-columns: 2fr 1fr 1fr 1fr 100px;
            gap: 15px;
            padding: 12px;
            border: 1px solid #e5e7eb;
            border-radius: 8px;
            align-items: center;
        }
        .space-name {
            font-weight: 600;
            color: #333;
        }
        .trend-indicator {
            display: inline-block;
            width: 0;
            height: 0;
            margin-left: 5px;
        }
        .trend-up {
            border-left: 5px solid transparent;
            border-right: 5px solid transparent;
            border-bottom: 8px solid #ef4444;
        }
        .trend-down {
            border-left: 5px solid transparent;
            border-right: 5px solid transparent;
            border-top: 8px solid #10b981;
        }
        .trend-stable {
            width: 10px;
            height: 2px;
            background: #6b7280;
        }
        .refresh-indicator {
            position: fixed;
            bottom: 20px;
            right: 20px;
            background: white;
            padding: 10px 20px;
            border-radius: 20px;
            box-shadow: 0 5px 20px rgba(0,0,0,0.1);
            font-size: 14px;
            color: #666;
        }
        .loading {
            animation: pulse 2s infinite;
        }
        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔍 ConfigHub Cost Impact Monitor</h1>
            <div class="subtitle">Real-time cost tracking for all ConfigHub deployments</div>
        </div>

        <div class="metrics-grid">
            <div class="metric-card">
                <div class="metric-label">Total Monthly Cost</div>
                <div class="metric-value" id="total-cost">Loading...</div>
                <div class="metric-delta neutral" id="cost-trend"></div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Projected Cost</div>
                <div class="metric-value" id="projected-cost">Loading...</div>
                <div class="metric-delta" id="projected-delta"></div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Pending Changes</div>
                <div class="metric-value" id="pending-count">0</div>
                <div class="metric-delta negative" id="high-risk"></div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Monitored Spaces</div>
                <div class="metric-value" id="space-count">0</div>
                <div class="metric-delta neutral" id="accuracy"></div>
            </div>
        </div>

        <div class="section">
            <h2 class="section-title">⚠️ Pending Changes (Pre-Deployment Analysis)</h2>
            <div class="pending-changes" id="pending-changes">
                <div class="loading">Loading pending changes...</div>
            </div>
        </div>

        <div class="section">
            <h2 class="section-title">📦 ConfigHub Spaces</h2>
            <div class="space-list" id="space-list">
                <div class="loading">Loading spaces...</div>
            </div>
        </div>

        <div class="section">
            <h2 class="section-title">📊 Deployment History & Accuracy</h2>
            <div id="history-chart">
                <canvas id="accuracy-chart" height="100"></canvas>
            </div>
        </div>
    </div>

    <div class="refresh-indicator" id="refresh">
        Last updated: <span id="last-update">Never</span>
    </div>

    <script>
        // Fetch and display data
        async function updateDashboard() {
            try {
                // Get snapshot
                const snapshotRes = await fetch('/api/snapshot');
                const snapshot = await snapshotRes.json();

                // Update main metrics
                document.getElementById('total-cost').textContent = '$' + snapshot.total_cost.toFixed(2);
                document.getElementById('projected-cost').textContent = '$' + snapshot.projected_cost.toFixed(2);
                document.getElementById('pending-count').textContent = snapshot.pending_changes;
                document.getElementById('space-count').textContent = snapshot.total_spaces;

                // Calculate deltas
                const costDelta = snapshot.projected_cost - snapshot.total_cost;
                const deltaElement = document.getElementById('projected-delta');
                deltaElement.textContent = (costDelta >= 0 ? '+' : '') + '$' + costDelta.toFixed(2);
                deltaElement.className = 'metric-delta ' + (costDelta > 0 ? 'negative' : 'positive');

                // High risk count
                if (snapshot.high_risk_changes > 0) {
                    document.getElementById('high-risk').textContent = snapshot.high_risk_changes + ' high risk';
                }

                // Get pending changes
                const pendingRes = await fetch('/api/pending');
                const pendingData = await pendingRes.json();
                displayPendingChanges(pendingData.pending_changes);

                // Get spaces
                const spacesRes = await fetch('/api/spaces');
                const spacesData = await spacesRes.json();
                displaySpaces(spacesData.spaces);

                // Get history for accuracy
                const historyRes = await fetch('/api/history');
                const historyData = await historyRes.json();
                document.getElementById('accuracy').textContent =
                    'Prediction accuracy: ' + historyData.accuracy_rate.toFixed(1) + '%';

                // Update timestamp
                document.getElementById('last-update').textContent = new Date().toLocaleTimeString();

            } catch (error) {
                console.error('Failed to update dashboard:', error);
            }
        }

        function displayPendingChanges(changes) {
            const container = document.getElementById('pending-changes');

            if (!changes || changes.length === 0) {
                container.innerHTML = '<div style="color: #666;">No pending changes</div>';
                return;
            }

            container.innerHTML = changes.map(change => ` + "`" + `
                <div class="change-card">
                    <div class="change-info">
                        <div class="change-name">${change.unit_name} (${change.space_name})</div>
                        <div class="change-details">
                            ${change.change_type} •
                            Current: $${change.current_cost.toFixed(2)} →
                            Projected: $${change.projected_cost.toFixed(2)}
                            (${change.cost_delta >= 0 ? '+' : ''}$${change.cost_delta.toFixed(2)})
                        </div>
                        ${change.claude_assessment ? ` + "`" + `<div class="change-details" style="margin-top: 5px; font-style: italic;">"${change.claude_assessment}"</div>` + "`" + ` : ''}
                    </div>
                    <div class="risk-badge risk-${change.risk_level}">${change.risk_level}</div>
                </div>
            ` + "`" + `).join('');
        }

        function displaySpaces(spaces) {
            const container = document.getElementById('space-list');

            if (!spaces || spaces.length === 0) {
                container.innerHTML = '<div style="color: #666;">No spaces monitored</div>';
                return;
            }

            container.innerHTML = spaces.map(space => {
                const trendClass = space.cost_trend.direction === 'increasing' ? 'trend-up' :
                                  space.cost_trend.direction === 'decreasing' ? 'trend-down' : 'trend-stable';

                return ` + "`" + `
                    <div class="space-row">
                        <div class="space-name">${space.space_name}</div>
                        <div>$${space.current_cost.toFixed(2)}/mo</div>
                        <div>$${space.projected_cost.toFixed(2)}/mo</div>
                        <div>${space.pending_changes.length} pending</div>
                        <div>
                            <span class="${trendClass}"></span>
                            ${space.cost_trend.weekly_change ? space.cost_trend.weekly_change.toFixed(1) + '%' : 'N/A'}
                        </div>
                    </div>
                ` + "`" + `;
            }).join('');
        }

        // Initial load and refresh every 10 seconds
        updateDashboard();
        setInterval(updateDashboard, 10000);
    </script>
</body>
</html>`

	t, err := template.New("dashboard").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := t.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleStatic serves static resources
func (d *MonitorDashboard) handleStatic(w http.ResponseWriter, r *http.Request) {
	// In production, would serve actual static files
	http.NotFound(w, r)
}