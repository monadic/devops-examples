#!/bin/bash

# Test Script: Validate Canonical ConfigHub Patterns & Robustness
# Tests adherence to global-app patterns and ConfigHub best practices

set -e

echo "========================================"
echo "ConfigHub Canonical Pattern Test Suite"
echo "========================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Test function
run_test() {
    local test_name=$1
    local command=$2
    local expected_result=$3

    echo -n "Testing: $test_name ... "

    if eval "$command" > /dev/null 2>&1; then
        if [ "$expected_result" = "pass" ]; then
            echo -e "${GREEN}PASS${NC}"
            ((TESTS_PASSED++))
        else
            echo -e "${RED}FAIL (expected to fail but passed)${NC}"
            ((TESTS_FAILED++))
        fi
    else
        if [ "$expected_result" = "fail" ]; then
            echo -e "${GREEN}PASS (correctly failed)${NC}"
            ((TESTS_PASSED++))
        else
            echo -e "${RED}FAIL${NC}"
            ((TESTS_FAILED++))
        fi
    fi
}

echo "1. CONFIGHHUB PATTERN ADHERENCE TESTS"
echo "======================================"

# Test 1.1: Check for unique project prefix (canonical from global-app)
echo ""
echo "1.1 Testing Unique Project Prefix Pattern..."
run_test "Space has unique prefix" "cub space list | grep -E 'drift-test-demo|drift-detector-[0-9]+'" "pass"

# Test 1.2: Check for proper environment hierarchy
echo ""
echo "1.2 Testing Environment Hierarchy..."
run_test "Base environment exists" "cub space list | grep -E 'base|dev'" "pass"

# Test 1.3: Check for Sets and Filters (real ConfigHub features)
echo ""
echo "1.3 Testing Sets and Filters..."
run_test "Filters exist" "cub filter list | grep -E 'Unit|app|infra'" "pass"

# Test 1.4: Check for upstream/downstream relationships
echo ""
echo "1.4 Testing Upstream/Downstream..."
run_test "Units have upstream relationships" "cub unit list --space drift-test-demo | grep -E 'UPGRADE-NEEDED'" "fail"

# Test 1.5: Verify NO kubectl commands (ConfigHub-driven)
echo ""
echo "1.5 Testing ConfigHub-Driven Deployment..."
run_test "No direct kubectl in corrections" "curl -s http://localhost:8082/api/live | jq -r '.corrections[].command' | grep kubectl" "fail"
run_test "Uses cub commands for corrections" "curl -s http://localhost:8082/api/live | jq -r '.corrections[].command' | grep 'cub unit'" "pass"

echo ""
echo "2. GLOBAL-APP CANONICAL PATTERNS"
echo "================================="

# Test 2.1: Check bin/ scripts pattern
echo ""
echo "2.1 Testing Canonical Scripts..."
if [ -d "bin" ]; then
    run_test "Has bin/install-base" "[ -f bin/install-base ]" "pass"
    run_test "Has bin/install-envs" "[ -f bin/install-envs ]" "pass"
    run_test "Has bin/apply-all" "[ -f bin/apply-all ]" "pass"
else
    echo -e "${YELLOW}SKIP: bin/ directory not found (create for full compliance)${NC}"
fi

# Test 2.2: Check for push-upgrade pattern
echo ""
echo "2.2 Testing Push-Upgrade Pattern..."
run_test "Uses BulkPatchUnits with Upgrade" "grep -r 'BulkPatchUnits.*Upgrade.*true' *.go" "pass"

# Test 2.3: No hallucinated APIs
echo ""
echo "2.3 Testing for Hallucinated APIs..."
run_test "No GetVariant (hallucinated)" "grep -r 'GetVariant' *.go" "fail"
run_test "No CloneWithVariant (hallucinated)" "grep -r 'CloneWithVariant' *.go" "fail"
run_test "No UpgradeSet (hallucinated)" "grep -r 'UpgradeSet' *.go" "fail"

echo ""
echo "3. MODULARITY TESTS"
echo "==================="

