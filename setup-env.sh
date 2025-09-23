#!/bin/bash

# DevOps as Apps - Environment Setup Script
# This script sets up API keys and credentials for all examples

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🚀 DevOps as Apps - Environment Setup"
echo "======================================"
echo ""

# Check if .env exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}⚠️  No .env file found. Creating from template...${NC}"
    cp .env.example .env
    echo -e "${GREEN}✅ Created .env file. Please edit it with your credentials.${NC}"
    echo ""
    echo "Required steps:"
    echo "1. Edit .env and add your CLAUDE_API_KEY"
    echo "2. Run this script again: source setup-env.sh"
    exit 1
fi

# Load environment variables from .env
export $(cat .env | grep -v '^#' | xargs)

# Get ConfigHub token automatically if not set
if [ -z "$CUB_TOKEN" ]; then
    echo -e "${YELLOW}📦 Getting ConfigHub token...${NC}"
    export CUB_TOKEN=$(cub auth get-token 2>/dev/null)
    if [ -n "$CUB_TOKEN" ]; then
        echo -e "${GREEN}✅ ConfigHub token obtained automatically${NC}"
        # Update .env file with the token
        sed -i.bak "s/CUB_TOKEN=.*/CUB_TOKEN=$CUB_TOKEN/" .env
    else
        echo -e "${RED}❌ Could not get ConfigHub token. Please run: cub auth login${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✅ ConfigHub token loaded from .env${NC}"
fi

# Check Claude API key
if [ "$ENABLE_CLAUDE" = "true" ]; then
    if [ -z "$CLAUDE_API_KEY" ] || [ "$CLAUDE_API_KEY" = "your-claude-api-key-here" ]; then
        echo -e "${RED}❌ CLAUDE_API_KEY not set in .env file${NC}"
        echo ""
        echo "To get your Claude API key:"
        echo "1. Go to: https://console.anthropic.com/settings/keys"
        echo "2. Create a new API key"
        echo "3. Add it to .env file: CLAUDE_API_KEY=sk-ant-..."
        echo ""
        echo "To disable Claude (use basic analysis):"
        echo "  Set ENABLE_CLAUDE=false in .env"
        exit 1
    else
        echo -e "${GREEN}✅ Claude API key loaded${NC}"
        if [ "$CLAUDE_DEBUG_LOGGING" = "true" ]; then
            echo -e "${GREEN}✅ Claude debug logging enabled${NC}"
        fi
    fi
else
    echo -e "${YELLOW}⚠️  Claude AI disabled (using basic analysis)${NC}"
    unset CLAUDE_API_KEY
fi

# Export all variables for child processes
export CUB_TOKEN
export CLAUDE_API_KEY
export ENABLE_CLAUDE
export CLAUDE_DEBUG_LOGGING
export ENABLE_CONFIGHUB

# Summary
echo ""
echo -e "${GREEN}✅ Environment ready!${NC}"
echo ""
echo "Active configuration:"
echo "  ConfigHub: ${ENABLE_CONFIGHUB:-true}"
echo "  Claude AI: ${ENABLE_CLAUDE:-true}"
echo "  Debug logging: ${CLAUDE_DEBUG_LOGGING:-false}"
echo ""
echo "To run examples:"
echo "  cd cost-optimizer && ./cost-optimizer"
echo "  cd drift-detector && ./drift-detector"
echo ""
echo "To disable Claude temporarily:"
echo "  export ENABLE_CLAUDE=false"
echo ""