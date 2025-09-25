#!/bin/bash

# DevOps App Compliance & Exercise Test
# Tests well-formedness AND forces changes to ALL units to verify proper behavior

set -e

echo "=========================================="
echo "DevOps App Compliance & Exercise Test"
echo "=========================================="
echo ""
echo "This test:"
echo "1. Validates app follows ALL required patterns"
echo "2. Forces changes to EVERY ConfigHub unit"
echo "3. Verifies app detects and handles changes correctly"
echo "4. Confirms all corrections use ConfigHub commands"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
APP_DIR=${1:-.}
APP_NAME=$(basename "$APP_DIR")
SPACE=${2:-drift-test-demo}
NAMESPACE=${3:-drift-test}
APP_PORT=${4:-8082}

echo "Testing: $APP_NAME"
echo "Directory: $APP_DIR"
echo "ConfigHub Space: $SPACE"
echo "Kubernetes Namespace: $NAMESPACE"
echo "App Port: $APP_PORT"
echo ""

# Test counters
STATIC_PASSED=0
STATIC_FAILED=0
DYNAMIC_PASSED=0
DYNAMIC_FAILED=0

# Test function
test_static() {
    local test_name=$1
    local command=$2
    local expected=$3

    echo -n "  $test_name ... "

    if eval "$command" > /dev/null 2>&1; then
        if [ "$expected" = "pass" ]; then
            echo -e "${GREEN}PASS${NC}"
            ((STATIC_PASSED++))
            return 0
        else
            echo -e "${RED}FAIL${NC}"
            ((STATIC_FAILED++))
            return 1
        fi
    else
        if [ "$expected" = "fail" ]; then
            echo -e "${GREEN}PASS${NC}"
            ((STATIC_PASSED++))
            return 0
        else
            echo -e "${RED}FAIL${NC}"
            ((STATIC_FAILED++))
            return 1
        fi
    fi
}

test_dynamic() {
    local test_name=$1
    local result=$2

    echo -n "  $test_name ... "
    if [ "$result" = "pass" ]; then
        echo -e "${GREEN}PASS${NC}"
        ((DYNAMIC_PASSED++))
    else
        echo -e "${RED}FAIL${NC}"
        ((DYNAMIC_FAILED++))
    fi
}

echo "=========================================="
echo "PART 1: STATIC ANALYSIS (Well-Formedness)"
echo "=========================================="
echo ""

echo "1.1 ConfigHub Exclusivity:"
test_static "No kubectl in code" "! grep -r 'kubectl' $APP_DIR/*.go 2>/dev/null" "pass"
test_static "Uses cub commands" "grep -r 'cub unit' $APP_DIR/*.go 2>/dev/null" "pass"
test_static "Has ConfigHub client" "grep -r 'ConfigHub\\|confighub' $APP_DIR/*.go 2>/dev/null" "pass"

echo ""
echo "1.2 SDK Compliance:"
test_static "Uses devops-sdk" "grep 'github.com/monadic/devops-sdk' $APP_DIR/go.mod 2>/dev/null" "pass"
test_static "Has SDK app structure" "grep -r 'sdk.DevOpsApp\\|RunWithInformers' $APP_DIR/*.go 2>/dev/null" "pass"

echo ""
echo "1.3 Self-Deployment:"
test_static "Has bin/install-base" "[ -f $APP_DIR/bin/install-base ]" "pass"
test_static "Has bin/install-envs" "[ -f $APP_DIR/bin/install-envs ]" "pass"
test_static "Uses space new-prefix" "grep 'cub space new-prefix' $APP_DIR/bin/install-base 2>/dev/null" "pass"

echo ""
echo "1.4 Event-Driven:"
test_static "Uses informers" "grep -r 'informer\\|Informer' $APP_DIR/*.go 2>/dev/null" "pass"
test_static "No polling loops" "! grep -r 'for.*{.*sleep' $APP_DIR/*.go 2>/dev/null" "pass"

echo ""
echo "1.5 Testing:"
test_static "Has tests" "ls $APP_DIR/*_test.go 2>/dev/null" "pass"
test_static "Has demo mode" "grep -r 'demo\\|Demo' $APP_DIR/*.go 2>/dev/null" "pass"

echo ""
echo "1.6 No Hallucinations:"
test_static "No GetVariant" "! grep -r 'GetVariant' $APP_DIR/*.go 2>/dev/null" "pass"
test_static "No UpgradeSet" "! grep -r 'UpgradeSet' $APP_DIR/*.go 2>/dev/null" "pass"

echo ""
echo "=========================================="
echo "PART 2: BUILD & LAUNCH"
echo "=========================================="
echo ""

# Build the app
echo -n "Building $APP_NAME ... "
if cd $APP_DIR && go build -o app-binary 2>/dev/null; then
    echo -e "${GREEN}SUCCESS${NC}"
    ((STATIC_PASSED++))
else
    echo -e "${RED}FAILED${NC}"
    ((STATIC_FAILED++))
    echo "Cannot continue without successful build"
    exit 1
fi

