module github.com/yourorg/monorepo/gen/go/public/file

go 1.24.0

require (
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.1
	github.com/yourorg/monorepo/pkg v0.0.0
	google.golang.org/genproto/googleapis/api v0.0.0-20260120221211-b8f7ae30c516
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/prometheus/client_golang v1.19.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
)

replace github.com/yourorg/monorepo/pkg => ../../../../pkg
