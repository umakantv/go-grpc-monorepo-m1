module github.com/yourorg/monorepo/clients/auth-client

go 1.24.0

require (
	github.com/yourorg/monorepo/gen/go/private/auth v0.0.0
	github.com/yourorg/monorepo/pkg v0.0.0
	google.golang.org/grpc v1.80.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.1 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260120221211-b8f7ae30c516 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/yourorg/monorepo/gen/go/private/auth => ../../gen/go/private/auth

replace github.com/yourorg/monorepo/pkg => ../../pkg
