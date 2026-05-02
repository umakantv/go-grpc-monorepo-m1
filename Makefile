.PHONY: all proto gen build test clean run-user-api run-auth-service run-file-service run-notification-service deps docker-build docker-up docker-down migrate-create migrate-up migrate-down migrate-force migrate-version help

# Default target
all: deps proto build

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	cd services/user-api && go mod download
	cd services/auth-service && go mod download
	cd services/file-service && go mod download
	cd services/notification-service && go mod download
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
	cd services/file-service && go build -o ../../bin/file-service ./cmd
	cd services/notification-service && go build -o ../../bin/notification-service ./cmd

# Build specific service
build-user-api:
	@echo "Building user-api..."
	cd services/user-api && go build -o ../../bin/user-api ./cmd

build-auth-service:
	@echo "Building auth-service..."
	cd services/auth-service && go build -o ../../bin/auth-service ./cmd

build-file-service:
	@echo "Building file-service..."
	cd services/file-service && go build -o ../../bin/file-service ./cmd

build-notification-service:
	@echo "Building notification-service..."
	cd services/notification-service && go build -o ../../bin/notification-service ./cmd

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

run-file-service:
	@echo "Running file-service..."
	cd services/file-service && go run ./cmd

run-notification-service:
	@echo "Running notification-service..."
	cd services/notification-service && go run ./cmd

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
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# ──────────────────────────────────────────────────────────────
# Database Migrations (powered by golang-migrate)
#
# All migration targets require SERVICE= to be set.
# migrate-up/down/force/version also require DATABASE_URL=.
#
# Examples:
#   make migrate-create SERVICE=auth-service NAME=add_email_index
#   make migrate-up     SERVICE=auth-service DATABASE_URL="postgres://postgres:postgres@localhost:5432/authdb?sslmode=disable"
#   make migrate-down   SERVICE=auth-service DATABASE_URL="postgres://postgres:postgres@localhost:5432/authdb?sslmode=disable"
#   make migrate-version SERVICE=auth-service DATABASE_URL="postgres://postgres:postgres@localhost:5432/authdb?sslmode=disable"
#   make migrate-force  SERVICE=auth-service DATABASE_URL="postgres://postgres:postgres@localhost:5432/authdb?sslmode=disable" VERSION=1
# ──────────────────────────────────────────────────────────────

# Create a new migration file pair (up + down)
migrate-create:
ifndef SERVICE
	$(error SERVICE is required. Usage: make migrate-create SERVICE=auth-service NAME=add_email_index)
endif
ifndef NAME
	$(error NAME is required. Usage: make migrate-create SERVICE=auth-service NAME=add_email_index)
endif
	@migrate create -ext sql -dir services/$(SERVICE)/migrations -seq $(NAME)
	@echo "Created migration files in services/$(SERVICE)/migrations/"

# Apply all pending migrations
migrate-up:
ifndef SERVICE
	$(error SERVICE is required. Usage: make migrate-up SERVICE=auth-service DATABASE_URL="...")
endif
ifndef DATABASE_URL
	$(error DATABASE_URL is required. Example: postgres://postgres:postgres@localhost:5432/authdb?sslmode=disable)
endif
	@migrate -path services/$(SERVICE)/migrations -database "$(DATABASE_URL)" up
	@echo "Migrations applied for $(SERVICE)."

# Roll back the last migration
migrate-down:
ifndef SERVICE
	$(error SERVICE is required. Usage: make migrate-down SERVICE=auth-service DATABASE_URL="...")
endif
ifndef DATABASE_URL
	$(error DATABASE_URL is required.)
endif
	@migrate -path services/$(SERVICE)/migrations -database "$(DATABASE_URL)" down 1
	@echo "Rolled back 1 migration for $(SERVICE)."

# Force-set the migration version (useful to fix dirty state)
migrate-force:
ifndef SERVICE
	$(error SERVICE is required.)
endif
ifndef DATABASE_URL
	$(error DATABASE_URL is required.)
endif
ifndef VERSION
	$(error VERSION is required. Usage: make migrate-force SERVICE=auth-service DATABASE_URL="..." VERSION=1)
endif
	@migrate -path services/$(SERVICE)/migrations -database "$(DATABASE_URL)" force $(VERSION)
	@echo "Forced migration version to $(VERSION) for $(SERVICE)."

# Print current migration version
migrate-version:
ifndef SERVICE
	$(error SERVICE is required.)
endif
ifndef DATABASE_URL
	$(error DATABASE_URL is required.)
endif
	@migrate -path services/$(SERVICE)/migrations -database "$(DATABASE_URL)" version

# Help
help:
	@echo "Available targets:"
	@echo "  deps              - Install dependencies"
	@echo "  proto             - Generate proto code"
	@echo "  gen               - Run all code generation"
	@echo "  build             - Build all services"
	@echo "  build-user-api    - Build user-api service"
	@echo "  build-auth-service- Build auth-service"
	@echo "  build-file-service- Build file-service"
	@echo "  build-notification-service - Build notification-service"
	@echo "  test              - Run tests"
	@echo "  test-verbose      - Run tests with verbose output"
	@echo "  run-user-api      - Run user-api service"
	@echo "  run-auth-service  - Run auth-service"
	@echo "  run-file-service  - Run file-service"
	@echo "  run-notification-service - Run notification-service"
	@echo "  clean             - Clean build artifacts"
	@echo "  lint              - Run linter"
	@echo "  fmt               - Format code"
	@echo "  vet               - Run go vet"
	@echo "  docker-build      - Build Docker images"
	@echo "  docker-up         - Start Docker containers"
	@echo "  docker-down       - Stop Docker containers"
	@echo "  docker-logs       - View Docker logs"
	@echo "  install-tools     - Install development tools"
	@echo ""
	@echo "Migration targets (require SERVICE= and DATABASE_URL= where noted):"
	@echo "  migrate-create    - Create new migration (SERVICE, NAME)"
	@echo "  migrate-up        - Apply pending migrations (SERVICE, DATABASE_URL)"
	@echo "  migrate-down      - Roll back last migration (SERVICE, DATABASE_URL)"
	@echo "  migrate-version   - Print current version (SERVICE, DATABASE_URL)"
	@echo "  migrate-force     - Force-set version (SERVICE, DATABASE_URL, VERSION)"
