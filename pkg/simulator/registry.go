package simulator

import (
	"fmt"
	"sync"
)

// Registry holds all registered simulators
type Registry struct {
	mu         sync.RWMutex
	simulators map[string]Simulator
}

// globalRegistry is the default registry
var globalRegistry = NewRegistry()

// NewRegistry creates a new simulator registry
func NewRegistry() *Registry {
	return &Registry{
		simulators: make(map[string]Simulator),
	}
}

// Register adds a simulator to the registry
func (r *Registry) Register(sim Simulator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := sim.Name()
	if _, exists := r.simulators[name]; exists {
		return fmt.Errorf("simulator %q already registered", name)
	}

	r.simulators[name] = sim
	return nil
}

// Get retrieves a simulator by name
func (r *Registry) Get(name string) (Simulator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sim, ok := r.simulators[name]
	return sim, ok
}

// Unregister removes a simulator from the registry
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.simulators, name)
}

// List returns all registered simulators
func (r *Registry) List() []Simulator {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Simulator, 0, len(r.simulators))
	for _, sim := range r.simulators {
		result = append(result, sim)
	}
	return result
}

// ListByCategory returns simulators for a specific category
func (r *Registry) ListByCategory(category Category) []Simulator {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Simulator
	for _, sim := range r.simulators {
		if sim.Category() == category {
			result = append(result, sim)
		}
	}
	return result
}

// Register adds a simulator to the global registry
func Register(sim Simulator) error {
	return globalRegistry.Register(sim)
}

// Get retrieves a simulator from the global registry
func Get(name string) (Simulator, bool) {
	return globalRegistry.Get(name)
}

// Unregister removes a simulator from the global registry
func Unregister(name string) {
	globalRegistry.Unregister(name)
}

// List returns all simulators from the global registry
func List() []Simulator {
	return globalRegistry.List()
}

// ListByCategory returns simulators for a category from the global registry
func ListByCategory(category Category) []Simulator {
	return globalRegistry.ListByCategory(category)
}
