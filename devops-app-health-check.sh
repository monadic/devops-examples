#!/bin/bash

# DevOps App Health Check & Unit Verification
# Comprehensive health check for all units, spaces, and resources

set -e

echo "=========================================="
echo "DevOps App Health Check System"
echo "=========================================="
echo "Date: $(date)"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
SPACE=${1:-drift-test-demo}
NAMESPACE=${2:-drift-test}
API_ENDPOINT=${3:-http://localhost:8082/api/live}

echo "Configuration:"
echo "  ConfigHub Space: $SPACE"
echo "  Kubernetes Namespace: $NAMESPACE"
echo "  API Endpoint: $API_ENDPOINT"
echo ""

# Health metrics
HEALTH_SCORE=100
ISSUES=()

# Function to check health
check_health() {
    local component=$1
    local check_name=$2
    local command=$3
    local expected=$4

    echo -n "  $check_name ... "

    if eval "$command" > /dev/null 2>&1; then
        if [ "$expected" = "pass" ]; then
            echo -e "${GREEN}✓ HEALTHY${NC}"
            return 0
        else
            echo -e "${RED}✗ UNHEALTHY${NC}"
            ((HEALTH_SCORE-=10))
            ISSUES+=("$component: $check_name failed")
            return 1
        fi
    else
        if [ "$expected" = "fail" ]; then
            echo -e "${GREEN}✓ HEALTHY${NC}"
            return 0
        else
            echo -e "${RED}✗ UNHEALTHY${NC}"
            ((HEALTH_SCORE-=10))
            ISSUES+=("$component: $check_name failed")
            return 1
        fi
    fi
}

echo "=========================================="
echo "1. CONFIGHHUB HEALTH CHECK"
echo "=========================================="
echo ""

# Check ConfigHub connectivity
echo "ConfigHub Connection:"
check_health "ConfigHub" "CLI available" "command -v cub" "pass"
check_health "ConfigHub" "Auth valid" "cub auth whoami 2>/dev/null" "pass"
check_health "ConfigHub" "Space exists" "cub space get $SPACE 2>/dev/null" "pass"

echo ""
echo "ConfigHub Units:"
# Get all units in space
UNITS=$(cub unit list --space $SPACE --output json 2>/dev/null | jq -r '.[].name' 2>/dev/null || echo "")
if [ -n "$UNITS" ]; then
    UNIT_COUNT=$(echo "$UNITS" | wc -w)
    echo -e "  Total units: ${GREEN}$UNIT_COUNT${NC}"

    for unit in $UNITS; do
        echo -n "  • $unit: "

        # Check unit details
        UNIT_DATA=$(cub unit get $unit --space $SPACE --output json 2>/dev/null || echo "{}")

        if [ -n "$UNIT_DATA" ] && [ "$UNIT_DATA" != "{}" ]; then
            # Extract key information
            KIND=$(echo "$UNIT_DATA" | jq -r '.kind' 2>/dev/null || echo "unknown")
            REPLICAS=$(echo "$UNIT_DATA" | jq -r '.spec.replicas' 2>/dev/null || echo "N/A")

            echo -e "${GREEN}OK${NC} (Type: $KIND, Replicas: $REPLICAS)"
        else
            echo -e "${RED}ERROR${NC}"
            ((HEALTH_SCORE-=5))
            ISSUES+=("ConfigHub: Unit $unit not accessible")
        fi
    done
else
    echo -e "  ${RED}No units found${NC}"
    ((HEALTH_SCORE-=20))
    ISSUES+=("ConfigHub: No units in space $SPACE")
fi

echo ""
echo "=========================================="
echo "2. KUBERNETES HEALTH CHECK"
echo "=========================================="
echo ""

echo "Kubernetes Cluster:"
check_health "Kubernetes" "Kubectl available" "command -v kubectl" "pass"
check_health "Kubernetes" "Cluster reachable" "kubectl cluster-info 2>/dev/null | grep -q 'running'" "pass"
check_health "Kubernetes" "Namespace exists" "kubectl get namespace $NAMESPACE" "pass"

echo ""
echo "Kubernetes Resources:"
# Get deployments
DEPLOYMENTS=$(kubectl get deployments -n $NAMESPACE -o json 2>/dev/null | jq -r '.items[].metadata.name' 2>/dev/null || echo "")

if [ -n "$DEPLOYMENTS" ]; then
    for deployment in $DEPLOYMENTS; do
        echo -n "  • Deployment $deployment: "

        # Get deployment status
        READY=$(kubectl get deployment $deployment -n $NAMESPACE -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        DESIRED=$(kubectl get deployment $deployment -n $NAMESPACE -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")

        if [ "$READY" = "$DESIRED" ] && [ "$READY" != "0" ]; then
            echo -e "${GREEN}HEALTHY${NC} ($READY/$DESIRED replicas)"
        else
            echo -e "${YELLOW}DEGRADED${NC} ($READY/$DESIRED replicas)"
            ((HEALTH_SCORE-=5))
            ISSUES+=("K8s: Deployment $deployment degraded")
        fi
    done
else
    echo -e "  ${YELLOW}No deployments found${NC}"
fi

echo ""
echo "=========================================="
echo "3. DRIFT DETECTION HEALTH"
echo "=========================================="
echo ""

# Check drift via API
echo "Drift Detection API:"
API_RESPONSE=$(curl -s $API_ENDPOINT 2>/dev/null || echo "{}")

if [ -n "$API_RESPONSE" ] && [ "$API_RESPONSE" != "{}" ]; then
    echo -e "  API Status: ${GREEN}ONLINE${NC}"

    # Parse drift data
    RESOURCES=$(echo "$API_RESPONSE" | jq '.resources' 2>/dev/null || echo "[]")
    RESOURCE_COUNT=$(echo "$RESOURCES" | jq 'length' 2>/dev/null || echo "0")
    DRIFTED_COUNT=$(echo "$RESOURCES" | jq '[.[] | select(.is_drifted == true)] | length' 2>/dev/null || echo "0")

    echo "  Resources monitored: $RESOURCE_COUNT"
    echo "  Resources with drift: $DRIFTED_COUNT"

    if [ "$DRIFTED_COUNT" -gt 0 ]; then
        echo ""
        echo "  Drifted Resources:"
        echo "$RESOURCES" | jq -r '.[] | select(.is_drifted == true) | "    • \(.name): Expected \(.expected_replicas), Got \(.replicas)"' 2>/dev/null

        # Check corrections
        CORRECTIONS=$(echo "$API_RESPONSE" | jq '.corrections' 2>/dev/null || echo "[]")
        if [ "$(echo "$CORRECTIONS" | jq 'length')" -gt 0 ]; then
            echo ""
            echo "  Corrections Available:"
            echo "$CORRECTIONS" | jq -r '.[] | "    • \(.resource): \(.impact)"' 2>/dev/null

            # Verify corrections use ConfigHub
            USES_CUB=$(echo "$CORRECTIONS" | jq -r '.[].command' | grep -c "cub unit" || echo "0")
            USES_KUBECTL=$(echo "$CORRECTIONS" | jq -r '.[].command' | grep -c "kubectl" || echo "0")

            if [ "$USES_KUBECTL" -gt 0 ]; then
                echo -e "  ${RED}✗ CRITICAL: Uses kubectl (prohibited!)${NC}"
                ((HEALTH_SCORE-=30))
                ISSUES+=("CRITICAL: Corrections use kubectl")
            elif [ "$USES_CUB" -gt 0 ]; then
                echo -e "  ${GREEN}✓ Corrections use ConfigHub only${NC}"
            fi
        fi
    else
        echo -e "  ${GREEN}✓ No drift detected${NC}"
    fi
else
    echo -e "  API Status: ${RED}OFFLINE${NC}"
    ((HEALTH_SCORE-=20))
    ISSUES+=("API: Drift detection API not responding")
fi

echo ""
echo "=========================================="
echo "4. UNIT SYNCHRONIZATION CHECK"
echo "=========================================="
echo ""

echo "Verifying ConfigHub ↔ Kubernetes Sync:"

# Compare ConfigHub units with K8s deployments
for unit in $UNITS; do
    # Extract deployment name from unit (assuming pattern like "app-name-unit")
    DEPLOYMENT_NAME=$(echo "$unit" | sed 's/-unit$//' | sed 's/deployment-//')

    echo -n "  $unit → $DEPLOYMENT_NAME: "

    # Check if deployment exists
    if kubectl get deployment $DEPLOYMENT_NAME -n $NAMESPACE > /dev/null 2>&1; then
        # Get replicas from both
        CONFIGHUB_REPLICAS=$(cub unit get $unit --space $SPACE --output json 2>/dev/null | jq -r '.spec.replicas' 2>/dev/null || echo "?")
        K8S_REPLICAS=$(kubectl get deployment $DEPLOYMENT_NAME -n $NAMESPACE -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "?")

        if [ "$CONFIGHUB_REPLICAS" = "$K8S_REPLICAS" ]; then
            echo -e "${GREEN}SYNCED${NC} (Replicas: $CONFIGHUB_REPLICAS)"
        else
            echo -e "${YELLOW}DRIFT${NC} (ConfigHub: $CONFIGHUB_REPLICAS, K8s: $K8S_REPLICAS)"
            ((HEALTH_SCORE-=5))
            ISSUES+=("Sync: $unit has drift")
        fi
    else
        echo -e "${RED}NOT DEPLOYED${NC}"
        ((HEALTH_SCORE-=10))
        ISSUES+=("Sync: $unit not found in Kubernetes")
    fi
done

echo ""
echo "=========================================="
echo "5. APP COMPLIANCE CHECK"
echo "=========================================="
echo ""

echo "Checking App Compliance:"

# Quick compliance checks for running apps
for app_dir in drift-detector cost-optimizer cost-impact-monitor; do
    APP_PATH="/Users/alexisrichardson/github-repos/devops-examples/$app_dir"

    if [ -d "$APP_PATH" ]; then
        echo -n "  $app_dir: "

        # Check for kubectl in code
        if grep -r "kubectl" $APP_PATH/*.go 2>/dev/null | grep -v "//" | grep -v "ConfigHub" > /dev/null; then
            echo -e "${RED}NON-COMPLIANT${NC} (uses kubectl)"
            ((HEALTH_SCORE-=15))
            ISSUES+=("Compliance: $app_dir uses kubectl")
        else
            echo -e "${GREEN}COMPLIANT${NC}"
        fi
    fi
done

echo ""
echo "=========================================="
echo "6. COST IMPACT CHECK"
echo "=========================================="
echo ""

if [ -n "$API_RESPONSE" ] && [ "$API_RESPONSE" != "{}" ]; then
    TOTAL_COST=$(echo "$API_RESPONSE" | jq '.total_monthly_cost' 2>/dev/null || echo "0")
    DRIFT_COST=$(echo "$API_RESPONSE" | jq '.drift_cost' 2>/dev/null || echo "0")
    SAVINGS=$(echo "$API_RESPONSE" | jq '.potential_savings' 2>/dev/null || echo "0")

    echo "Cost Analysis:"
    echo "  Total Monthly Cost: \$$TOTAL_COST"
    echo "  Drift Cost Impact: \$$DRIFT_COST"
    echo "  Potential Savings: \$$SAVINGS"

    if [ "$(echo "$DRIFT_COST > 10" | bc -l 2>/dev/null || echo "0")" = "1" ]; then
        echo -e "  ${YELLOW}⚠ High drift cost detected${NC}"
        ((HEALTH_SCORE-=5))
        ISSUES+=("Cost: High drift cost (\$$DRIFT_COST/month)")
    else
        echo -e "  ${GREEN}✓ Drift cost within acceptable range${NC}"
    fi
fi

echo ""
echo "=========================================="
echo "HEALTH CHECK SUMMARY"
echo "=========================================="
echo ""

# Calculate health status
if [ $HEALTH_SCORE -ge 90 ]; then
    STATUS="${GREEN}✓ HEALTHY${NC}"
    STATUS_TEXT="System is fully operational"
elif [ $HEALTH_SCORE -ge 70 ]; then
    STATUS="${YELLOW}⚠ DEGRADED${NC}"
    STATUS_TEXT="System has minor issues"
else
    STATUS="${RED}✗ CRITICAL${NC}"
    STATUS_TEXT="System has critical issues"
fi

echo -e "Overall Health Score: $HEALTH_SCORE/100"
echo -e "Status: $STATUS"
echo "$STATUS_TEXT"

if [ ${#ISSUES[@]} -gt 0 ]; then
    echo ""
    echo "Issues Found:"
    for issue in "${ISSUES[@]}"; do
        echo "  • $issue"
    done
fi

echo ""
echo "Quick Actions:"
if [ "$DRIFTED_COUNT" -gt 0 ]; then
    echo "  • Fix drift: curl -s $API_ENDPOINT | jq -r '.corrections[].command'"
fi
if [ "$HEALTH_SCORE" -lt 90 ]; then
    echo "  • Review issues above and take corrective action"
fi

echo ""
echo "Verification Commands:"
echo "  • View units: cub unit list --space $SPACE"
echo "  • Check deployments: kubectl get deployments -n $NAMESPACE"
echo "  • API status: curl -s $API_ENDPOINT | jq '.timestamp'"
echo "  • Run compliance test: ./test-app-compliance-quick.sh"

exit $([ $HEALTH_SCORE -ge 70 ] && echo 0 || echo 1)