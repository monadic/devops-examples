# Cost Impact Monitor Tests

## Test Types

### Build Test
Verifies the project compiles:
```bash
go build .
```

### Unit Tests
Go tests (add `*_test.go` files as needed):
```bash
go test -v ./...
```

### CLI Validation
Validates all cub commands in bin/ scripts:
```bash
/Users/alexis/Public/github-repos/devops-sdk/cub-command-analyzer.sh bin/
```

### YAML Validation
Validates all Kubernetes manifests:
```bash
python3 -c "import yaml; yaml.safe_load(open('file.yaml'))"
```

## Running All Tests

```bash
./test/run-all-tests.sh
```

## Test Requirements

- Go 1.21+
- Python 3 (for YAML validation)
- ConfigHub CLI (`cub`) for CLI validation

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