# Test 3.1: SDK usage
echo ""
echo "3.1 Testing SDK Integration..."
run_test "Uses devops-sdk" "grep 'github.com/monadic/devops-sdk' go.mod" "pass"
run_test "Uses sdk.DevOpsApp pattern" "grep 'sdk.DevOpsApp' *.go" "pass"

# Test 3.2: Component isolation
echo ""
echo "3.2 Testing Component Isolation..."
run_test "Dashboard on separate port" "curl -s http://localhost:8082/ > /dev/null" "pass"
run_test "API endpoint available" "curl -s http://localhost:8082/api/live | jq '.timestamp' > /dev/null" "pass"

echo ""
echo "4. ROBUSTNESS TESTS"
echo "==================="

# Test 4.1: API availability
echo ""
echo "4.1 Testing API Robustness..."
for i in {1..5}; do
    run_test "API request $i" "curl -s http://localhost:8082/api/live | jq '.total_monthly_cost' > /dev/null" "pass"
    sleep 1
done

# Test 4.2: Drift detection accuracy
echo ""
echo "4.2 Testing Drift Detection..."
DRIFT_COUNT=$(curl -s http://localhost:8082/api/live | jq '[.resources[] | select(.is_drifted == true)] | length')
if [ "$DRIFT_COUNT" -gt 0 ]; then
    echo -e "${GREEN}PASS: Detected $DRIFT_COUNT drifted resources${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}INFO: No drift currently detected${NC}"
fi

# Test 4.3: Cost calculation consistency
echo ""
echo "4.3 Testing Cost Calculations..."
COST1=$(curl -s http://localhost:8082/api/live | jq '.total_monthly_cost')
sleep 2
COST2=$(curl -s http://localhost:8082/api/live | jq '.total_monthly_cost')
if [ "$COST1" = "$COST2" ]; then
    echo -e "${GREEN}PASS: Cost calculations are consistent${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}INFO: Cost changed (may indicate live changes)${NC}"
fi

# Test 4.4: ConfigHub connection
echo ""
echo "4.4 Testing ConfigHub Connection..."
run_test "ConfigHub connected" "curl -s http://localhost:8082/api/live | jq -r '.confighub_info.connected'" "pass"
run_test "ConfigHub units listed" "curl -s http://localhost:8082/api/live | jq '.confighub_info.units | length > 0'" "pass"

# Test 4.5: Kubernetes connection
echo ""
echo "4.5 Testing Kubernetes Connection..."
run_test "Cluster context available" "curl -s http://localhost:8082/api/live | jq -r '.cluster_info.context' | grep -E 'kind|eks|gke'" "pass"
run_test "Resources being monitored" "curl -s http://localhost:8082/api/live | jq '.resources | length > 0'" "pass"

echo ""
echo "5. DEVOPS-AS-APPS PHILOSOPHY"
echo "============================="

# Test 5.1: Persistent vs Ephemeral
echo ""
echo "5.1 Testing Persistent App Pattern..."
DASHBOARD_PID=$(ps aux | grep "live-dashboard" | grep -v grep | wc -l)
if [ "$DASHBOARD_PID" -gt 0 ]; then
    echo -e "${GREEN}PASS: App is persistent (not ephemeral workflow)${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}FAIL: App not running persistently${NC}"
    ((TESTS_FAILED++))
fi

# Test 5.2: Event-driven architecture
echo ""
echo "5.2 Testing Event-Driven Pattern..."
run_test "Uses informers (not polling)" "grep -r 'informer\|Informer' ../drift-detector/*.go" "pass"

# Test 5.3: Self-deployment through ConfigHub
echo ""
echo "5.3 Testing Self-Deployment..."
echo -e "${YELLOW}INFO: Apps should deploy themselves via ConfigHub units, not kubectl${NC}"
echo "      Proper pattern: bin/install-base -> bin/apply-all"

echo ""
echo "========================================"
echo "TEST RESULTS"
echo "========================================"
echo -e "${GREEN}Tests Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Tests Failed: $TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}✓ All tests passed! System adheres to canonical patterns.${NC}"
    exit 0
else
    echo -e "\n${RED}✗ Some tests failed. Review ConfigHub patterns.${NC}"
    exit 1
fi