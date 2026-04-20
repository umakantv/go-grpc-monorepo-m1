# Go Monorepo Framework

A monorepo framework for building multiple Go services with gRPC and HTTP support.

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────────────┐
│                           Proto Definitions                          │
│                    (Single Source of Truth)                          │
└─────────────────────────────────────┬────────────────────────────────┘
                                      │
                    ┌─────────────────┴─────────────────┐
                    ▼                                   ▼
        ┌───────────────────────┐         ┌───────────────────────┐
        │   gRPC Service Code   │         │   HTTP/Gateway Code   │
        │   (protoc-gen-go)     │         │ (protoc-gen-grpc-     │
        │                       │         │  gateway)             │
        └───────────────────────┘         └───────────────────────┘
                    │                                   │
                    ▼                                   ▼
        ┌───────────────────────┐         ┌───────────────────────┐
        │   Internal Services   │         │   Public Services     │
        │   (gRPC only)         │         │   (HTTP → gRPC)       │
        └───────────────────────┘         └───────────────────────┘
```

## Project Structure

```
/
├── proto/                    # Centralized proto definitions
│   ├── public/               # Public-facing service protos (with HTTP annotations)
│   │   ├── user/v1/
│   │   └── product/v1/
│   └── internal/             # Internal-only service protos (no HTTP)
│       ├── auth/v1/
│       └── payment/v1/
│
├── gen/                      # Generated code
│   ├── go/                   # Generated Go code
│   └── openapi/              # Generated OpenAPI specs
│
├── pkg/                      # Shared libraries
│   ├── config/               # Config loading utilities
│   ├── database/             # DB connection helpers
│   ├── logging/              # Logging setup
│   ├── middleware/           # Shared HTTP/gRPC middleware
│   └── discovery/            # Service discovery client
│
├── services/
│   ├── user-api/             # Public service (HTTP + gRPC)
│   └── auth-service/         # Internal service (gRPC only)
│
├── clients/                  # Pre-built client libraries
│   └── auth-client/
│
└── tools/                    # Development tools
```

## Service Types

### Public Services (HTTP + gRPC)
- Expose both HTTP and gRPC endpoints
- Use `google.api.http` annotations in proto
- gRPC-Gateway generates HTTP handlers automatically
- Example: `user-api`, `product-api`

### Internal Services (gRPC Only)
- Only expose gRPC endpoints
- No HTTP annotations in proto
- Called by other services via gRPC
- Example: `auth-service`, `payment-service`

## Quick Start

### Prerequisites
- Go 1.22+
- Buf CLI (for proto generation)
- Docker (optional, for local development)

### Install Development Tools
```bash
make install-tools
```

### Generate Proto Code
```bash
make proto
```

### Build All Services
```bash
make build
```

### Run Services Locally

1. Start databases:
```bash
docker-compose up -d user-db auth-db
```

2. Run auth-service:
```bash
make run-auth-service
```

3. In another terminal, run user-api:
```bash
make run-user-api
```

### Run with Docker Compose
```bash
make docker-up
```

## API Endpoints

### User API (Public)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/v1/users/{user_id}` | Get user by ID |
| POST | `/v1/users` | Create user |
| PUT | `/v1/users/{user_id}` | Update user |
| DELETE | `/v1/users/{user_id}` | Delete user |
| GET | `/v1/users` | List users |

### Auth Service (Internal - gRPC only)

| Method | Description |
|--------|-------------|
| ValidateToken | Validate JWT token |
| GenerateToken | Generate new JWT token |
| RefreshToken | Refresh existing token |
| RevokeToken | Revoke token (logout) |

## Configuration

Each service has its own `config.yaml`:

```yaml
service:
  name: user-api
  env: development

grpc:
  port: 9090

http:
  enabled: true    # false for internal services
  port: 8080

logging:
  level: debug
  format: console

database:
  enabled: true
  driver: postgres
  host: localhost
  port: 5432
  name: userdb
  user: postgres
  password: postgres
```

### Environment-Specific Configs

The config loader first reads the base `config.yaml`, then overlays `config.{env}.yaml` if present.

Set the environment via the `APP_ENV` environment variable or the `service.env` field:

```bash
# Load config.production.yaml on top of config.yaml
APP_ENV=production ./user-api
```

This allows keeping common defaults in `config.yaml` and environment-specific overrides in separate files (e.g. `config.production.yaml`, `config.staging.yaml`).

Environment variables always take highest precedence (e.g. `DATABASE_HOST=...`).

### Database Drivers

The `database.driver` setting controls the backend:

- `postgres` (default): Uses lib/pq with host/port/user/password config
- `sqlite`: Uses modernc.org/sqlite

For SQLite:
- `path: ":memory:"` → in-memory database (ideal for tests)
- `path: ./data/app.db` → file-based database (local development)

Example for local development:

```yaml
database:
  enabled: true
  driver: sqlite
  path: ./data/user.db
```

Example for tests:

```yaml
database:
  enabled: true
  driver: sqlite
  path: ":memory:"
```

Environment variables still override all fields.

## Adding a New Service

### 1. Create Proto Definition

For public services (`proto/public/myservice/v1/myservice.proto`):
```protobuf
syntax = "proto3";
package public.myservice.v1;

import "google/api/annotations.proto";

service MyService {
  rpc DoSomething(DoSomethingRequest) returns (DoSomethingResponse) {
    option (google.api.http) = {
      post: "/v1/do-something"
      body: "*"
    };
  }
}
```

For internal services (`proto/internal/myservice/v1/myservice.proto`):
```protobuf
syntax = "proto3";
package internal.myservice.v1;

// No google/api/annotations import
service MyService {
  rpc DoSomething(DoSomethingRequest) returns (DoSomethingResponse);
}
```

### 2. Generate Code
```bash
make proto
```

### 3. Create Service Directory
```bash
mkdir -p services/myservice/cmd
mkdir -p services/myservice/internal/service
mkdir -p services/myservice/internal/repository
```

### 4. Implement Service
- Create `internal/service/service.go` implementing the gRPC interface
- Create `cmd/main.go` to start the server

### 5. Add to go.work
```
use ./services/myservice
```

## Import Rules

Services cannot directly import code from other services' internal packages:

```
# Allowed
import "github.com/yourorg/monorepo/pkg/config"
import "github.com/yourorg/monorepo/gen/go/public/user/v1"
import "github.com/yourorg/monorepo/clients/auth-client"

# NOT Allowed
import "github.com/yourorg/monorepo/services/user-api/internal/service"
import "github.com/yourorg/monorepo/services/auth-service/internal/repository"
```

Use gRPC clients for inter-service communication instead.

## Available Make Targets

```
make help          # Show all available targets
make proto         # Generate proto code
make build         # Build all services
make test          # Run tests
make lint          # Run linter
make docker-up     # Start all services with Docker
make docker-down   # Stop all services
```

## Development Workflow

1. Define API in proto file
2. Run `make proto` to generate code
3. Implement service logic
4. Run `make test` to verify
5. Run `make build` to build binaries
