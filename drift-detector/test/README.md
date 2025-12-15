# Drift Detector Tests

## Test Types

### Unit Tests
Go tests that run without external dependencies:
```bash
go test -v ./...
```

Files: `main_test.go`, `k8s_test.go`

### Integration Tests
Tests that require ConfigHub authentication:
```bash
# First authenticate
cub auth login

# Run integration tests
go test -v -tags=integration ./...
```

File: `integration_test.go`

### CLI Validation
Validates all cub commands in bin/ scripts:
```bash
/Users/alexis/Public/github-repos/devops-sdk/cub-command-analyzer.sh bin/
```

## Running All Tests

```bash
./test/run-all-tests.sh
```

## Test Requirements

- Go 1.21+
- ConfigHub CLI (`cub`) for integration tests
- ConfigHub authentication for integration tests
- Kubernetes cluster (kind) for full e2e testing

## Mandatory Testing Protocols

This project follows the DevOps-as-Apps testing standards defined in:

**CLAUDE.md** (`devops-as-apps-project/CLAUDE.md`)

Key sections:
- **MANDATORY Testing Protocol** - Mini TCK, cub-command-analyzer, config validation
- **Before Committing (BLOCKING)** - Pre-commit checklist
- **ConfigHub CLI Mental Model** - Command patterns and validation

Location: https://github.com/monadic/devops-as-apps-project/blob/main/CLAUDE.md

### Quick Reference

```bash
# Run Mini TCK first
cd /Users/alexis/Public/github-repos/devops-sdk && ./test-confighub-k8s

# Validate CLI commands
/Users/alexis/Public/github-repos/devops-sdk/cub-command-analyzer.sh bin/

# Run all tests
./test/run-all-tests.sh
```
