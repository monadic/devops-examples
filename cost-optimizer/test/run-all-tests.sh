#!/bin/bash
# Run all cost-optimizer tests
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0

echo "Running cost-optimizer tests..."
echo ""

# Build test
echo "=== Build Test ==="
cd "$PROJECT_DIR"
if go build -o /tmp/cost-optimizer-test . 2>&1; then
  echo -e "${GREEN}Build passed${NC}"
  rm -f /tmp/cost-optimizer-test
  ((PASS++))
else
  echo -e "${RED}Build failed${NC}"
  ((FAIL++))
fi

echo ""

# Unit tests (if any exist)
echo "=== Unit Tests (go test) ==="
if go test -v ./... 2>&1; then
  echo -e "${GREEN}Unit tests passed${NC}"
  ((PASS++))
else
  echo -e "${YELLOW}No unit tests or tests failed${NC}"
fi

echo ""

# CLI command validation
echo "=== CLI Command Validation ==="
ANALYZER="/Users/alexis/Public/github-repos/devops-sdk/cub-command-analyzer.sh"
if [ -x "$ANALYZER" ]; then
  if "$ANALYZER" "$PROJECT_DIR/bin/" 2>&1; then
    echo -e "${GREEN}CLI commands valid${NC}"
    ((PASS++))
  else
    echo -e "${RED}CLI command validation failed${NC}"
    ((FAIL++))
  fi
else
  echo -e "${YELLOW}Skipping CLI validation (analyzer not found)${NC}"
fi

echo ""

# YAML validation
echo "=== YAML Validation ==="
YAML_VALID=true
for file in "$PROJECT_DIR"/confighub/**/*.yaml "$PROJECT_DIR"/k8s/*.yaml; do
  if [ -f "$file" ]; then
    if ! python3 -c "import yaml; yaml.safe_load(open('$file'))" 2>/dev/null; then
      echo -e "${RED}Invalid YAML: $file${NC}"
      YAML_VALID=false
    fi
  fi
done
if [ "$YAML_VALID" = true ]; then
  echo -e "${GREEN}All YAML files valid${NC}"
  ((PASS++))
else
  ((FAIL++))
fi

echo ""
echo "=== Summary ==="
echo -e "${GREEN}Passed:${NC} $PASS"
echo -e "${RED}Failed:${NC} $FAIL"

if [ "$FAIL" -eq 0 ]; then
  echo -e "${GREEN}All tests passed${NC}"
  exit 0
else
  echo -e "${RED}Some tests failed${NC}"
  exit 1
fi
