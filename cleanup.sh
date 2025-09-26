#!/bin/bash
# Global cleanup script for DevOps Examples
# Ensures clean slate before running any example

set -e

echo "🧹 DevOps Examples - Global Cleanup"
echo "===================================="
echo ""
echo "This will clean up ALL resources from previous runs:"
echo "  - Kubernetes deployments in drift-test namespace"
echo "  - ConfigHub spaces matching common patterns"
echo "  - Local .cub-project files"
echo ""

# Confirm with user
read -p "Are you sure you want to clean up all resources? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cleanup cancelled."
    exit 0
fi

echo ""
echo "🗑️  Starting cleanup..."

# Function to safely delete resources
safe_delete() {
    local cmd=$1
    local resource=$2
    if $cmd 2>/dev/null; then
        echo "  ✅ Deleted: $resource"
    else
        echo "  ⏭️  Skipped: $resource (not found or already deleted)"
    fi
}

# Clean up Kubernetes resources
echo ""
echo "📦 Cleaning Kubernetes resources..."

# Delete namespace if it exists (this deletes all resources in it)
if kubectl get namespace drift-test &>/dev/null; then
    echo "  Deleting drift-test namespace and all resources..."
    kubectl delete namespace drift-test --wait=false 2>/dev/null || true
    echo "  ✅ Namespace deletion initiated"
else
    echo "  ⏭️  drift-test namespace doesn't exist"
fi

# Clean up cost-optimizer namespace if exists
if kubectl get namespace cost-optimizer &>/dev/null; then
    echo "  Deleting cost-optimizer namespace..."
    kubectl delete namespace cost-optimizer --wait=false 2>/dev/null || true
    echo "  ✅ Namespace deletion initiated"
fi

# Clean up ConfigHub resources
echo ""
echo "🔧 Cleaning ConfigHub resources..."

# Get all spaces that match our patterns
PATTERNS=(
    "drift-detector"
    "cost-optimizer"
    "devops-example"
    "test-demo"
    "*-drift-*"
    "*-cost-*"
)

for pattern in "${PATTERNS[@]}"; do
    # List spaces matching pattern
    spaces=$(cub space list --quiet 2>/dev/null | grep -E "$pattern" | awk '{print $1}' || true)

    if [ -n "$spaces" ]; then
        echo "  Found spaces matching pattern '$pattern':"
        for space in $spaces; do
            safe_delete "cub space delete $space" "space: $space"
        done
    fi
done

# Delete common workers
echo ""
echo "👷 Cleaning ConfigHub workers..."
workers=$(cub worker list --quiet 2>/dev/null | grep -E "(devops|drift|cost)" | awk '{print $1}' || true)

if [ -n "$workers" ]; then
    for worker in $workers; do
        safe_delete "cub worker delete $worker" "worker: $worker"
    done
else
    echo "  No workers to clean"
fi

# Clean up local project files
echo ""
echo "📁 Cleaning local project files..."

# Find and remove .cub-project files
find . -name ".cub-project" -type f 2>/dev/null | while read -r file; do
    rm -f "$file"
    echo "  ✅ Removed: $file"
done

# Remove .drift-detector-* files
find . -name ".drift-detector-*" -type f 2>/dev/null | while read -r file; do
    rm -f "$file"
    echo "  ✅ Removed: $file"
done

# Remove .cost-optimizer-* files
find . -name ".cost-optimizer-*" -type f 2>/dev/null | while read -r file; do
    rm -f "$file"
    echo "  ✅ Removed: $file"
done

echo ""
echo "✨ Cleanup complete!"
echo ""
echo "You can now run any example with a clean environment:"
echo "  cd drift-detector && ./bin/install-base"
echo "  cd cost-optimizer && ./bin/install-base"
echo ""
echo "Remember to start a worker after setup:"
echo "  ./setup-worker.sh"
echo "  cub worker run devops-worker"