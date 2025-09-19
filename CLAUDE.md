# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AvroCurio is a Go library for Apache Avro serialization/deserialization using Confluent Schema Registry wire format with Apicurio Schema Registry. It's a Go port of the Python AvroCurio library.

## Common Development Commands

```bash
# Build the library
just build

# Run unit tests
just test

# Run tests with race detection
just test-race

# Run tests with coverage
just test-cover

# Run integration tests (requires Docker)
just test-integration

# Run linting
just lint

# Format code
just fmt

# Full development workflow (deps, format, lint, test)
just dev

# Start Apicurio Registry for testing
just docker-up

# Stop Apicurio Registry
just docker-down
```

## Architecture

### Core Components

- **ApicurioClient** (`client.go`): HTTP client for Apicurio Schema Registry with TTL-based caching for schemas and failed lookups
- **AvroSerializer** (`serializer.go`): Main serialization/deserialization interface using Confluent wire format
- **WireFormat** (`wireformat.go`): Implements Confluent Schema Registry wire format (magic byte + schema ID + payload)
- **Config** (`config.go`): Configuration management with retry policies and timeouts

### Key Dependencies

- `github.com/hamba/avro/v2` - Avro serialization
- `github.com/jellydator/ttlcache/v3` - Schema caching
- `github.com/sethvargo/go-retry` - Retry mechanisms

### Wire Format Implementation

The library implements Confluent Schema Registry wire format:
- Magic Byte (0x0) + Schema ID (4 bytes, big-endian) + Avro Payload

### Caching Strategy

- Successful schema lookups cached indefinitely
- Failed lookups cached with TTL to allow retries for transient errors
- Separate caches for schemas and failed lookups

## Testing

### Integration Tests
- Require running Apicurio Registry (use `make docker-up`)
- Tagged with `//go:build integration`
- Example usage available in `example_test.go`

### Test Structure
- Unit tests alongside source files (`*_test.go`)
- Integration tests in separate files with build tags
- Helper schemas in `internal/testhelpers/schemas.go`

## Key Environment Variables

- `APICURIO_URL` - Override default registry URL (default: http://localhost:8080)
