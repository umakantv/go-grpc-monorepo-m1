package file

import "github.com/yourorg/monorepo/pkg/metrics"

// GatewayPatterns returns the HTTP route patterns defined in this service.
func GatewayPatterns() []metrics.RoutePattern {
	return []metrics.RoutePattern{
		{Method: "POST", Pattern: "/v1/files/upload-signed-url"},
		{Method: "POST", Pattern: "/v1/files/{file_id}/confirm"},
		{Method: "GET", Pattern: "/v1/files/{file_id}"},
		{Method: "DELETE", Pattern: "/v1/files/{file_id}"},
		{Method: "GET", Pattern: "/v1/files"},
	}
}