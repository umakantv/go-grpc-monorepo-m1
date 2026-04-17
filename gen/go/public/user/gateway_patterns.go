package user

import "github.com/yourorg/monorepo/pkg/metrics"

// GatewayPatterns returns the HTTP route patterns defined in this service.
// These patterns are extracted from the proto google.api.http annotations.
func GatewayPatterns() []metrics.RoutePattern {
	return []metrics.RoutePattern{
		{Method: "GET", Pattern: "/v1/users/{user_id}"},
		{Method: "POST", Pattern: "/v1/users"},
		{Method: "PUT", Pattern: "/v1/users/{user_id}"},
		{Method: "DELETE", Pattern: "/v1/users/{user_id}"},
		{Method: "GET", Pattern: "/v1/users"},
	}
}
