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
	"strings"

	"go.uber.org/multierr"
)

const (
	featureGatesFlag            = "feature-gates"
	featureGatesFlagDescription = "Comma-delimited list of feature gate identifiers. Prefix with '-' to disable the feature. '+' or no prefix will enable the feature."
)

// RegisterFlags adds feature gate flags to the provided FlagSet.
func (r *Registry) RegisterFlags(flagSet *flag.FlagSet) {
	flagSet.Var(&flagValue{reg: r}, featureGatesFlag, featureGatesFlagDescription)
}

// RegisterFlags adds feature gate flags to the provided FlagSet using the global registry.
func RegisterFlags(flagSet *flag.FlagSet) {
	globalRegistry.RegisterFlags(flagSet)
}

// flagValue implements the flag.Value interface and applies feature gate settings to a Registry.
type flagValue struct {
	reg *Registry
}

// String returns the current state of all feature gates as a comma-separated string.
func (f *flagValue) String() string {
	// Handle nil registry (can happen during flag initialization)
	if f.reg == nil {
		return ""
	}

	var ids []string
	f.reg.VisitAll(func(g *Gate) {
		id := g.Name
		if !g.IsEnabled() {
			id = "-" + id
		}
		ids = append(ids, id)
	})
	return strings.Join(ids, ",")
}

// Set parses and applies feature gate settings from a comma-separated string.
// Format: "gate1,gate2,-gate3,+gate4"
// Supported prefixes:
//   - No prefix or '+': enable the gate
//   - '-': disable the gate
func (f *flagValue) Set(s string) error {
	if s == "" {
		return nil
	}

	var errs error
	gates := strings.Split(s, ",")

	for _, gateStr := range gates {
		gateStr = strings.TrimSpace(gateStr)
		if gateStr == "" {
			continue
		}

		// Default to enabled
		enabled := true
		name := gateStr

		// Check for explicit enable/disable prefixes
		switch gateStr[0] {
		case '-':
			enabled = false
			name = gateStr[1:]
		case '+':
			enabled = true
			name = gateStr[1:]
		}

		// Apply the setting
		if err := f.reg.Set(name, enabled); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return errs
}
