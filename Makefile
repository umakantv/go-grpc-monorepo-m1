.PHONY: all proto gen build test clean run-user-api run-auth-service deps docker-build docker-up docker-down help

# Default target
all: deps proto build

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	cd services/user-api && go mod download
	cd services/auth-service && go mod download
	cd proto && buf dep update

# Generate proto code
proto:
	@echo "Generating proto code..."
	cd proto && buf generate

# Run all code generation (proto + other generators)
gen: proto
	@echo "Running code generation..."
	go generate ./...

# Build all services
build:
	@echo "Building all services..."
	cd services/user-api && go build -o ../../bin/user-api ./cmd
	cd services/auth-service && go build -o ../../bin/auth-service ./cmd

# Build specific service
build-user-api:
	@echo "Building user-api..."
	cd services/user-api && go build -o ../../bin/user-api ./cmd

build-auth-service:
	@echo "Building auth-service..."
	cd services/auth-service && go build -o ../../bin/auth-service ./cmd

# Run tests
test:
	@echo "Running tests..."
	go test ./...

test-verbose:
	@echo "Running tests with verbose output..."
	go test -v ./...

# Run specific service
run-user-api:
	@echo "Running user-api..."
	cd services/user-api && go run ./cmd

run-auth-service:
	@echo "Running auth-service..."
	cd services/auth-service && go run ./cmd

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf gen/go/
	rm -rf gen/openapi/

# Lint code
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Check for issues
vet:
	@echo "Running go vet..."
	go vet ./...

# Docker commands
docker-build:
	@echo "Building Docker images..."
	docker-compose build

docker-up:
	@echo "Starting Docker containers..."
	docker-compose up -d

docker-down:
	@echo "Stopping Docker containers..."
	docker-compose down

docker-logs:
	docker-compose logs -f

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Generate database migrations (placeholder)
migrate-create:
	@echo "Creating migration..."
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations $$name

# Help
help:
	@echo "Available targets:"
	@echo "  deps              - Install dependencies"
	@echo "  proto             - Generate proto code"
	@echo "  gen               - Run all code generation"
	@echo "  build             - Build all services"
	@echo "  build-user-api    - Build user-api service"
	@echo "  build-auth-service- Build auth-service"
	@echo "  test              - Run tests"
	@echo "  test-verbose      - Run tests with verbose output"
	@echo "  run-user-api      - Run user-api service"
	@echo "  run-auth-service  - Run auth-service"
	@echo "  clean             - Clean build artifacts"
	@echo "  lint              - Run linter"
	@echo "  fmt               - Format code"
	@echo "  vet               - Run go vet"
	@echo "  docker-build      - Build Docker images"
	@echo "  docker-up         - Start Docker containers"
	@echo "  docker-down       - Stop Docker containers"
	@echo "  docker-logs       - View Docker logs"
	@echo "  install-tools     - Install development tools"
