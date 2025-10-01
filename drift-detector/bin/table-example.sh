#!/bin/bash
# Example: Using SDK table-renderer from bash scripts
# This shows how to integrate ASCII tables into bash-based verification

# Example 1: ConfigHub Spaces
echo "📊 ConfigHub Spaces:"
echo ""

# Get spaces from ConfigHub and format as table
cub space list --json 2>/dev/null | jq -r '{
  headers: ["Space ID", "Slug", "Display Name", "Created"],
  rows: [.[] | [
    .space_id[0:8] + "...",
    .slug,
    .display_name,
    .created_at[0:10]
  ]],
  style: "default"
}' | table-renderer

echo ""

# Example 2: ConfigHub Units
echo "📦 ConfigHub Units:"
echo ""

SPACE="drift-detector"
cub unit list --space $SPACE --json 2>/dev/null | jq -r '{
  headers: ["Unit ID", "Slug", "Type", "Upstream"],
  rows: [.[] | [
    .unit_id[0:8] + "...",
    .slug,
    (.labels.type // "app"),
    (if .upstream_unit_id then "✓" else "-" end)
  ]],
  style: "default"
}' | table-renderer

echo ""

# Example 3: Kubernetes Resources
echo "☸️  Kubernetes Resources:"
echo ""

kubectl get pods --all-namespaces -o json 2>/dev/null | jq -r '{
  headers: ["Namespace", "Pod", "Status", "Restarts", "Age"],
  rows: [.items[] | [
    .metadata.namespace,
    .metadata.name[0:30],
    .status.phase,
    (.status.containerStatuses[0].restartCount | tostring),
    .metadata.creationTimestamp[0:10]
  ]][0:10],
  style: "default"
}' | table-renderer

echo ""

# Example 4: Simple data
echo "💰 Cost Analysis:"
echo ""

echo '{
  "headers": ["Resource", "CPU", "Memory", "Cost/mo"],
  "rows": [
    ["frontend-web", "30%", "33%", "$245.50"],
    ["backend-api", "40%", "40%", "$408.75"],
    ["cache-redis", "15%", "15%", "$89.25"]
  ],
  "style": "default"
}' | table-renderer
