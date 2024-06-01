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

	"github.com/influxdata/toml"
	"github.com/jmickey/telegraf-sidecar-operator/internal/classdata"
	"github.com/jmickey/telegraf-sidecar-operator/internal/metadata"
)

type telegrafConfig struct {
	classDataHandler classdata.Handler

	class          string
	metricsPath    string
	scheme         string
	namepass       string
	rawInput       string
	ports          []uint16
	interval       time.Duration
	metricVersion  uint8
	enableInternal bool
}

const (
	defaultInterval = 10 * time.Second
)

// newTelegrafConfig returns a pointer to a new telegrafConfig with
// default values initialized.
func newTelegrafConfig(classDataHandler classdata.Handler, class string, enableInternal bool) *telegrafConfig {
	return &telegrafConfig{
		classDataHandler: classDataHandler,
		class:            class,
		metricsPath:      "/metrics",
		ports:            []uint16{},
		scheme:           "http",
		metricVersion:    1,
		namepass:         "",
		interval:         defaultInterval,
		enableInternal:   enableInternal,
		rawInput:         "",
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

	//nolint:staticcheck
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
			c.metricVersion = uint8(ver)
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

func (c *telegrafConfig) buildConfigData() (string, error) {
	var config string

	classData, ok := c.classDataHandler.GetDataForClass(c.class)
	if !ok {
		return "", fmt.Errorf("failed to get class data: %s, class name doesn't exist", c.class)
	}

	config = fmt.Sprintf("%s\n\n", strings.TrimSpace(classData))

	if len(c.ports) > 0 {
		promConfig := "[[inputs.prometheus]]\n"
		var urls []string

		for _, port := range c.ports {
			urls = append(urls, fmt.Sprintf("\"%s://localhost:%d%s\"", c.scheme, port, c.metricsPath))
		}

		promConfig = fmt.Sprintf("%s  urls = [%s]\n", promConfig, strings.Join(urls, ", "))
		promConfig = fmt.Sprintf("%s  interval = \"%s\"\n", promConfig, c.interval.String())

		if c.metricVersion > 0 {
			promConfig = fmt.Sprintf("%s  metric_version = %d\n", promConfig, c.metricVersion)
		}

		if c.namepass != "" {
			promConfig = fmt.Sprintf("%s  namepass = %s\n", promConfig, c.namepass)
		}

		config = fmt.Sprintf("%s%s\n", config, promConfig)
	}

	if c.enableInternal {
		config = fmt.Sprintf("%s[[inputs.internal]]\n\n", config)
	}

	if c.rawInput != "" {
		config = fmt.Sprintf("%s%s", config, c.rawInput)
	}

	config = fmt.Sprintf("%s\n", strings.TrimSpace(config))

	if _, err := toml.Parse([]byte(config)); err != nil {
		return "", fmt.Errorf("failed to parse the final telegaf configuration: %w", err)
	}

	return config, nil
}
