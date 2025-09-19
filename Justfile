# AvroCurio Go Justfile

_default:
    @{{just_executable()}} --choose

# Run unit tests
test:
    go test -shuffle=on -timeout=15s ./...

# Run tests with race detection
test-race:
    go test -shuffle=on -race -timeout=1m ./...

# Run tests with coverage
test-cover:
    go test -shuffle=on -coverprofile=coverage.out -timeout=5m ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Run tests excluding integration tests
test-short:
    go test -shuffle=on -short -timeout=15s ./...

# Run integration tests (requires Docker)
test-integration:
    go test -shuffle=on -tags=integration -timeout=1m ./...

# Run benchmarks
bench:
    go test -bench=. -benchmem ./...

# Run CPU benchmarks
bench-cpu:
    go test -bench=. -cpuprofile=cpu.prof ./...
    go tool pprof cpu.prof

# Run memory benchmarks
bench-mem:
    go test -bench=. -memprofile=mem.prof ./...
    go tool pprof mem.prof

# Run linter
lint:
    golangci-lint run

# Format code
fmt:
    go fmt ./...
    @if command -v goimports >/dev/null 2>&1; then goimports -w .; fi

# Download dependencies
deps:
    go mod download

# Tidy dependencies
tidy:
    go mod tidy

# Start Apicurio Registry for testing
docker-up:
    docker compose up -d

# Stop Apicurio Registry
docker-down:
    docker compose down

# Build the library
build:
    go build ./...

# Clean build artifacts and test files
clean:
    go clean
    rm -f coverage.out coverage.html
    rm -f cpu.prof mem.prof
    rm -f *.test

# Run development workflow (deps, format, lint, test)
dev: deps fmt lint test

# Generate documentation
docs:
    go doc -http

# Show build information
buildinfo:
    @echo "Go version: $(go version)"
    @echo "Module: $(go list -m)"
    @echo "Dependencies:"
    @go list -m -u all