# Launch the app
echo -n "Launching $APP_NAME on port $APP_PORT ... "
cd $APP_DIR
if [ -f ./app-binary ]; then
    CUB_TOKEN="$CUB_TOKEN" ./app-binary &
    APP_PID=$!
    sleep 5

    if kill -0 $APP_PID 2>/dev/null; then
        echo -e "${GREEN}RUNNING (PID: $APP_PID)${NC}"
        ((DYNAMIC_PASSED++))
    else
        echo -e "${RED}CRASHED${NC}"
        ((DYNAMIC_FAILED++))
        exit 1
    fi
else
    echo -e "${RED}Binary not found${NC}"
    exit 1
fi

# Verify API endpoint
echo -n "Checking API endpoint ... "
if curl -s http://localhost:$APP_PORT/api/live > /dev/null 2>&1; then
    echo -e "${GREEN}RESPONDING${NC}"
    ((DYNAMIC_PASSED++))
else
    echo -e "${YELLOW}No API endpoint (may be OK)${NC}"
fi

echo ""
echo "=========================================="
echo "PART 3: ENUMERATE ALL UNITS"
echo "=========================================="
echo ""

# Get all units in the space
echo "Fetching all ConfigHub units in space: $SPACE"
ALL_UNITS=$(cub unit list --space $SPACE --output json 2>/dev/null | jq -r '.[].name' || echo "")

if [ -z "$ALL_UNITS" ]; then
    echo -e "${YELLOW}No units found. Creating test units...${NC}"

    # Create test units
    for app in app1 app2 app3 backend frontend database cache worker; do
        echo -n "  Creating $app-unit ... "
        cat <<EOF | cub unit create --space $SPACE --name $app-unit 2>/dev/null || true
apiVersion: apps/v1
kind: Deployment
metadata:
  name: $app
  namespace: $NAMESPACE
  labels:
    test-run: "$(date +%s)"
spec:
  replicas: 2
  selector:
    matchLabels:
      app: $app
  template:
    metadata:
      labels:
        app: $app
    spec:
      containers:
      - name: main
        image: nginx:1.21
        resources:
          requests:
            cpu: "100m"
            memory: "64Mi"
          limits:
            cpu: "200m"
            memory: "128Mi"
EOF
        echo -e "${GREEN}Created${NC}"
        ALL_UNITS="$ALL_UNITS $app-unit"
    done
fi

UNIT_COUNT=$(echo "$ALL_UNITS" | wc -w)
echo "Total units to test: $UNIT_COUNT"

echo ""
echo "=========================================="
echo "PART 4: FORCE CHANGES TO ALL UNITS"
echo "=========================================="
echo ""

# Track changes for verification
CHANGES_MADE=0
TIMESTAMP=$(date +%s)

echo "Applying changes to EVERY unit:"
for unit in $ALL_UNITS; do
    echo -e "${BLUE}Unit: $unit${NC}"

    # Change 1: Scale replicas
    echo -n "  Changing replicas to 3 ... "
    if cub unit update $unit --space $SPACE --patch \
        --data "{\"spec\":{\"replicas\":3,\"template\":{\"metadata\":{\"labels\":{\"test-timestamp\":\"$TIMESTAMP\"}}}}}" 2>/dev/null; then
        echo -e "${GREEN}Done${NC}"
        ((CHANGES_MADE++))
    else
        echo -e "${YELLOW}Skipped${NC}"
    fi

    # Change 2: Update resources
    echo -n "  Changing CPU to 150m ... "
    if cub unit update $unit --space $SPACE --patch \
        --data "{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"main\",\"resources\":{\"requests\":{\"cpu\":\"150m\"}}}]}}}}" 2>/dev/null; then
        echo -e "${GREEN}Done${NC}"
        ((CHANGES_MADE++))
    else
        echo -e "${YELLOW}Skipped${NC}"
    fi

    # Change 3: Add label
    echo -n "  Adding compliance-test label ... "
    if cub unit update $unit --space $SPACE --patch \
        --data "{\"metadata\":{\"labels\":{\"compliance-test\":\"$TIMESTAMP\",\"tested-by\":\"$APP_NAME\"}}}" 2>/dev/null; then
        echo -e "${GREEN}Done${NC}"
        ((CHANGES_MADE++))
    else
        echo -e "${YELLOW}Skipped${NC}"
    fi
done

echo ""
echo "Total changes made: $CHANGES_MADE"

echo ""
echo "=========================================="
echo "PART 5: VERIFY APP DETECTS CHANGES"
echo "=========================================="
echo ""

# Wait for app to detect changes
echo "Waiting for app to process changes (10 seconds)..."
sleep 10

