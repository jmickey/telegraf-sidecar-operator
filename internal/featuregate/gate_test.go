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
	"flag"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestGate_IsEnabled(t *testing.T) {
	tests := []struct {
		name           string
		defaultEnabled bool
		setEnabled     *bool // nil means don't call setEnabled
		expected       bool
	}{
		{
			name:           "disabled by default",
			defaultEnabled: false,
			expected:       false,
		},
		{
			name:           "enabled by default",
			defaultEnabled: true,
			expected:       true,
		},
		{
			name:           "disabled then enabled",
			defaultEnabled: false,
			setEnabled:     boolPtr(true),
			expected:       true,
		},
		{
			name:           "enabled then disabled",
			defaultEnabled: true,
			setEnabled:     boolPtr(false),
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()
			gate := reg.Register("test.gate", "Test gate", tt.defaultEnabled)

			if tt.setEnabled != nil {
				gate.setEnabled(*tt.setEnabled)
			}

			if got := gate.IsEnabled(); got != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGate_Metadata(t *testing.T) {
	tests := []struct {
		name        string
		gateName    string
		description string
	}{
		{
			name:        "simple gate",
			gateName:    "simple",
			description: "Simple test gate",
		},
		{
			name:        "namespaced gate",
			gateName:    "operator.feature.test",
			description: "Namespaced feature gate",
		},
		{
			name:        "empty description",
			gateName:    "test.empty",
			description: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()
			gate := reg.Register(tt.gateName, tt.description, false)

			if gate.Name != tt.gateName {
				t.Errorf("Name = %q, want %q", gate.Name, tt.gateName)
			}
			if gate.Description != tt.description {
				t.Errorf("Description = %q, want %q", gate.Description, tt.description)
			}
		})
	}
}

func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name           string
		gateName       string
		description    string
		defaultEnabled bool
		shouldPanic    bool
		setupGate      string // gate to register first (for duplicate test)
	}{
		{
			name:           "successful registration",
			gateName:       "test.success",
			description:    "Successful test",
			defaultEnabled: false,
			shouldPanic:    false,
		},
		{
			name:           "enabled by default",
			gateName:       "test.enabled",
			description:    "Enabled test",
			defaultEnabled: true,
			shouldPanic:    false,
		},
		{
			name:        "duplicate registration",
			gateName:    "test.duplicate",
			description: "Duplicate test",
			setupGate:   "test.duplicate",
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()

			// Setup existing gate if needed
			if tt.setupGate != "" {
				reg.Register(tt.setupGate, "Setup gate", false)
			}

			if tt.shouldPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic but none occurred")
					}
				}()
			}

			gate := reg.Register(tt.gateName, tt.description, tt.defaultEnabled)

			if !tt.shouldPanic {
				if gate == nil {
					t.Fatal("Expected gate to be non-nil")
				}
				if gate.IsEnabled() != tt.defaultEnabled {
					t.Errorf("IsEnabled() = %v, want %v", gate.IsEnabled(), tt.defaultEnabled)
				}
			}
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	tests := []struct {
		name      string
		gateName  string
		setupGate string
		expectNil bool
	}{
		{
			name:      "existing gate",
			gateName:  "test.exists",
			setupGate: "test.exists",
			expectNil: false,
		},
		{
			name:      "non-existent gate",
			gateName:  "test.missing",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()

			var originalGate *Gate
			if tt.setupGate != "" {
				originalGate = reg.Register(tt.setupGate, "Test gate", false)
			}

			gate := reg.Get(tt.gateName)

			if tt.expectNil {
				if gate != nil {
					t.Error("Expected nil gate but got non-nil")
				}
			} else {
				if gate == nil {
					t.Fatal("Expected non-nil gate but got nil")
				}
				if gate != originalGate {
					t.Error("Expected Get to return the same gate instance")
				}
			}
		})
	}
}

func TestRegistry_Set(t *testing.T) {
	tests := []struct {
		name          string
		gateName      string
		setValue      bool
		setupGate     string
		expectError   bool
		errorContains string
	}{
		{
			name:      "set existing gate to true",
			gateName:  "test.settrue",
			setValue:  true,
			setupGate: "test.settrue",
		},
		{
			name:      "set existing gate to false",
			gateName:  "test.setfalse",
			setValue:  false,
			setupGate: "test.setfalse",
		},
		{
			name:          "set non-existent gate",
			gateName:      "test.missing",
			setValue:      true,
			expectError:   true,
			errorContains: "unknown feature gate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()

			if tt.setupGate != "" {
				reg.Register(tt.setupGate, "Test gate", !tt.setValue) // opposite of setValue
			}

			err := reg.Set(tt.gateName, tt.setValue)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error but got: %v", err)
				}
				gate := reg.Get(tt.gateName)
				if gate.IsEnabled() != tt.setValue {
					t.Errorf("Expected gate to be %v after Set(%v)", tt.setValue, tt.setValue)
				}
			}
		})
	}
}

