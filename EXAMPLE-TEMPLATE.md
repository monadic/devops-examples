# DevOps Example Template - REQUIRED for All New Examples

## Directory Structure (MUST FOLLOW)
```
example-name/
├── .env.example           # Copy from parent, include CLAUDE_API_KEY
├── run.sh                 # REQUIRED: Auto-setup script (see below)
├── main.go                # Main application
├── go.mod                 # Dependencies
├── Dockerfile             # Container image
├── README.md              # Documentation
├── bin/                   # ConfigHub deployment scripts
│   ├── install-base       # Create ConfigHub structure
│   ├── install-envs       # Create environment hierarchy
│   ├── apply-all          # Deploy via ConfigHub
│   ├── promote            # Promote between environments
│   └── cleanup            # Remove all resources
└── confighub/
    └── base/              # Base K8s manifests
        ├── deployment.yaml
        ├── service.yaml
        ├── rbac.yaml
        └── configmap.yaml
```

## Required run.sh Script

Every example MUST include this run.sh script (customize app name):

```bash
#!/bin/bash

# [APP-NAME] Run Script - Automatically sets up Claude and ConfigHub
# This script ensures Claude AI and ConfigHub are always enabled by default

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 [APP-NAME] with Claude AI${NC}"
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
    echo -e "${YELLOW}⚠️  Claude AI disabled by user${NC}"
    unset CLAUDE_API_KEY
else
    # Claude is enabled by default
    if [ -z "$CLAUDE_API_KEY" ] || [ "$CLAUDE_API_KEY" = "your-claude-api-key-here" ]; then
        echo -e "${YELLOW}⚠️  No Claude API key found${NC}"
        echo ""
        echo "Please provide your Claude API key:"
        echo "  export CLAUDE_API_KEY=sk-ant-..."
        echo ""
        echo "Or to disable Claude:"
        echo "  export ENABLE_CLAUDE=false"
        echo ""
        read -p "Enter your Claude API key (or press Enter to skip): " input_key
        if [ -n "$input_key" ]; then
            export CLAUDE_API_KEY="$input_key"
            echo -e "${GREEN}✅ Claude API key set${NC}"
        else
            echo -e "${YELLOW}⚠️  Continuing without Claude${NC}"
        fi
    else
        echo -e "${GREEN}✅ Claude AI enabled${NC}"
    fi
fi

# Enable Claude debug logging by default
if [ "$CLAUDE_DEBUG_LOGGING" != "false" ]; then
    export CLAUDE_DEBUG_LOGGING=true
    echo -e "${GREEN}✅ Claude debug logging enabled${NC}"
fi

# Build if needed
if [ ! -f [app-name] ] || [ main.go -nt [app-name] ]; then
    echo -e "${YELLOW}🔨 Building [app-name]...${NC}"
    go build -o [app-name] .
    echo -e "${GREEN}✅ Build complete${NC}"
fi

echo ""
echo "Starting [app-name] with:"
echo "  • ConfigHub: ✅ Enabled"
if [ -n "$CLAUDE_API_KEY" ]; then
    echo "  • Claude AI: ✅ Enabled (with debug logging)"
else
    echo "  • Claude AI: ❌ Disabled"
fi
echo ""
echo "To disable Claude temporarily:"
echo "  ENABLE_CLAUDE=false ./run.sh"
echo ""

# Run the app
exec ./[app-name] "$@"
```

## Required main.go Structure

```go
package main

import (
    "log"
    "os"
    sdk "github.com/monadic/devops-sdk"
)

func main() {
    // Initialize SDK with Claude and ConfigHub
    app, err := sdk.NewDevOpsApp("example-name", "v1.0.0",
        "Description of what this app does")
    if err != nil {
        log.Fatalf("Failed to initialize: %v", err)
    }

    // Claude is initialized automatically if CLAUDE_API_KEY is set
    // Debug logging is enabled if CLAUDE_DEBUG_LOGGING=true

    // Check if Claude is available
    if app.Claude != nil {
        app.Logger.Println("✅ Claude AI enabled for intelligent analysis")

        // Enable debug logging if requested
        if os.Getenv("CLAUDE_DEBUG_LOGGING") == "true" {
            app.Claude.EnableDebugLogging()
        }
    } else {
        app.Logger.Println("⚠️  Running without Claude (basic mode)")
    }

    // ConfigHub is initialized automatically if CUB_TOKEN is set
    if app.Cub != nil {
        app.Logger.Println("✅ ConfigHub connected")
    }

    // Your app logic here
}
```

## Required Environment Variables

### In .env.example (MUST include):
```bash
# REQUIRED: Claude AI API Key (enabled by default)
CLAUDE_API_KEY=sk-ant-your-key-here

# REQUIRED: ConfigHub Token (auto-obtained from cub auth)
CUB_TOKEN=

# Feature Flags
ENABLE_CLAUDE=true           # Set to false to disable Claude
CLAUDE_DEBUG_LOGGING=true    # Set to false to disable debug logs
ENABLE_CONFIGHUB=true        # Set to false for local-only mode
```

## Required README.md Sections

Every example README must include:

```markdown
## Quick Start

```bash
# Option 1: Use the run script (recommended)
./run.sh

# Option 2: Set up environment first
source ../setup-env.sh
./example-name
```

## Claude AI Integration

This example uses Claude AI by default for intelligent analysis.

### What Claude Provides
- [Specific features for this example]
- [e.g., Root cause analysis, recommendations, etc.]

### To Disable Claude
```bash
# Temporarily
ENABLE_CLAUDE=false ./run.sh

# Permanently in .env
ENABLE_CLAUDE=false
```

### Debug Logging
Claude debug logging is enabled by default. You'll see:
- Full prompts sent to Claude
- Complete responses received
- Timing and request IDs

To disable: `export CLAUDE_DEBUG_LOGGING=false`
```

## Required ConfigHub Integration

### bin/install-base Must Include:
```bash
#!/bin/bash
# Generate unique prefix
prefix=$(cub space new-prefix)
project="${prefix}-example-name"
echo $project > .cub-project

# Create spaces
cub space create $project --label app=example-name
cub space create $project-base --label base=true
cub space create $project-filters --label type=filters

# Create sets and filters as needed
```

## Testing Requirements

### Every Example Must:
1. Run without Claude when ENABLE_CLAUDE=false
2. Show debug logs when CLAUDE_DEBUG_LOGGING=true
3. Auto-authenticate with ConfigHub
4. Include demo mode for testing without real infrastructure

### Test Script:
```bash
# Test with Claude disabled
ENABLE_CLAUDE=false ./run.sh

# Test with debug logging
CLAUDE_DEBUG_LOGGING=true ./run.sh

# Test demo mode
./example-name demo
```

## Documentation Requirements

### Must Document:
1. What Claude analyzes/provides
2. How to disable Claude (3 methods)
3. What logs are generated
4. ConfigHub resources created
5. Cost implications of Claude usage

## Checklist for New Examples

- [ ] run.sh script with Claude setup
- [ ] .env.example with CLAUDE_API_KEY
- [ ] Claude initialization in main.go
- [ ] Fallback when Claude is disabled
- [ ] Debug logging support
- [ ] README with Claude sections
- [ ] ConfigHub bin/ scripts
- [ ] Test without Claude
- [ ] Demo mode
- [ ] Cost documentation