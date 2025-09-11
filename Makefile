# AvroCurio Go Makefile

.PHONY: help test test-integration test-race test-cover bench lint fmt clean deps tidy docker-up docker-down

help: ## Show this help message
	@echo 'Usage: make <target>'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run unit tests
	go test -shuffle=on -timeout=15s ./...

test-race: ## Run tests with race detection
	go test -shuffle=on -race -timeout=1m ./...

test-cover: ## Run tests with coverage
	go test -shuffle=on -coverprofile=coverage.out -timeout=5m ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-short: ## Run tests excluding integration tests
	go test -shuffle=on -short -timeout=15s ./...

test-integration: ## Run integration tests (requires Docker)
	go test -shuffle=on -tags=integration -timeout=1m ./...

bench: ## Run benchmarks
	go test -bench=. -benchmem ./...

bench-cpu: ## Run CPU benchmarks
	go test -bench=. -cpuprofile=cpu.prof ./...
	go tool pprof cpu.prof

bench-mem: ## Run memory benchmarks
	go test -bench=. -memprofile=mem.prof ./...
	go tool pprof mem.prof

lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

deps: ## Download dependencies
	go mod download

tidy: ## Tidy dependencies
	go mod tidy

docker-up: ## Start Apicurio Registry for testing
	docker compose up -d;

docker-down: ## Stop Apicurio Registry
	docker compose down;

build: ## Build the library
	go build ./...

clean: ## Clean build artifacts and test files
	go clean
	rm -f coverage.out coverage.html
	rm -f cpu.prof mem.prof
	rm -f *.test

dev: deps fmt lint test ## Run development workflow (deps, format, lint, test)

docs: ## Generate documentation
	go doc -http

buildinfo: ## Show build information
	@echo "Go version: $(shell go version)"
	@echo "Module: $(shell go list -m)"
	@echo "Dependencies:"
	@go list -m -u all