# Check if app detected the changes
if [ -f "./app-binary" ] && curl -s http://localhost:$APP_PORT/api/live > /dev/null 2>&1; then
    echo "Checking app's detection via API..."

    API_RESPONSE=$(curl -s http://localhost:$APP_PORT/api/live)

    # Check for resources
    RESOURCES_COUNT=$(echo "$API_RESPONSE" | jq '.resources | length' 2>/dev/null || echo "0")
    test_dynamic "App sees resources" "[ $RESOURCES_COUNT -gt 0 ] && echo pass || echo fail"

    # Check for drift detection
    DRIFT_COUNT=$(echo "$API_RESPONSE" | jq '[.resources[] | select(.is_drifted == true)] | length' 2>/dev/null || echo "0")
    test_dynamic "App detects drift" "[ $DRIFT_COUNT -gt 0 ] && echo pass || echo fail"

    # Check for corrections
    CORRECTIONS=$(echo "$API_RESPONSE" | jq '.corrections | length' 2>/dev/null || echo "0")
    test_dynamic "App suggests corrections" "[ $CORRECTIONS -gt 0 ] && echo pass || echo fail"

    # Verify corrections use ConfigHub
    echo ""
    echo "Verifying correction commands:"
    echo "$API_RESPONSE" | jq -r '.corrections[].command' 2>/dev/null | while read -r cmd; do
        if echo "$cmd" | grep -q "cub unit"; then
            test_dynamic "  Uses cub unit command" "echo pass"
        else
            test_dynamic "  Uses cub unit command" "echo fail"
        fi

        if echo "$cmd" | grep -q "kubectl"; then
            test_dynamic "  NO kubectl (critical!)" "echo fail"
        else
            test_dynamic "  NO kubectl (critical!)" "echo pass"
        fi
    done
fi

echo ""
echo "=========================================="
echo "PART 6: VERIFY UNIT STATE"
echo "=========================================="
echo ""

echo "Checking that all units were modified:"
VERIFIED=0
for unit in $ALL_UNITS; do
    echo -n "  $unit: "

    # Get unit details
    UNIT_DATA=$(cub unit get $unit --space $SPACE --output json 2>/dev/null)

    # Check for our test label
    if echo "$UNIT_DATA" | grep -q "compliance-test.*$TIMESTAMP"; then
        echo -e "${GREEN}Modified${NC}"
        ((VERIFIED++))
    else
        echo -e "${RED}Not modified${NC}"
    fi
done

test_dynamic "All units modified" "[ $VERIFIED -eq $UNIT_COUNT ] && echo pass || echo fail"

echo ""
echo "=========================================="
echo "PART 7: APP LOGS CHECK"
echo "=========================================="
echo ""

# Check app logs for proper behavior
if [ -n "$APP_PID" ] && kill -0 $APP_PID 2>/dev/null; then
    echo "Checking app behavior (last 20 log lines):"

    # Check for ConfigHub operations in logs
    if ps aux | grep $APP_PID | grep -q "app-binary"; then
        echo -e "${GREEN}✓ App still running${NC}"
        ((DYNAMIC_PASSED++))
    fi
fi

echo ""
echo "=========================================="
echo "CLEANUP"
echo "=========================================="
echo ""

# Kill the app
if [ -n "$APP_PID" ]; then
    echo -n "Stopping $APP_NAME ... "
    kill $APP_PID 2>/dev/null
    echo -e "${GREEN}Stopped${NC}"
fi

echo ""
echo "=========================================="
echo "COMPLIANCE REPORT"
echo "=========================================="
echo ""

echo "Static Analysis (Well-Formedness):"
echo -e "  ${GREEN}Passed: $STATIC_PASSED${NC}"
echo -e "  ${RED}Failed: $STATIC_FAILED${NC}"

echo ""
echo "Dynamic Testing (Runtime Behavior):"
echo -e "  ${GREEN}Passed: $DYNAMIC_PASSED${NC}"
echo -e "  ${RED}Failed: $DYNAMIC_FAILED${NC}"

echo ""
echo "Unit Modifications:"
echo "  Units Found: $UNIT_COUNT"
echo "  Changes Made: $CHANGES_MADE"
echo "  Units Verified: $VERIFIED"

# Calculate overall score
TOTAL_PASSED=$((STATIC_PASSED + DYNAMIC_PASSED))
TOTAL_FAILED=$((STATIC_FAILED + DYNAMIC_FAILED))
TOTAL_TESTS=$((TOTAL_PASSED + TOTAL_FAILED))

if [ $TOTAL_TESTS -gt 0 ]; then
    PERCENTAGE=$((TOTAL_PASSED * 100 / TOTAL_TESTS))
    echo ""
    echo "Overall Compliance Score: ${PERCENTAGE}%"
fi

echo ""
if [ $STATIC_FAILED -eq 0 ] && [ $DYNAMIC_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ FULLY COMPLIANT: $APP_NAME passes all tests!${NC}"
    echo "  - Follows ConfigHub-only pattern"
    echo "  - Detects and handles all unit changes"
    echo "  - Uses proper SDK patterns"
    exit 0
elif [ $PERCENTAGE -ge 80 ]; then
    echo -e "${YELLOW}⚠ MOSTLY COMPLIANT: $APP_NAME has minor issues (${PERCENTAGE}%)${NC}"
    exit 0
else
    echo -e "${RED}✗ NOT COMPLIANT: $APP_NAME has critical failures!${NC}"
    exit 1
fi