func TestRegistry_VisitAll(t *testing.T) {
	tests := []struct {
		name      string
		gateNames []string
		expected  []string // expected order
	}{
		{
			name:      "single gate",
			gateNames: []string{"test.single"},
			expected:  []string{"test.single"},
		},
		{
			name:      "multiple gates in order",
			gateNames: []string{"a.gate", "b.gate", "c.gate"},
			expected:  []string{"a.gate", "b.gate", "c.gate"},
		},
		{
			name:      "multiple gates out of order",
			gateNames: []string{"c.gate", "a.gate", "b.gate"},
			expected:  []string{"a.gate", "b.gate", "c.gate"},
		},
		{
			name:      "empty registry",
			gateNames: []string{},
			expected:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()

			// Register gates
			for _, gateName := range tt.gateNames {
				reg.Register(gateName, "Test gate", false)
			}

			// Visit all and collect names
			var names []string
			reg.VisitAll(func(g *Gate) {
				names = append(names, g.Name)
			})

			if len(names) != len(tt.expected) {
				t.Fatalf("Expected %d gates, got %d", len(tt.expected), len(names))
			}

			for i, name := range names {
				if name != tt.expected[i] {
					t.Errorf("Expected gate %d to be %q, got %q", i, tt.expected[i], name)
				}
			}
		})
	}
}

func TestFlagValue_Set(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		setupGates    map[string]bool // gateName -> defaultEnabled
		expected      map[string]bool // gateName -> expectedEnabled
		expectError   bool
		errorContains string
	}{
		{
			name:       "single gate enable",
			input:      "test.gate1",
			setupGates: map[string]bool{"test.gate1": false},
			expected:   map[string]bool{"test.gate1": true},
		},
		{
			name:       "single gate disable",
			input:      "-test.gate1",
			setupGates: map[string]bool{"test.gate1": true},
			expected:   map[string]bool{"test.gate1": false},
		},
		{
			name:       "explicit enable with plus",
			input:      "+test.gate1",
			setupGates: map[string]bool{"test.gate1": false},
			expected:   map[string]bool{"test.gate1": true},
		},
		{
			name:  "multiple gates mixed",
			input: "test.gate1,-test.gate2,+test.gate3",
			setupGates: map[string]bool{
				"test.gate1": false,
				"test.gate2": true,
				"test.gate3": false,
			},
			expected: map[string]bool{
				"test.gate1": true,
				"test.gate2": false,
				"test.gate3": true,
			},
		},
		{
			name:        "empty input",
			input:       "",
			setupGates:  map[string]bool{"test.gate1": false},
			expected:    map[string]bool{"test.gate1": false},
			expectError: false,
		},
		{
			name:       "whitespace handling",
			input:      " test.gate1 , -test.gate2 ",
			setupGates: map[string]bool{"test.gate1": false, "test.gate2": true},
			expected:   map[string]bool{"test.gate1": true, "test.gate2": false},
		},
		{
			name:       "empty segments",
			input:      "test.gate1,,",
			setupGates: map[string]bool{"test.gate1": false},
			expected:   map[string]bool{"test.gate1": true},
		},
		{
			name:          "unknown gate",
			input:         "unknown.gate",
			setupGates:    map[string]bool{},
			expectError:   true,
			errorContains: "unknown feature gate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()

			// Setup gates
			for gateName, defaultEnabled := range tt.setupGates {
				reg.Register(gateName, "Test gate", defaultEnabled)
			}

			fv := &flagValue{reg: reg}
			err := fv.Set(tt.input)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error but got: %v", err)
				}

				// Check expected states
				for gateName, expectedEnabled := range tt.expected {
					gate := reg.Get(gateName)
					if gate == nil {
						t.Fatalf("Gate %q not found", gateName)
					}
					if gate.IsEnabled() != expectedEnabled {
						t.Errorf("Gate %q: expected %v, got %v", gateName, expectedEnabled, gate.IsEnabled())
					}
				}
			}
		})
	}
}

