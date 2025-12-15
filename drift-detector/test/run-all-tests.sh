#!/bin/bash
# Run all drift-detector tests
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0

echo "Running drift-detector tests..."
echo ""

# Unit tests (Go)
echo "=== Unit Tests (go test) ==="
cd "$PROJECT_DIR"
if go test -v ./... 2>&1; then
  echo -e "${GREEN}Unit tests passed${NC}"
  ((PASS++))
else
  echo -e "${RED}Unit tests failed${NC}"
  ((FAIL++))
fi

echo ""

# Integration tests (requires ConfigHub auth)
echo "=== Integration Tests ==="
if cub auth get-token >/dev/null 2>&1; then
  if go test -v -tags=integration ./... 2>&1; then
    echo -e "${GREEN}Integration tests passed${NC}"
    ((PASS++))
  else
    echo -e "${YELLOW}Integration tests failed (may need ConfigHub setup)${NC}"
    ((FAIL++))
  fi
else
  echo -e "${YELLOW}Skipping integration tests (not authenticated with ConfigHub)${NC}"
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
