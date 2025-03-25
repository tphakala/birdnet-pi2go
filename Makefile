.PHONY: all build test test-short test-verbose test-cover test-race benchmark fuzz lint clean

GO_FILES := $(shell find . -name "*.go" -not -path "./vendor/*")

# Default target
all: lint test build

# Build the application
build:
	go build -v

# Run all tests
test:
	go test -v ./...

# Run tests in short mode (skip long-running tests)
test-short:
	go test -short ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run tests with race detector
test-race:
	go test -race ./...

# Run benchmarks
benchmark:
	go test -bench=. -benchmem ./...

# Run fuzz tests (requires Go 1.18+)
fuzz:
	echo "Running fuzz tests (will run for 10 seconds each)"
	go test -fuzz=FuzzGenerateClipName -fuzztime=10s
	go test -fuzz=FuzzConvertDetectionToNote -fuzztime=10s
	go test -fuzz=FuzzFormulateQuery -fuzztime=10s

# Run linter (requires golangci-lint)
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, installing..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

# Clean build artifacts
clean:
	go clean
	rm -f coverage.out coverage.html
	rm -f birdnet-pi2go

# Run all standard tests and static analysis
ci: lint test-cover test-race