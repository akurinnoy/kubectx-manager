# kubectx-manager Makefile

# Variables
BINARY_NAME=kubectx-manager
MAIN_PACKAGE=.
BUILD_DIR=build
COVERAGE_DIR=coverage
LDFLAGS=-ldflags="-s -w"

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build         Build the binary"
	@echo "  test          Run all tests"
	@echo "  test-unit     Run unit tests only"
	@echo "  test-integration Run integration tests only"
	@echo "  test-coverage Generate test coverage report"
	@echo "  test-coverage-html Generate HTML test coverage report"
	@echo "  test-race     Run tests with race detection"
	@echo "  test-ci       Run tests exactly like CI (with race detection and coverage)"
	@echo "  clean         Clean build artifacts"
	@echo "  install       Install binary to $$GOPATH/bin"
	@echo "  lint          Run golangci-lint"
	@echo "  lint-fix      Run golangci-lint with auto-fix"
	@echo "  security      Run gosec security scanner"
	@echo "  security-sarif Run gosec and generate SARIF report"
	@echo "  format        Format code with gofmt"
	@echo "  vet           Run go vet"
	@echo "  mod-tidy      Run go mod tidy"
	@echo "  mod-verify    Verify go mod dependencies"
	@echo "  ci            Run all CI checks (format, vet, lint, test)"
	@echo "  ci-full       Run complete CI pipeline locally (matches GitHub workflow)"

# Build targets
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

.PHONY: install
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) $(MAIN_PACKAGE)

# Test targets
.PHONY: test
test:
	@echo "Running all tests..."
	go test -v ./...

.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	go test -v -short ./...

.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	go test -v -run=TestIntegration ./...

.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	go test -v -race ./...

.PHONY: test-ci
test-ci:
	@echo "Running tests exactly like CI (with race detection and coverage)..."
	@mkdir -p $(COVERAGE_DIR)
	go test -v -short -race -coverprofile=$(COVERAGE_DIR)/coverage.out ./...

.PHONY: test-coverage
test-coverage:
	@echo "Generating test coverage..."
	@mkdir -p $(COVERAGE_DIR)
	go test -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	go tool cover -func=$(COVERAGE_DIR)/coverage.out

.PHONY: test-coverage-html
test-coverage-html: test-coverage
	@echo "Generating HTML coverage report..."
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated: $(COVERAGE_DIR)/coverage.html"

.PHONY: bench
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Code quality targets
.PHONY: format
format:
	@echo "Formatting code..."
	gofmt -s -w .
	goimports -w .

.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...

.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	golangci-lint run

.PHONY: lint-fix
lint-fix:
	@echo "Running golangci-lint with auto-fix..."
	golangci-lint run --fix

.PHONY: mod-tidy
mod-tidy:
	@echo "Running go mod tidy..."
	go mod tidy

.PHONY: mod-verify
mod-verify:
	@echo "Verifying go mod dependencies..."
	go mod verify

# Security targets
.PHONY: security
security:
	@echo "Running gosec security scanner..."
	@command -v gosec >/dev/null 2>&1 || { echo "Installing gosec..."; go install github.com/securego/gosec/v2/cmd/gosec@latest; }
	gosec ./...

.PHONY: security-sarif
security-sarif:
	@echo "Running gosec security scanner with SARIF output..."
	@mkdir -p $(COVERAGE_DIR)
	@command -v gosec >/dev/null 2>&1 || { echo "Installing gosec..."; go install github.com/securego/gosec/v2/cmd/gosec@latest; }
	gosec -no-fail -fmt sarif -out $(COVERAGE_DIR)/gosec.sarif ./...
	@echo "SARIF report generated: $(COVERAGE_DIR)/gosec.sarif"

# CI targets
.PHONY: ci
ci: format vet lint test
	@echo "All basic CI checks passed!"

.PHONY: ci-full
ci-full: mod-tidy mod-verify vet lint-fix test-ci security-sarif
	@echo "Running complete CI pipeline locally..."
	@echo "✅ Dependencies verified"
	@echo "✅ Code formatted and linted"
	@echo "✅ Tests passed with race detection and coverage"
	@echo "✅ Security scan completed"
	@echo "🎉 Full CI pipeline completed successfully!"

# Clean targets
.PHONY: clean
clean:
	@echo "Cleaning up..."
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)
	go clean

# Development targets
.PHONY: dev-deps
dev-deps:
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest

.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: run-dry
run-dry: build
	@echo "Running $(BINARY_NAME) in dry-run mode..."
	./$(BUILD_DIR)/$(BINARY_NAME) --dry-run --verbose

# Release targets
.PHONY: release
release:
	@echo "Building release binaries..."
	@mkdir -p $(BUILD_DIR)/release
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-arm64 $(MAIN_PACKAGE)
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	@echo "Release binaries built in $(BUILD_DIR)/release/"

.PHONY: checksums
checksums: release
	@echo "Generating checksums..."
	cd $(BUILD_DIR)/release && sha256sum * > checksums.txt
	@echo "Checksums generated in $(BUILD_DIR)/release/checksums.txt"

# Docker targets (optional)
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):latest .

.PHONY: docker-test
docker-test:
	@echo "Running tests in Docker..."
	docker run --rm -v $(PWD):/app -w /app golang:1.21 make test
