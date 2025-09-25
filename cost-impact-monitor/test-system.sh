#!/bin/bash

# System Test: Validate Cost Impact Monitor Robustness & Patterns
# Tests via API without requiring ConfigHub authentication

set -e

echo "========================================"
echo "Cost Impact Monitor System Test"
echo "========================================"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Simple test function
test_api() {
    local test_name=$1
    local jq_query=$2
    local expected=$3

    echo -n "Testing: $test_name ... "

    result=$(curl -s http://localhost:8082/api/live | jq -r "$jq_query" 2>/dev/null || echo "ERROR")

    if [[ "$result" == "$expected" ]] || [[ "$result" =~ $expected ]]; then
        echo -e "${GREEN}PASS${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}FAIL${NC} (got: $result, expected: $expected)"
        ((TESTS_FAILED++))
    fi
}

echo "1. API AVAILABILITY & ROBUSTNESS"
echo "================================="
echo ""

# Test API is responding
test_api "API responds" '.timestamp != null' "true"
test_api "Has cluster info" '.cluster_info.context' "kind-devops-test"
test_api "Has ConfigHub info" '.confighub_info.connected' "true"
test_api "Has resources" '.resources | length > 0' "true"

echo ""
echo "2. CONFIGHHUB PATTERN ADHERENCE"
echo "================================"
echo ""

# Test ConfigHub patterns via API
test_api "Shows ConfigHub units" '.confighub_info.units | length' "4"
test_api "Uses cub commands" '.corrections[0].command | contains("cub unit")' "true"
test_api "NO kubectl commands" '.corrections[0].command | contains("kubectl")' "false"
test_api "Shows drift-test-demo space" '.confighub_info.spaces[0]' "drift-test-demo"

echo ""
echo "3. DRIFT DETECTION ACCURACY"
echo "============================"
echo ""

# Test drift detection
test_api "Detects backend-api drift" '.resources[] | select(.name=="backend-api") | .is_drifted' "true"
test_api "Backend-api expected replicas" '.resources[] | select(.name=="backend-api") | .expected_replicas' "3"
test_api "Backend-api actual replicas" '.resources[] | select(.name=="backend-api") | .replicas' "5"
test_api "Has correction command" '.corrections | length > 0' "true"

echo ""
echo "4. COST CALCULATIONS"
echo "===================="
echo ""

# Test cost calculations
test_api "Total cost is positive" '(.total_monthly_cost > 0)' "true"
test_api "Drift cost calculated" '(.drift_cost != 0)' "true"
test_api "Has potential savings" '(.potential_savings > 0)' "true"

echo ""
echo "5. MODULARITY & COMPONENTS"
echo "=========================="
echo ""

# Test component structure
test_api "Claude analysis present" '.claude_analysis != null' "true"
test_api "Corrections have impact" '.corrections[0].impact != null' "true"
test_api "Resources have type" '.resources[0].type' "Deployment"

echo ""
echo "6. LOAD & CONSISTENCY TEST"
echo "==========================="
echo ""

# Rapid API calls to test stability
echo -n "Rapid requests (10x): "
failed=0
for i in {1..10}; do
    if curl -s http://localhost:8082/api/live > /dev/null 2>&1; then
        echo -n "."
    else
        echo -n "X"
        ((failed++))
    fi
done

if [ $failed -eq 0 ]; then
    echo -e " ${GREEN}PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e " ${RED}FAIL ($failed requests failed)${NC}"
    ((TESTS_FAILED++))
fi

echo ""
echo "7. CANONICAL PATTERN FILES"
echo "=========================="
echo ""

# Check for canonical pattern files
echo -n "Testing: Uses SDK ... "
if grep -q "github.com/monadic/devops-sdk" go.mod 2>/dev/null; then
    echo -e "${GREEN}PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}FAIL${NC}"
    ((TESTS_FAILED++))
fi

echo -n "Testing: No hallucinated APIs ... "
if ! grep -q "GetVariant\|CloneWithVariant\|UpgradeSet" *.go 2>/dev/null; then
    echo -e "${GREEN}PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}FAIL${NC}"
    ((TESTS_FAILED++))
fi

echo ""
echo "8. FAILURE SCENARIOS"
echo "===================="
echo ""

# Test handling of invalid requests
echo -n "Testing: Handles invalid endpoint ... "
if curl -s http://localhost:8082/invalid 2>/dev/null | grep -q "404"; then
    echo -e "${GREEN}PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}WARN (no 404 page)${NC}"
fi

echo ""
echo "========================================"
echo "TEST RESULTS"
echo "========================================"
echo -e "${GREEN}Tests Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Tests Failed: $TESTS_FAILED${NC}"

# Calculate percentage
TOTAL_TESTS=$((TESTS_PASSED + TESTS_FAILED))
if [ $TOTAL_TESTS -gt 0 ]; then
    PERCENTAGE=$((TESTS_PASSED * 100 / TOTAL_TESTS))
    echo "Success Rate: ${PERCENTAGE}%"
fi

echo ""
echo "KEY FINDINGS:"
echo "============="

# Check adherence to patterns
if curl -s http://localhost:8082/api/live | jq -r '.corrections[0].command' | grep -q "cub unit"; then
    echo -e "${GREEN}✓ Adheres to ConfigHub patterns (uses cub commands)${NC}"
else
    echo -e "${RED}✗ Not using ConfigHub patterns${NC}"
fi

if curl -s http://localhost:8082/api/live | jq '.confighub_info.connected' | grep -q "true"; then
    echo -e "${GREEN}✓ ConfigHub integration working${NC}"
else
    echo -e "${RED}✗ ConfigHub not connected${NC}"
fi

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}✓ All tests passed! System is robust and follows patterns.${NC}"
    exit 0
else
    echo -e "\n${YELLOW}⚠ Some tests failed. Review and fix issues.${NC}"
    exit 1
fi