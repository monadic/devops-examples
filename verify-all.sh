#!/bin/bash

# Complete verification of DevOps as Apps system
set -e

echo "========================================="
echo "DevOps Apps System Verification"
echo "========================================="
echo "Date: $(date)"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Track overall status
OVERALL_STATUS="PASS"

# Function to check command
check_command() {
    local cmd=$1
    local name=$2
    if command -v $cmd &> /dev/null; then
        echo -e "${GREEN}✅ $name found${NC}"
        return 0
    else
        echo -e "${RED}❌ $name not found - install with: $3${NC}"
        OVERALL_STATUS="FAIL"
        return 1
    fi
}

# Function to check service
check_service() {
    local port=$1
    local name=$2
    if curl -s http://localhost:$port > /dev/null 2>&1; then
        echo -e "${GREEN}✅ $name running on port $port${NC}"
        return 0
    else
        echo -e "${YELLOW}⚠️  $name not running on port $port${NC}"
        return 1
    fi
}

echo "1. PREREQUISITES CHECK"
echo "----------------------"
check_command "go" "Go" "brew install go"
check_command "kubectl" "Kubectl" "brew install kubectl"
check_command "kind" "Kind" "brew install kind"
check_command "cub" "ConfigHub CLI" "get from confighub.com"
check_command "jq" "jq" "brew install jq"

echo ""
echo "2. ENVIRONMENT CHECK"
echo "--------------------"
if [ -n "$CUB_TOKEN" ]; then
    echo -e "${GREEN}✅ CUB_TOKEN set${NC}"
    if cub auth whoami > /dev/null 2>&1; then
        echo -e "${GREEN}✅ ConfigHub authenticated${NC}"
    else
        echo -e "${RED}❌ ConfigHub authentication failed${NC}"
        OVERALL_STATUS="FAIL"
    fi
else
    echo -e "${RED}❌ CUB_TOKEN not set${NC}"
    OVERALL_STATUS="FAIL"
fi

if [ -n "$CLAUDE_API_KEY" ]; then
    echo -e "${GREEN}✅ Claude API key set${NC}"
else
    echo -e "${YELLOW}⚠️  Claude API key not set (optional but recommended)${NC}"
fi

echo ""
echo "3. KUBERNETES CHECK"
echo "-------------------"
if kubectl cluster-info > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Kubernetes cluster accessible${NC}"

    # Check for namespace
    if kubectl get namespace drift-test > /dev/null 2>&1; then
        echo -e "${GREEN}✅ drift-test namespace exists${NC}"
    else
        echo -e "${YELLOW}⚠️  drift-test namespace not found - creating...${NC}"
        kubectl create namespace drift-test
    fi
else
    echo -e "${RED}❌ No Kubernetes cluster found${NC}"
    echo "   Run: kind create cluster --name devops-test"
    OVERALL_STATUS="FAIL"
fi

echo ""
echo "4. BUILD STATUS"
echo "---------------"
# Check if binaries exist
if [ -f "drift-detector/drift-detector" ]; then
    echo -e "${GREEN}✅ drift-detector built${NC}"
else
    echo -e "${YELLOW}⚠️  drift-detector not built${NC}"
fi

if [ -f "cost-optimizer/cost-optimizer" ]; then
    echo -e "${GREEN}✅ cost-optimizer built${NC}"
else
    echo -e "${YELLOW}⚠️  cost-optimizer not built${NC}"
fi

if [ -f "cost-impact-monitor/live-dashboard" ]; then
    echo -e "${GREEN}✅ live-dashboard built${NC}"
else
    echo -e "${YELLOW}⚠️  live-dashboard not built${NC}"
fi

echo ""
echo "5. SERVICE STATUS"
echo "-----------------"
check_service 8082 "Live Dashboard"
check_service 8081 "Cost Optimizer"

echo ""
echo "6. API HEALTH CHECK"
echo "-------------------"
if curl -s http://localhost:8082/api/health > /dev/null 2>&1; then
    HEALTH_SCORE=$(curl -s http://localhost:8082/api/health | jq '.health_score')
    STATUS=$(curl -s http://localhost:8082/api/health | jq -r '.status')

    if [ "$HEALTH_SCORE" -ge 90 ]; then
        echo -e "${GREEN}✅ Health Check: $STATUS (Score: $HEALTH_SCORE/100)${NC}"
    elif [ "$HEALTH_SCORE" -ge 70 ]; then
        echo -e "${YELLOW}⚠️  Health Check: $STATUS (Score: $HEALTH_SCORE/100)${NC}"
    else
        echo -e "${RED}❌ Health Check: $STATUS (Score: $HEALTH_SCORE/100)${NC}"
    fi

    # Show any issues
    ISSUES=$(curl -s http://localhost:8082/api/health | jq -r '.issues[]' 2>/dev/null)
    if [ -n "$ISSUES" ]; then
        echo "   Issues found:"
        echo "$ISSUES" | while read issue; do
            echo "     • $issue"
        done
    fi
else
    echo -e "${YELLOW}⚠️  Health API not accessible${NC}"
fi

echo ""
echo "7. CONFIGHHUB COMPLIANCE"
echo "------------------------"
# Quick compliance check
if [ -f "test-app-compliance-quick.sh" ]; then
    echo "Running compliance check..."
    if ./test-app-compliance-quick.sh drift-detector > /dev/null 2>&1; then
        echo -e "${GREEN}✅ ConfigHub compliance: PASS${NC}"
    else
        echo -e "${YELLOW}⚠️  ConfigHub compliance: Check needed${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Compliance test script not found${NC}"
fi

echo ""
echo "========================================="
echo "VERIFICATION SUMMARY"
echo "========================================="

if [ "$OVERALL_STATUS" = "PASS" ]; then
    echo -e "${GREEN}✅ SYSTEM READY${NC}"
    echo ""
    echo "Dashboard URL: http://localhost:8082"
    echo "Health API: http://localhost:8082/api/health"
    echo ""
    echo "Quick commands:"
    echo "  • View dashboard: open http://localhost:8082"
    echo "  • Check health: curl http://localhost:8082/api/health | jq '.'"
    echo "  • Run compliance: ./test-app-compliance-quick.sh"
else
    echo -e "${RED}❌ SYSTEM NOT READY${NC}"
    echo ""
    echo "Fix the issues above, then:"
    echo "  1. Run: ./build-all.sh"
    echo "  2. Start dashboard: cd cost-impact-monitor && ./live-dashboard &"
    echo "  3. Verify again: ./verify-all.sh"
fi

exit $([ "$OVERALL_STATUS" = "PASS" ] && echo 0 || echo 1)