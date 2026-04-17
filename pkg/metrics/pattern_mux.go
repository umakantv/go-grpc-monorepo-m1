package metrics

import (
	"strings"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

// RoutePattern represents a registered route pattern
type RoutePattern struct {
	Method  string
	Pattern string
}

// PatternRegistry stores route patterns for path normalization
type PatternRegistry struct {
	mu       sync.RWMutex
	patterns []RoutePattern
}

// NewPatternRegistry creates a new pattern registry
func NewPatternRegistry() *PatternRegistry {
	return &PatternRegistry{
		patterns: make([]RoutePattern, 0),
	}
}

// Register adds a route pattern to the registry
func (r *PatternRegistry) Register(method, pattern string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.patterns = append(r.patterns, RoutePattern{
		Method:  method,
		Pattern: normalizeGatewayPattern(pattern),
	})
}

// RegisterAll adds multiple route patterns to the registry
func (r *PatternRegistry) RegisterAll(patterns []RoutePattern) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range patterns {
		r.patterns = append(r.patterns, RoutePattern{
			Method:  p.Method,
			Pattern: normalizeGatewayPattern(p.Pattern),
		})
	}
}

// Match finds the matching pattern for a given method and path
func (r *PatternRegistry) Match(method, path string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.patterns {
		if p.Method != method {
			continue
		}
		if matchPattern(p.Pattern, path) {
			return p.Pattern
		}
	}
	return path // fallback to raw path if no pattern matches
}

// Patterns returns all registered patterns
func (r *PatternRegistry) Patterns() []RoutePattern {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]RoutePattern, len(r.patterns))
	copy(result, r.patterns)
	return result
}

// PatternMux wraps runtime.ServeMux with pattern tracking.
// It intercepts Handle calls to capture registered patterns.
type PatternMux struct {
	*runtime.ServeMux
	registry *PatternRegistry
}

// NewPatternMux creates a new PatternMux with the given ServeMuxOptions.
// The returned mux can be passed to gRPC-Gateway registration functions.
func NewPatternMux(opts ...runtime.ServeMuxOption) *PatternMux {
	return &PatternMux{
		ServeMux: runtime.NewServeMux(opts...),
		registry: NewPatternRegistry(),
	}
}

// Registry returns the pattern registry for this mux
func (pm *PatternMux) Registry() *PatternRegistry {
	return pm.registry
}

// MatchPattern finds the matching pattern for a given request
func (pm *PatternMux) MatchPattern(method, path string) string {
	return pm.registry.Match(method, path)
}

// normalizeGatewayPattern converts gRPC-Gateway pattern to normalized form
// e.g., /v1/users/{user_id} -> /v1/users/:user_id
func normalizeGatewayPattern(pattern string) string {
	// Replace {param} with :param
	result := pattern
	for {
		start := strings.Index(result, "{")
		if start == -1 {
			break
		}
		end := strings.Index(result, "}")
		if end == -1 || end < start {
			break
		}
		param := result[start+1 : end]
		// Handle patterns like {user_id=.*} by taking only the param name
		if eq := strings.Index(param, "="); eq != -1 {
			param = param[:eq]
		}
		result = result[:start] + ":" + param + result[end+1:]
	}
	return result
}

// matchPattern checks if a path matches a pattern
// pattern: /v1/users/:user_id
// path: /v1/users/123
func matchPattern(pattern, path string) bool {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i := 0; i < len(patternParts); i++ {
		pp := patternParts[i]
		lp := pathParts[i]

		// If pattern part starts with ':', it's a variable - matches anything
		if strings.HasPrefix(pp, ":") {
			continue
		}

		// Otherwise, must match exactly
		if pp != lp {
			return false
		}
	}

	return true
}
