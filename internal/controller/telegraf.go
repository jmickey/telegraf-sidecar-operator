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

package controller

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jmickey/telegraf-sidecar-operator/internal/metadata"
)

type telegrafConfig struct {
	class          string
	metricsPath    string
	scheme         string
	namepass       string
	rawInput       string
	ports          []uint16
	interval       time.Duration
	metricsVersion uint8
	enableInternal bool
}

const (
	defaultInterval = 10 * time.Second
)

// newTelegrafConfig returns a pointer to a new telegrafConfig with
// default values initialized.
func newTelegrafConfig(class string, enableInternal bool) *telegrafConfig {
	return &telegrafConfig{
		class:          class,
		metricsPath:    "/metrics",
		ports:          []uint16{},
		scheme:         "http",
		metricsVersion: 1,
		namepass:       "",
		interval:       defaultInterval,
		enableInternal: enableInternal,
		rawInput:       "",
	}
}

// applyAnnotationOverrides overrides the existing values of the
// telegrafConfig based on the provided pod annotations.
func (c *telegrafConfig) applyAnnotationOverrides(annotations map[string]string) error {
	// Invalid annotation values should not stop the creation of the secret as this will
	// cause the pod creation to fail indefinitely. Instead collect warnings and use to
	// return a non-fatal error.
	var warnings []string

	if override, ok := annotations[metadata.TelegrafConfigClassAnnotation]; ok {
		c.class = override
	}

	if override, ok := annotations[metadata.TelegrafConfigMetricsPortAnnotation]; ok {
		warnings = append(warnings, fmt.Sprintf("Deprecated: %s will be removed in a future version, use %s instead.",
			metadata.TelegrafConfigMetricsPortAnnotation, metadata.TelegrafConfigMetricsPortsAnnotation))

		if port, err := strconv.ParseInt(override, 10, 16); err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to convert value: %s for %s to integer, error: %s",
				override, metadata.TelegrafConfigMetricsPortAnnotation, err.Error()))
		} else {
			c.ports = append(c.ports, uint16(port))
		}
	}

	if override, ok := annotations[metadata.TelegrafConfigMetricsPortsAnnotation]; ok {
		ports := strings.Split(override, ",")

		for _, portStr := range ports {
			if port, err := strconv.ParseInt(strings.TrimSpace(portStr), 10, 16); err != nil {
				warnings = append(warnings, fmt.Sprintf("failed to convert value: %s for %s to integer, error: %s",
					override, metadata.TelegrafConfigMetricsPortsAnnotation, err.Error()))
			} else {
				c.ports = append(c.ports, uint16(port))
			}
		}
	}

	if override, ok := annotations[metadata.TelegrafConfigMetricsPathAnnotation]; ok {
		c.metricsPath = override
	}

	if override, ok := annotations[metadata.TelegrafConfigMetricsSchemeAnnotation]; ok {
		c.scheme = override
	}

	if override, ok := annotations[metadata.TelegrafConfigMetricsNamepass]; ok {
		c.namepass = override
	}

	if override, ok := annotations[metadata.TelegrafConfigMetricVersionAnnotation]; ok {
		if ver, err := strconv.ParseInt(override, 10, 16); err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to convert value: %s for %s to integer, error: %s",
				override, metadata.TelegrafConfigMetricVersionAnnotation, err.Error()))
		} else {
			c.metricsVersion = uint8(ver)
		}
	}

	if override, ok := annotations[metadata.TelegrafConfigIntervalAnnotation]; ok {
		if interval, err := time.ParseDuration(override); err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to convert value: %s for %s to duration, error: %s",
				override, metadata.TelegrafConfigIntervalAnnotation, err.Error()))
		} else {
			c.interval = interval
		}
	}

	if override, ok := annotations[metadata.TelegrafConfigEnableInternalAnnotation]; ok {
		if override != "" {
			c.enableInternal = true
		}
	}

	if override, ok := annotations[metadata.TelegrafConfigRawInputAnnotation]; ok {
		c.rawInput = override
	}

	if len(warnings) > 0 {
		errText := strings.Join(warnings, "; ")
		return errors.New(errText)
	}

	return nil
}
