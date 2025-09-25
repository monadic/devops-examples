#!/bin/bash

# Build all DevOps applications
set -e

echo "========================================="
echo "Building All DevOps Applications"
echo "========================================="

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Build drift-detector
echo "Building drift-detector..."
cd drift-detector
if go build -o drift-detector .; then
    echo -e "${GREEN}✅ drift-detector built${NC}"
else
    echo -e "${RED}❌ drift-detector build failed${NC}"
    exit 1
fi
cd ..

# Build cost-optimizer
echo "Building cost-optimizer..."
cd cost-optimizer
if go build -o cost-optimizer .; then
    echo -e "${GREEN}✅ cost-optimizer built${NC}"
else
    echo -e "${RED}❌ cost-optimizer build failed${NC}"
    exit 1
fi
cd ..

# Build cost-impact-monitor (live dashboard)
echo "Building live-dashboard..."
cd cost-impact-monitor
if go build -o live-dashboard live-dashboard.go; then
    echo -e "${GREEN}✅ live-dashboard built${NC}"
else
    echo -e "${RED}❌ live-dashboard build failed${NC}"
    exit 1
fi
cd ..

echo ""
echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}All applications built successfully!${NC}"
echo -e "${GREEN}=========================================${NC}"
echo ""
echo "Next steps:"
echo "  1. Start the dashboard: cd cost-impact-monitor && ./live-dashboard"
echo "  2. Access at: http://localhost:8082"
echo "  3. Run health check: curl http://localhost:8082/api/health | jq '.'"