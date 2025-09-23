#!/bin/bash

# Cost Optimizer Run Script - Automatically sets up Claude and ConfigHub
# This script ensures Claude AI and ConfigHub are always enabled by default

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Cost Optimizer with Claude AI${NC}"
echo "=================================="
echo ""

# Load environment from parent .env if exists
if [ -f ../.env ]; then
    echo -e "${GREEN}✅ Loading environment from ../.env${NC}"
    export $(cat ../.env | grep -v '^#' | xargs)
elif [ -f .env ]; then
    echo -e "${GREEN}✅ Loading environment from .env${NC}"
    export $(cat .env | grep -v '^#' | xargs)
fi

# Get ConfigHub token automatically if not set
if [ -z "$CUB_TOKEN" ]; then
    echo -e "${YELLOW}📦 Getting ConfigHub token...${NC}"
    export CUB_TOKEN=$(cub auth get-token 2>/dev/null)
    if [ -n "$CUB_TOKEN" ]; then
        echo -e "${GREEN}✅ ConfigHub authenticated${NC}"
    else
        echo -e "${RED}❌ Please authenticate: cub auth login${NC}"
        exit 1
    fi
fi

# Check if user wants to disable Claude (default is enabled)
if [ "$ENABLE_CLAUDE" = "false" ]; then
    echo -e "${YELLOW}⚠️  Claude AI disabled by user (using basic analysis)${NC}"
    unset CLAUDE_API_KEY
else
    # Claude is enabled by default
    if [ -z "$CLAUDE_API_KEY" ] || [ "$CLAUDE_API_KEY" = "your-claude-api-key-here" ]; then
        echo -e "${YELLOW}⚠️  No Claude API key found${NC}"
        echo ""
        echo "Please provide your Claude API key:"
        echo "  export CLAUDE_API_KEY=sk-ant-..."
        echo ""
        echo "Or to disable Claude (use basic analysis):"
        echo "  export ENABLE_CLAUDE=false"
        echo ""
        echo "To get a Claude API key:"
        echo "  1. Go to: https://console.anthropic.com/settings/keys"
        echo "  2. Create a new API key"
        echo ""
        read -p "Enter your Claude API key (or press Enter to skip): " input_key
        if [ -n "$input_key" ]; then
            export CLAUDE_API_KEY="$input_key"
            echo -e "${GREEN}✅ Claude API key set${NC}"
        else
            echo -e "${YELLOW}⚠️  Continuing without Claude (basic analysis only)${NC}"
        fi
    else
        echo -e "${GREEN}✅ Claude AI enabled${NC}"
    fi
fi

# Enable Claude debug logging by default (unless disabled)
if [ "$CLAUDE_DEBUG_LOGGING" != "false" ]; then
    export CLAUDE_DEBUG_LOGGING=true
    echo -e "${GREEN}✅ Claude debug logging enabled (prompts & responses will be shown)${NC}"
fi

# Use existing ConfigHub space if provided
if [ -n "$CONFIGHUB_SPACE_ID" ]; then
    echo -e "${GREEN}✅ Using ConfigHub space: $CONFIGHUB_SPACE_ID${NC}"
fi

# Build if needed
if [ ! -f cost-optimizer ] || [ main.go -nt cost-optimizer ]; then
    echo -e "${YELLOW}🔨 Building cost-optimizer...${NC}"
    go build -o cost-optimizer .
    echo -e "${GREEN}✅ Build complete${NC}"
fi

echo ""
echo "Starting cost-optimizer with:"
echo "  • ConfigHub: ✅ Enabled"
if [ -n "$CLAUDE_API_KEY" ]; then
    echo "  • Claude AI: ✅ Enabled (with debug logging)"
else
    echo "  • Claude AI: ❌ Disabled (using basic analysis)"
fi
echo "  • Dashboard: http://localhost:8081"
echo ""
echo "To disable Claude temporarily:"
echo "  ENABLE_CLAUDE=false ./run.sh"
echo ""

# Run the optimizer
exec ./cost-optimizer "$@"