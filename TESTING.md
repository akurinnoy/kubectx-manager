# Testing Guide for kubectx-manager

This document provides comprehensive information about the testing strategy and coverage for kubectx-manager.

## Test Coverage Summary

kubectx-manager has comprehensive test coverage across all components:

- **Overall Coverage**: 77.7%
- **Logger Package**: 100% coverage
- **Config Package**: 88.9% coverage
- **Kubeconfig Package**: 91.5% coverage  
- **CMD Package**: 67.1% coverage

## Test Structure

### Unit Tests

#### 1. **Config Package Tests** (`internal/config/config_test.go`)

- ✅ Configuration file loading and parsing
- ✅ Pattern matching with glob patterns (`*`, `?`)
- ✅ Comment and empty line handling
- ✅ Default configuration creation
- ✅ Error handling for invalid files and permissions
- ✅ Regex compilation and pattern validation

#### 2. **Kubeconfig Package Tests** (`internal/kubeconfig/kubeconfig_test.go`)

- ✅ YAML parsing and structure validation
- ✅ Context, cluster, and user management
- ✅ Context removal and orphaned entry cleanup
- ✅ Backup creation and file operations
- ✅ Authentication validation (token, cert, basic, exec, auth-provider)
- ✅ Internal map building and lookup functions
- ✅ File save/load operations

#### 3. **Logger Package Tests** (`internal/logger/logger_test.go`)

- ✅ All log levels (Debug, Info, Warn, Error)
- ✅ Verbose and quiet mode combinations
- ✅ Output redirection (stdout vs stderr)
- ✅ Message formatting and prefixes
- ✅ Comprehensive behavior matrix testing

#### 4. **CLI Command Tests** (`cmd/root_test.go`, `cmd/restore_test.go`)

- ✅ Command flag initialization and parsing
- ✅ Context filtering logic
- ✅ Interactive confirmation prompts
- ✅ Backup finding and selection
- ✅ Restore functionality
- ✅ Input validation and error handling

### Integration Tests

#### 5. **End-to-End Tests** (`integration_test.go`)

- ✅ **Dry-run mode**: Complete workflow without modifications
- ✅ **Actual cleanup**: Real context removal with verification
- ✅ **Restore operations**: Backup creation and restoration
- ✅ **Authentication checking**: Auth status validation
- ✅ **Output modes**: Quiet, verbose, and normal mode testing

## Running Tests

### Quick Test Commands

```bash
# Run all unit tests
make test-unit

# Run all tests (unit + integration)
make test

# Run with coverage report
make test-coverage

# Generate HTML coverage report
make test-coverage-html

# Run with race detection
make test-race

# Run integration tests only
make test-integration
```

### Manual Test Commands

```bash
# Unit tests only (skip integration)
go test -v -short ./...

# Integration tests only
go test -v -run=TestIntegration ./...

# Coverage with HTML report
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Race detection
go test -v -race ./...

# Verbose coverage by function
go test -v -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Test Categories

### 1. **Functional Tests**

- Configuration loading and pattern matching
- Kubeconfig parsing and manipulation
- Context filtering and removal logic
- Backup and restore operations

### 2. **Error Handling Tests**

- Invalid configuration files
- Malformed YAML kubeconfig files
- Missing files and permission errors
- Invalid user input handling

### 3. **Edge Case Tests**

- Empty configuration files
- Kubeconfig files with no contexts
- Authentication validation edge cases
- Pattern matching special characters

### 4. **Integration Tests**

- Complete workflow testing
- Multi-step operations (backup → modify → restore)
- CLI flag combinations
- Real file system operations

## Test Data

Tests use temporary directories and files to ensure:

- ✅ **Isolation**: Each test runs independently
- ✅ **Cleanup**: No artifacts left after test completion
- ✅ **Reproducibility**: Consistent test environments

### Sample Test Configurations

```yaml
# Test Kubeconfig
apiVersion: v1
kind: Config
current-context: production-cluster
contexts:
- name: production-cluster
  context: {cluster: prod, user: admin}
- name: development-cluster
  context: {cluster: dev, user: dev-user}
clusters:
- name: prod
  cluster: {server: "https://prod.example.com"}
users:
- name: admin
  user: {token: "prod-token"}
```

```bash
# Test Configuration File
production-*
staging-cluster
*-important
```

## Continuous Integration

### GitHub Actions Workflow

- ✅ **Multi-version testing**: Go 1.21 and 1.22
- ✅ **Code quality**: golangci-lint, go vet, go fmt
- ✅ **Dependency verification**: go mod tidy, go mod verify
- ✅ **Comprehensive testing**: Unit + integration tests
- ✅ **Coverage reporting**: Codecov integration
- ✅ **Cross-compilation**: Multiple OS/architecture builds

### Coverage Thresholds

- **Minimum Overall Coverage**: 75%
- **Critical Packages**: >85% coverage
  - Config package: 88.9% ✅
  - Kubeconfig package: 91.5% ✅
  - Logger package: 100% ✅

## Test Development Guidelines

### Adding New Tests

1. **Unit Tests**: Add to appropriate `*_test.go` file
2. **Integration Tests**: Add to `integration_test.go`
3. **Table-Driven Tests**: Use for multiple test cases
4. **Error Cases**: Always test error conditions
5. **Edge Cases**: Test boundary conditions

### Test Naming Convention

```go
func TestFunctionName(t *testing.T)           // Basic test
func TestFunctionNameErrorCase(t *testing.T)  // Error testing
func TestFunctionNameEdgeCase(t *testing.T)   // Edge case testing
```

### Test Structure

```go
func TestExample(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected OutputType
        wantErr  bool
    }{
        // Test cases here
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Benchmarking

```bash
# Run benchmarks
make bench

# or manually
go test -bench=. -benchmem ./...
```

## Test Utilities

The testing framework includes utilities for:

- ✅ **Temporary file/directory creation**
- ✅ **Mock stdin/stdout/stderr**
- ✅ **Configuration file generation**
- ✅ **Kubeconfig file creation**
- ✅ **Process execution testing**

## Test Coverage Analysis

### High Coverage Areas (>90%)

- Logger functionality (100%)
- Kubeconfig operations (91.5%)
- Pattern matching (part of config 88.9%)

### Medium Coverage Areas (70-90%)

- Configuration loading (88.9%)
- CLI command handling (67.1%)

### Improvement Opportunities

- CLI error handling paths
- Complex integration scenarios
- Concurrent operation testing

## Debugging Tests

```bash
# Run specific test with verbose output
go test -v -run=TestSpecificFunction ./package

# Debug with delve
dlv test github.com/che-incubator/kubectx-manager/internal/config

# Print coverage details
go test -v -coverprofile=profile.out ./...
go tool cover -func=profile.out | sort -k 3 -nr
```

## Security Testing

Tests include security considerations:

- ✅ **File permission validation**
- ✅ **Input sanitization testing**
- ✅ **Backup file security**
- ✅ **Configuration file validation**

The comprehensive test suite ensures kubectx-manager is reliable, secure, and maintains high code quality standards.
