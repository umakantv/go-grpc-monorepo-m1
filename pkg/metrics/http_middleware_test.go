package metrics

import "testing"

func TestPatternRegistry(t *testing.T) {
	registry := NewPatternRegistry()

	// Register patterns from gRPC-Gateway style
	registry.Register("GET", "/v1/users/{user_id}")
	registry.Register("GET", "/v1/users")
	registry.Register("POST", "/v1/users")
	registry.Register("PUT", "/v1/users/{user_id}")
	registry.Register("GET", "/files/{source}/{filename}")

	tests := []struct {
		name     string
		method   string
		path     string
		expected string
	}{
		{
			name:     "match user by numeric id",
			method:   "GET",
			path:     "/v1/users/123",
			expected: "/v1/users/:user_id",
		},
		{
			name:     "match user by uuid",
			method:   "GET",
			path:     "/v1/users/550e8400-e29b-41d4-a716-446655440000",
			expected: "/v1/users/:user_id",
		},
		{
			name:     "match user by string id",
			method:   "GET",
			path:     "/v1/users/abc123",
			expected: "/v1/users/:user_id",
		},
		{
			name:     "match list users",
			method:   "GET",
			path:     "/v1/users",
			expected: "/v1/users",
		},
		{
			name:     "match create user",
			method:   "POST",
			path:     "/v1/users",
			expected: "/v1/users",
		},
		{
			name:     "match files with two params",
			method:   "GET",
			path:     "/files/s3/document.pdf",
			expected: "/files/:source/:filename",
		},
		{
			name:     "no match - wrong method",
			method:   "DELETE",
			path:     "/v1/users/123",
			expected: "/v1/users/123", // fallback to raw path
		},
		{
			name:     "no match - unknown path",
			method:   "GET",
			path:     "/v1/unknown/123",
			expected: "/v1/unknown/123", // fallback to raw path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.Match(tt.method, tt.path)
			if result != tt.expected {
				t.Errorf("Match(%q, %q) = %q, want %q", tt.method, tt.path, result, tt.expected)
			}
		})
	}
}

func TestNormalizeGatewayPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple param",
			input:    "/v1/users/{user_id}",
			expected: "/v1/users/:user_id",
		},
		{
			name:     "param with regex",
			input:    "/v1/users/{user_id=.*}",
			expected: "/v1/users/:user_id",
		},
		{
			name:     "multiple params",
			input:    "/files/{source}/{filename}",
			expected: "/files/:source/:filename",
		},
		{
			name:     "no params",
			input:    "/v1/users",
			expected: "/v1/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeGatewayPattern(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeGatewayPattern(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		path     string
		expected bool
	}{
		{
			name:     "match with param",
			pattern:  "/v1/users/:user_id",
			path:     "/v1/users/123",
			expected: true,
		},
		{
			name:     "match with uuid param",
			pattern:  "/v1/users/:user_id",
			path:     "/v1/users/550e8400-e29b-41d4-a716-446655440000",
			expected: true,
		},
		{
			name:     "no match - different length",
			pattern:  "/v1/users/:user_id",
			path:     "/v1/users",
			expected: false,
		},
		{
			name:     "no match - literal mismatch",
			pattern:  "/v1/users/:user_id",
			path:     "/v1/orders/123",
			expected: false,
		},
		{
			name:     "match multiple params",
			pattern:  "/files/:source/:filename",
			path:     "/files/s3/document.pdf",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchPattern(tt.pattern, tt.path)
			if result != tt.expected {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.path, result, tt.expected)
			}
		})
	}
}
