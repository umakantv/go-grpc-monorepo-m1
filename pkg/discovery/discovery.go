package discovery

import (
	"fmt"
	"sync"
)

// Registry is an interface for service discovery
type Registry interface {
	Resolve(serviceName string) (string, error)
	Register(serviceName, addr string) error
	Deregister(serviceName string) error
}

// StaticRegistry is a simple in-memory service registry for development
type StaticRegistry struct {
	mu      sync.RWMutex
	service map[string]string
}

// NewStaticRegistry creates a new static registry
func NewStaticRegistry() *StaticRegistry {
	return &StaticRegistry{
		service: make(map[string]string),
	}
}

// Resolve returns the address for a service
func (r *StaticRegistry) Resolve(serviceName string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	addr, ok := r.service[serviceName]
	if !ok {
		return "", fmt.Errorf("service %s not found", serviceName)
	}
	return addr, nil
}

// Register registers a service with its address
func (r *StaticRegistry) Register(serviceName, addr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.service[serviceName] = addr
	return nil
}

// Deregister removes a service from the registry
func (r *StaticRegistry) Deregister(serviceName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.service, serviceName)
	return nil
}

// DefaultRegistry is a global registry instance
var DefaultRegistry = NewStaticRegistry()

// MustResolve resolves a service address or panics
func MustResolve(serviceName string) string {
	addr, err := DefaultRegistry.Resolve(serviceName)
	if err != nil {
		panic(fmt.Sprintf("failed to resolve service %s: %v", serviceName, err))
	}
	return addr
}
