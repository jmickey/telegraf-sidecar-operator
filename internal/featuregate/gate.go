/*
Copyright 2024 Josh Michielsen.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package featuregate

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
)

// Gate represents a feature that can be enabled or disabled.
type Gate struct {
	Name        string
	Description string
	enabled     *atomic.Bool
}

// IsEnabled returns true if the feature gate is enabled.
func (g *Gate) IsEnabled() bool {
	return g.enabled.Load()
}

// setEnabled sets the enabled state of the gate.
func (g *Gate) setEnabled(enabled bool) {
	g.enabled.Store(enabled)
}

// Registry manages feature gates in a thread-safe manner.
type Registry struct {
	gates sync.Map
}

// NewRegistry creates a new Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register creates and registers a new Gate.
// Panics if a gate with the same name is already registered.
func (r *Registry) Register(name, description string, defaultEnabled bool) *Gate {
	gate := &Gate{
		Name:        name,
		Description: description,
		enabled:     &atomic.Bool{},
	}
	gate.enabled.Store(defaultEnabled)

	if _, loaded := r.gates.LoadOrStore(name, gate); loaded {
		panic(fmt.Sprintf("feature gate %q is already registered", name))
	}

	return gate
}

// Get retrieves a Gate by name.
// Returns nil if the gate doesn't exist.
func (r *Registry) Get(name string) *Gate {
	if value, ok := r.gates.Load(name); ok {
		return value.(*Gate)
	}
	return nil
}

// Set enables or disables a gate by name.
// Returns an error if the gate doesn't exist.
func (r *Registry) Set(name string, enabled bool) error {
	gate := r.Get(name)
	if gate == nil {
		var validGates []string
		r.VisitAll(func(g *Gate) {
			validGates = append(validGates, g.Name)
		})
		return fmt.Errorf("unknown feature gate %q, valid gates: %v", name, validGates)
	}
	gate.setEnabled(enabled)
	return nil
}

// VisitAll calls the provided function for each registered Gate in lexicographical order.
func (r *Registry) VisitAll(fn func(*Gate)) {
	var gates []*Gate
	r.gates.Range(func(key, value interface{}) bool {
		gates = append(gates, value.(*Gate))
		return true
	})

	// Sort gates by name for consistent iteration
	sort.Slice(gates, func(i, j int) bool {
		return gates[i].Name < gates[j].Name
	})

	for _, gate := range gates {
		fn(gate)
	}
}

// Global registry instance
var globalRegistry = NewRegistry()

// GlobalRegistry returns the global Registry instance.
func GlobalRegistry() *Registry {
	return globalRegistry
}

// Convenience functions that operate on the global registry

// Register creates and registers a new Gate in the global registry.
func Register(name, description string, defaultEnabled bool) *Gate {
	return globalRegistry.Register(name, description, defaultEnabled)
}

// Get retrieves a Gate by name from the global registry.
func Get(name string) *Gate {
	return globalRegistry.Get(name)
}

// Set enables or disables a gate by name in the global registry.
func Set(name string, enabled bool) error {
	return globalRegistry.Set(name, enabled)
}

// VisitAll calls the provided function for each Gate in the global registry.
func VisitAll(fn func(*Gate)) {
	globalRegistry.VisitAll(fn)
}