func TestFlagValue_String(t *testing.T) {
	tests := []struct {
		name        string
		setupGates  map[string]bool // gateName -> enabled
		contains    []string        // strings that should be in output
		notContains []string        // strings that should not be in output
	}{
		{
			name:        "single enabled gate",
			setupGates:  map[string]bool{"test.enabled": true},
			contains:    []string{"test.enabled"},
			notContains: []string{"-test.enabled"},
		},
		{
			name:        "single disabled gate",
			setupGates:  map[string]bool{"test.disabled": false},
			contains:    []string{"-test.disabled"},
			notContains: []string{"test.disabled,", ",test.disabled"},
		},
		{
			name: "mixed states",
			setupGates: map[string]bool{
				"test.enabled":  true,
				"test.disabled": false,
			},
			contains: []string{"test.enabled", "-test.disabled"},
		},
		{
			name:       "nil registry",
			setupGates: nil,
			contains:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reg *Registry
			if tt.setupGates != nil {
				reg = NewRegistry()
				for gateName, enabled := range tt.setupGates {
					gate := reg.Register(gateName, "Test gate", enabled)
					_ = gate // avoid unused variable
				}
			}

			fv := &flagValue{reg: reg}
			result := fv.String()

			for _, should := range tt.contains {
				if !strings.Contains(result, should) {
					t.Errorf("Expected result to contain %q, got %q", should, result)
				}
			}

			for _, shouldNot := range tt.notContains {
				if strings.Contains(result, shouldNot) {
					t.Errorf("Expected result to not contain %q, got %q", shouldNot, result)
				}
			}
		})
	}
}

func TestConcurrency(t *testing.T) {
	reg := NewRegistry()
	gate := reg.Register("test.concurrent", "Concurrent test", false)

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start multiple goroutines that read and write the gate state
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// Alternate between enabling and disabling
				enable := (id+j)%2 == 0
				_ = reg.Set("test.concurrent", enable) // Ignore error in test
				gate.IsEnabled()                       // Read the state
			}
		}(i)
	}

	wg.Wait()
	// If we get here without a race condition, the test passes
}

func TestGlobalRegistryFunctions(t *testing.T) {
	// Note: These tests modify the global registry and may affect other tests
	// In production, you might want to make the global registry replaceable

	t.Run("global functions work correctly", func(t *testing.T) {
		gateName := "test.global.function"

		// Clean up any existing gate
		globalRegistry.gates.Delete(gateName)

		gate := Register(gateName, "Global test", false)
		if gate == nil {
			t.Fatal("Expected non-nil gate from Register")
		}

		retrieved := Get(gateName)
		if retrieved != gate {
			t.Error("Expected Get to return same gate instance")
		}

		err := Set(gateName, true)
		if err != nil {
			t.Fatalf("Expected no error from Set, got: %v", err)
		}

		if !gate.IsEnabled() {
			t.Error("Expected gate to be enabled after global Set")
		}

		// Cleanup
		globalRegistry.gates.Delete(gateName)
	})
}

func TestRegisterFlags(t *testing.T) {
	reg := NewRegistry()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	reg.RegisterFlags(fs)

	flag := fs.Lookup("feature-gates")
	if flag == nil {
		t.Error("Expected feature-gates flag to be registered")
		return
	}

	if flag.Usage != featureGatesFlagDescription {
		t.Errorf("Expected flag description %q, got %q", featureGatesFlagDescription, flag.Usage)
	}
}

// Helper function to create bool pointers
func boolPtr(b bool) *bool {
	return &b
}

func ExampleRegistry() {
	// Create a new registry
	registry := NewRegistry()

	// Register some gates
	featureA := registry.Register("feature.a", "Enable feature A", false)
	featureB := registry.Register("feature.b", "Enable feature B", true)

	// Check if features are enabled
	fmt.Printf("Feature A enabled: %v\n", featureA.IsEnabled())
	fmt.Printf("Feature B enabled: %v\n", featureB.IsEnabled())

	// Enable feature A
	_ = registry.Set("feature.a", true) // Ignore error in example
	fmt.Printf("Feature A enabled after set: %v\n", featureA.IsEnabled())

	// List all gates
	registry.VisitAll(func(g *Gate) {
		status := "disabled"
		if g.IsEnabled() {
			status = "enabled"
		}
		fmt.Printf("Gate %s: %s (%s)\n", g.Name, status, g.Description)
	})

	// Output:
	// Feature A enabled: false
	// Feature B enabled: true
	// Feature A enabled after set: true
	// Gate feature.a: enabled (Enable feature A)
	// Gate feature.b: enabled (Enable feature B)
}
