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

	"github.com/BurntSushi/toml"
	"github.com/jmickey/telegraf-sidecar-operator/internal/classdata"
	"github.com/jmickey/telegraf-sidecar-operator/internal/metadata"
)

const (
	defaultInterval = 10 * time.Second
)

type annotationValues struct {
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
	globalTags     map[string]string
}

type prometheusInput struct {
	Urls          []string `toml:"urls"`
	Interval      string   `toml:"interval"`
	MetricVersion uint8    `toml:"metric_version"`
	Namepass      []string `toml:"namepass"`
}

type telegrafConfig struct {
	Agent       map[string]any    `toml:"agent"`
	Inputs      map[string]any    `toml:"inputs"`
	Outputs     map[string]any    `toml:"outputs"`
	Aggregators map[string]any    `toml:"aggregators,omitempty"`
	Processors  map[string]any    `toml:"processors,omitempty"`
	GlobalTags  map[string]string `toml:"global_tags"`
}

type rawInputs struct {
	Inputs map[string]any `toml:"inputs"`
}

// newAnnotationValues returns a pointer to a new annotationValues with default values initialized.
func newAnnotationValues(classDataHandler classdata.Handler, class string, enableInternal bool) *annotationValues {
	return &annotationValues{
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
		globalTags:       make(map[string]string),
	}
}

// applyAnnotationOverrides overrides the existing values of the
// telegrafConfig based on the provided pod annotations.
func (c *annotationValues) applyAnnotationOverrides(annotations map[string]string) error {
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
		c.namepass = strings.ReplaceAll(strings.Trim(override, "[]"), "'", "")
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

	c.globalTags = metadata.GetAnnotationsWithPrefix(annotations,
		metadata.TelegrafConfigGlobalTagLiteralPrefixAnnotation)

	if len(warnings) > 0 {
		errText := strings.Join(warnings, "; ")
		return errors.New(errText)
	}

	return nil
}

func (c *annotationValues) buildConfigData() (string, error) {
	cfg := telegrafConfig{
		Inputs:     map[string]any{},
		GlobalTags: map[string]string{},
	}

	classData, ok := c.classDataHandler.GetDataForClass(c.class)
	if !ok {
		return "", fmt.Errorf("failed to get class data: %s, class name doesn't exist", c.class)
	}

	if err := toml.Unmarshal(classData, &cfg); err != nil {
		return "", fmt.Errorf("failed to unmarshal class data, error: %w", err)
	}

	if len(c.ports) > 0 {
		var promCfg prometheusInput

		for _, port := range c.ports {
			promCfg.Urls = append(promCfg.Urls, fmt.Sprintf("%s://localhost:%d%s", c.scheme, port, c.metricsPath))
		}

		promCfg.Interval = c.interval.String()
		promCfg.MetricVersion = c.metricVersion

		if c.namepass != "" {
			namepass := strings.Split(c.namepass, ",")
			for _, item := range namepass {
				promCfg.Namepass = append(promCfg.Namepass, strings.TrimSpace(item))
			}
		}

		cfg.Inputs["prometheus"] = []prometheusInput{promCfg}
	}

	if c.enableInternal {
		cfg.Inputs["internal"] = []map[string]any{make(map[string]any)}
	}

	if c.rawInput != "" {
		rawInputs := rawInputs{}
		if err := toml.Unmarshal([]byte(strings.TrimSpace(c.rawInput)), &rawInputs); err != nil {
			return "", fmt.Errorf("failed to unmarshal raw input annotation data, error: %w", err)
		}

		for k, v := range rawInputs.Inputs {
			cfg.Inputs[k] = v
		}
	}

	if len(c.globalTags) > 0 {
		for k, v := range c.globalTags {
			cfg.GlobalTags[k] = v
		}
	}

	config, err := toml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal final toml output, error: %w", err)
	}

	return string(config), nil
}
