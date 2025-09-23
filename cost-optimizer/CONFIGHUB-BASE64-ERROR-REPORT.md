# ConfigHub Unit Creation Base64 Encoding Error Report

## Issue Summary
**Date**: 2025-09-23
**Component**: cost-optimizer
**Severity**: Medium
**Status**: Active (workaround available)
**Impact**: Cost analysis data cannot be persisted to ConfigHub units

## Error Details

### Error Message
```
API error 400: {"message":"illegal base64 data at input byte 0"}
```

### Occurrence Pattern
- Happens every 30 seconds when cost-optimizer attempts to store analysis results
- Error occurs in `storeAnalysisInConfigHub()` function
- Affects both analysis units and recommendation units

### Root Cause
ConfigHub API requires the `Data` field in unit creation requests to be **base64-encoded**, but the cost-optimizer is sending **plain JSON strings**.

## Technical Analysis

### Current Implementation (Incorrect)
```go
// main.go:507-515
analysisData, err := json.MarshalIndent(analysis, "", "  ")
if err != nil {
    return fmt.Errorf("marshal analysis: %w", err)
}

_, err = c.app.Cub.CreateUnit(c.spaceID, sdk.CreateUnitRequest{
    Slug:        fmt.Sprintf("cost-analysis-%d", time.Now().Unix()),
    DisplayName: fmt.Sprintf("Cost Analysis %s", time.Now().Format("2006-01-02 15:04")),
    Data:        string(analysisData),  // ❌ Plain JSON string, not base64
    // ...
})
```

### Expected Format
ConfigHub expects:
- **Input**: Base64-encoded string
- **Example**: `eyJ0aW1lc3RhbXAiOiAiMjAyNS0wOS0yMyIsIC4uLn0=`
- **Not**: `{"timestamp": "2025-09-23", ...}`

## Fix Implementation

### Required Changes

1. **Add base64 encoding to main.go**:
```go
import "encoding/base64"

func (c *CostOptimizer) storeAnalysisInConfigHub(analysis *CostAnalysis) error {
    analysisData, err := json.MarshalIndent(analysis, "", "  ")
    if err != nil {
        return fmt.Errorf("marshal analysis: %w", err)
    }

    // Encode to base64
    encodedData := base64.StdEncoding.EncodeToString(analysisData)

    _, err = c.app.Cub.CreateUnit(c.spaceID, sdk.CreateUnitRequest{
        Slug:        fmt.Sprintf("cost-analysis-%d", time.Now().Unix()),
        DisplayName: fmt.Sprintf("Cost Analysis %s", time.Now().Format("2006-01-02 15:04")),
        Data:        encodedData,  // ✅ Now base64-encoded
        // ...
    })
}
```

2. **Update recommendation storage** (line 535):
```go
recData, _ := json.MarshalIndent(rec, "", "  ")
encodedRecData := base64.StdEncoding.EncodeToString(recData)

unit, err := c.app.Cub.CreateUnit(c.spaceID, sdk.CreateUnitRequest{
    // ...
    Data: encodedRecData,  // ✅ Base64-encoded
    // ...
})
```