# Claude AI Setup for DevOps Examples

## Quick Start

All DevOps examples now use Claude AI by default for intelligent analysis. Follow these steps:

### 1. Get Your Claude API Key
1. Go to: https://console.anthropic.com/settings/keys
2. Create a new API key
3. Copy the key (starts with `sk-ant-`)

### 2. Set Up Environment
```bash
# Option A: Create .env file (recommended)
cp .env.example .env
# Edit .env and add your CLAUDE_API_KEY

# Option B: Export directly
export CLAUDE_API_KEY="sk-ant-your-key-here"
export CLAUDE_DEBUG_LOGGING=true  # To see all prompts/responses
```

### 3. Run Examples
```bash
# Using the run script (recommended - handles everything)
cd cost-optimizer
./run.sh

# Or run directly with environment
source ../setup-env.sh
./cost-optimizer
```

## Features When Claude is Enabled

### Cost Optimizer
- **Intelligent cost analysis** based on actual usage patterns
- **Risk assessment** for each recommendation (low/medium/high)
- **Specific optimization actions** with expected savings
- **Context-aware suggestions** based on your infrastructure

### Drift Detector
- **Root cause analysis** of configuration drift
- **Automated fix recommendations** with safety ratings
- **Pattern recognition** for recurring drift issues

## How to Disable Claude

### Method 1: Environment Variable (Temporary)
```bash
# Disable for one run
ENABLE_CLAUDE=false ./run.sh

# Or export for session
export ENABLE_CLAUDE=false
./cost-optimizer
```

### Method 2: Edit .env File (Persistent)
```bash
# In .env file, change:
ENABLE_CLAUDE=false
```

### Method 3: Comment Out in Code
```go
// In main.go, comment out Claude initialization:
// app.Claude = sdk.NewClaudeClient(os.Getenv("CLAUDE_API_KEY"))
```

## Debug Logging

Claude debug logging is **enabled by default** to show all prompts and responses.

### View Full Prompts and Responses
```bash
export CLAUDE_DEBUG_LOGGING=true  # Already default
./cost-optimizer
```

Output will show:
```
[Claude] req-1 ◀ FULL_PROMPT:
Analyze the following Kubernetes resource usage...

[Claude] req-1 ▶ FULL_RESPONSE:
{
  "total_monthly_cost": 922.00,
  "recommendations": [...]
}
```

### Disable Debug Logging
```bash
export CLAUDE_DEBUG_LOGGING=false
```

## What Claude Logs

When debug logging is enabled, you'll see:

1. **Request Counter**: `req-1`, `req-2`, etc.
2. **Full Prompts**: Complete text sent to Claude
3. **Full Responses**: Complete JSON/text returned
4. **Timing**: Duration of each API call
5. **Errors**: Any API failures with details

Example log output:
```
[Claude] 🔍 Debug logging enabled - all prompts and responses will be logged
[Claude] req-1 ◀ REQUEST: Analyze the following Kubernetes resource usage data...
[Claude] req-1 ◀ FULL_PROMPT:
Analyze the following Kubernetes resource usage data and provide cost optimization recommendations.
Focus on:
1. Resources with low utilization (<50%)...

[Claude] req-1 → Sending API request
[Claude] req-1 ▶ RESPONSE (1.2s): {"total_monthly_cost": 922.00, "potential_savings": 208.95...
[Claude] req-1 ▶ FULL_RESPONSE:
{
  "total_monthly_cost": 922.00,
  "potential_savings": 208.95,
  "savings_percentage": 22.8,
  "recommendations": [
    {
      "resource": "deployment/frontend-web",
      "priority": "high",
      "monthly_savings": 73.65,
      "risk": "low",
      "explanation": "Frontend is over-provisioned..."
    }
  ]
}
```

## ConfigHub Integration

ConfigHub is also enabled by default. Authentication happens automatically:
```bash
# Token is obtained automatically from cub CLI
# Or set manually:
export CUB_TOKEN=$(cub auth get-token)
```

## Standard Pattern for All Examples

Every example follows this pattern:

1. **run.sh script** - Handles all setup automatically
2. **Claude enabled by default** - With easy disable option
3. **Debug logging by default** - See all AI interactions
4. **ConfigHub integration** - Automatic authentication
5. **Fallback to basic** - Works without Claude if needed

## Troubleshooting

### "Claude API key not set"
```bash
# Check your key is exported
echo $CLAUDE_API_KEY

# Or add to .env file
echo "CLAUDE_API_KEY=sk-ant-your-key" >> .env
```

### "API error 401: Unauthorized"
Your API key is invalid. Get a new one from https://console.anthropic.com/settings/keys

### "Claude analysis failed"
Check debug logs for details:
```bash
export CLAUDE_DEBUG_LOGGING=true
./run.sh
```

## Cost Considerations

- Claude API calls cost ~$0.25 per 1M input tokens
- Each analysis typically uses ~2-5K tokens
- Cost optimizer runs analysis every 30 seconds
- Approximate cost: $0.01-0.02 per hour of operation

To reduce costs:
- Increase analysis interval in code
- Use ENABLE_CLAUDE=false for development
- Enable only for production monitoring