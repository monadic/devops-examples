#!/bin/bash

# Quick DevOps App Compliance Check
# Based on recent learnings: ConfigHub-only commands are mandatory

set -e

echo "=========================================="
echo "DevOps App Quick Compliance Check"
echo "=========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

APP_DIR=${1:-.}
APP_NAME=$(basename "$APP_DIR")

echo "Checking: $APP_NAME"
echo ""

# Critical checks based on CLAUDE.md requirements
PASSED=0
FAILED=0

check() {
    local test_name=$1
    local command=$2
    local expected=$3

    echo -n "  $test_name ... "

    if eval "$command" > /dev/null 2>&1; then
        if [ "$expected" = "pass" ]; then
            echo -e "${GREEN}✓${NC}"
            ((PASSED++))
        else
            echo -e "${RED}✗${NC}"
            ((FAILED++))
        fi
    else
        if [ "$expected" = "fail" ]; then
            echo -e "${GREEN}✓${NC}"
            ((PASSED++))
        else
            echo -e "${RED}✗${NC}"
            ((FAILED++))
        fi
    fi
}

echo "CRITICAL REQUIREMENTS (from CLAUDE.md):"
echo "----------------------------------------"
check "NO kubectl in code" "! grep -r 'kubectl' $APP_DIR/*.go 2>/dev/null" "pass"
check "Uses cub unit commands" "grep -r 'cub unit' $APP_DIR/*.go 2>/dev/null" "pass"
check "Has ConfigHub client" "grep -r 'ConfigHub\\|confighub' $APP_DIR/*.go 2>/dev/null" "pass"

echo ""
echo "SDK & PATTERNS:"
echo "---------------"
check "Uses devops-sdk" "grep 'github.com/monadic/devops-sdk' $APP_DIR/go.mod 2>/dev/null" "pass"
check "Event-driven (informers)" "grep -r 'RunWithInformers\\|informer' $APP_DIR/*.go 2>/dev/null" "pass"
check "No polling loops" "! grep -r 'for.*{.*sleep\\|time.Sleep.*for' $APP_DIR/*.go 2>/dev/null" "pass"

echo ""
echo "SELF-DEPLOYMENT:"
echo "----------------"
check "Has bin/install-base" "[ -f $APP_DIR/bin/install-base ]" "pass"
check "Has bin/install-envs" "[ -f $APP_DIR/bin/install-envs ]" "pass"
check "Has bin/apply-all" "[ -f $APP_DIR/bin/apply-all ]" "pass"

echo ""
echo "CANONICAL PATTERNS:"
echo "-------------------"
check "Uses Sets" "grep -r 'CreateSet\\|SetID' $APP_DIR/*.go 2>/dev/null" "pass"
check "Uses Filters" "grep -r 'CreateFilter\\|Filter.*Where' $APP_DIR/*.go 2>/dev/null" "pass"
check "Push-upgrade" "grep -r 'BulkPatchUnits.*Upgrade\\|push-upgrade' $APP_DIR/*.go 2>/dev/null" "pass"

echo ""
echo "NO HALLUCINATIONS:"
echo "------------------"
check "No GetVariant" "! grep -r 'GetVariant' $APP_DIR/*.go 2>/dev/null" "pass"
check "No UpgradeSet" "! grep -r 'UpgradeSet' $APP_DIR/*.go 2>/dev/null" "pass"
check "No CloneWithVariant" "! grep -r 'CloneWithVariant' $APP_DIR/*.go 2>/dev/null" "pass"

echo ""
echo "CLAUDE AI:"
echo "----------"
check "Has Claude integration" "grep -r 'CLAUDE_API_KEY\\|claude' $APP_DIR/*.go 2>/dev/null" "pass"
check "Has ENABLE_CLAUDE flag" "grep -r 'ENABLE_CLAUDE' $APP_DIR/*.go 2>/dev/null" "pass"

echo ""
echo "API ENDPOINTS (if applicable):"
echo "-------------------------------"
if ls $APP_DIR/*.go 2>/dev/null | xargs grep -l "http.ListenAndServe\\|http.Serve" > /dev/null 2>&1; then
    # Check API endpoint for ConfigHub corrections
    PORT=$(grep -r "ListenAndServe.*:" $APP_DIR/*.go 2>/dev/null | head -1 | sed 's/.*:\([0-9]*\).*/\1/')
    if [ -n "$PORT" ]; then
        echo "  Found API on port $PORT"

        # Try to get corrections from API
        API_RESP=$(curl -s "http://localhost:$PORT/api/live" 2>/dev/null || echo "{}")
        if echo "$API_RESP" | jq '.corrections' > /dev/null 2>&1; then
            USES_CUB=$(echo "$API_RESP" | jq -r '.corrections[].command' 2>/dev/null | grep -c "cub unit" || echo "0")
            USES_KUBECTL=$(echo "$API_RESP" | jq -r '.corrections[].command' 2>/dev/null | grep -c "kubectl" || echo "0")

            if [ "$USES_KUBECTL" -eq 0 ] && [ "$USES_CUB" -gt 0 ]; then
                echo -e "  API corrections use cub ... ${GREEN}✓${NC}"
                ((PASSED++))
            elif [ "$USES_KUBECTL" -gt 0 ]; then
                echo -e "  API corrections use cub ... ${RED}✗ (uses kubectl!)${NC}"
                ((FAILED++))
            else
                echo -e "  API corrections use cub ... ${YELLOW}No corrections found${NC}"
            fi
        fi
    fi
else
    echo "  No HTTP API found"
fi

echo ""
echo "=========================================="
echo "COMPLIANCE SCORE"
echo "=========================================="
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"

TOTAL=$((PASSED + FAILED))
if [ $TOTAL -gt 0 ]; then
    SCORE=$((PASSED * 100 / TOTAL))
    echo "Score: ${SCORE}%"

    if [ $SCORE -eq 100 ]; then
        echo -e "\n${GREEN}✓ FULLY COMPLIANT${NC}"
        exit 0
    elif [ $SCORE -ge 80 ]; then
        echo -e "\n${YELLOW}⚠ MOSTLY COMPLIANT (${SCORE}%)${NC}"
        exit 0
    else
        echo -e "\n${RED}✗ NOT COMPLIANT (${SCORE}%)${NC}"
        exit 1
    fi
fi