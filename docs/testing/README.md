# Basket Service — Testing Guide

## Overview
The basket service uses Go's testing framework plus golangci-lint for static analysis. Tests fall into three buckets: fast unit tests, Redis-backed integration tests, and CI-focused workflows that exercise both.

## Prerequisites
- Go 1.21+
- Redis 7+ (for integration tests)
- GNU Make

## Unit Tests
Run the fast suites that mock Redis and RabbitMQ:
```bash
# Run all unit tests
make test-unit

# Or directly
go test ./...
```
Key spec files:
- `internal/model/cart_test.go` — domain logic validation
- `internal/service/cart_service_test.go` — service behavior with mocks

## Integration Tests
These suites connect to a live Redis instance. `make test-integration` expects Redis reachable on `localhost:6379` (set `REDIS_PASSWORD` if needed). `make test-integration-ci` requires `REDIS_ADDR` and `REDIS_PASSWORD` env vars pointing at a reachable Redis instance — no `kubectl` context required.
```bash
# Run Redis-backed integration suite
make test-integration

# CI variant expects env vars REDIS_ADDR/REDIS_PASSWORD
make test-integration-ci
```
Spec files:
- `internal/service/cart_service_integration_test.go`
- `internal/repository/cart_repository_integration_test.go`

## Coverage
Generate HTML/text coverage reports across all packages:
```bash
make coverage
# outputs coverage.out + coverage.html
```

## Linting
Static analysis via golangci-lint (govet, errcheck, staticcheck, gofmt, goimports):
```bash
make lint
```

## End-to-End Considerations
Full platform flows are exercised in the shopping-cart-e2e-tests repo. When debugging auth or order hand-offs, run the frontend + backend stack through k3d-manager to reproduce issues end-to-end.
