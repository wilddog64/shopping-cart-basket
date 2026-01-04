.PHONY: all build test test-unit test-integration coverage lint fmt clean run docker-build docker-run help

# Variables
APP_NAME := cart-service
GO := go
GOFLAGS := -v
LDFLAGS := -ldflags "-s -w"
DOCKER_IMAGE := shopping-cart/cart-service
DOCKER_TAG := latest

# Default target
all: lint test build

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(APP_NAME) ./cmd/server

# Run the application
run:
	@echo "Running $(APP_NAME)..."
	$(GO) run ./cmd/server

# Run all tests
test:
	@echo "Running all tests..."
	$(GO) test $(GOFLAGS) ./... -cover

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	$(GO) test $(GOFLAGS) ./internal/... -cover

# Run integration tests (starts port-forward if needed, runs tests, cleans up)
test-integration:
	@echo "Running integration tests..."
	@if nc -z localhost 6379 2>/dev/null; then \
		echo "Redis already accessible on localhost:6379"; \
		REDIS_PASSWORD=$$(kubectl get secret -n shopping-cart-data redis-cart-secret -o jsonpath='{.data.password}' | base64 -d) \
		$(GO) test $(GOFLAGS) -tags=integration ./... -cover -timeout 120s; \
	else \
		echo "Starting Redis port-forward..."; \
		kubectl port-forward -n shopping-cart-data svc/redis-cart 6379:6379 & \
		PF_PID=$$!; \
		sleep 2; \
		REDIS_PASSWORD=$$(kubectl get secret -n shopping-cart-data redis-cart-secret -o jsonpath='{.data.password}' | base64 -d) \
		$(GO) test $(GOFLAGS) -tags=integration ./... -cover -timeout 120s; \
		TEST_EXIT=$$?; \
		kill $$PF_PID 2>/dev/null; \
		exit $$TEST_EXIT; \
	fi

# Run integration tests with custom Redis (for CI with external Redis)
test-integration-ci:
	@echo "Running integration tests (CI mode)..."
	REDIS_ADDR=$(REDIS_ADDR) REDIS_PASSWORD=$(REDIS_PASSWORD) $(GO) test $(GOFLAGS) -tags=integration ./... -cover -timeout 120s

# Generate coverage report
coverage:
	@echo "Generating coverage report..."
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, running go vet..."; \
		$(GO) vet ./...; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf dist/
	rm -f coverage.out coverage.html

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run -p 8083:8083 --env-file .env $(DOCKER_IMAGE):$(DOCKER_TAG)

# Generate mocks (requires mockgen)
mocks:
	@echo "Generating mocks..."
	@if command -v mockgen >/dev/null 2>&1; then \
		mockgen -source=internal/repository/cart_repository.go -destination=internal/repository/mock_cart_repository.go -package=repository; \
		mockgen -source=internal/service/cart_service.go -destination=internal/service/mock_cart_service.go -package=service; \
	else \
		echo "mockgen not installed. Run: go install github.com/golang/mock/mockgen@latest"; \
	fi

# Show help
help:
	@echo "Available targets:"
	@echo "  all              - Run lint, test, and build"
	@echo "  build            - Build the application"
	@echo "  run              - Run the application"
	@echo ""
	@echo "Testing:"
	@echo "  test             - Run unit tests (no external deps)"
	@echo "  test-unit        - Run unit tests only"
	@echo "  test-integration - Run integration tests (auto-starts Redis port-forward)"
	@echo "  test-integration-ci - Run integration tests with REDIS_ADDR/REDIS_PASSWORD env vars"
	@echo "  coverage         - Generate coverage report"
	@echo ""
	@echo "Development:"
	@echo "  lint             - Run linter"
	@echo "  fmt              - Format code"
	@echo "  clean            - Clean build artifacts"
	@echo "  deps             - Download dependencies"
	@echo "  mocks            - Generate mock files"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-run       - Run Docker container"
	@echo ""
	@echo "  help             - Show this help"
