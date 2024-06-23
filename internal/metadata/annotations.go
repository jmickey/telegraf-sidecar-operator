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

package metadata

const (
	/*
	 * Sidecar Container Configuration
	 */

	// SidecarCustomImageAnnotation can be used to override
	// the telegraf sidecar image.
	SidecarCustomImageAnnotation = Prefix + "/image"

	// SidecarRequestsCPUAnnotation can be used to override the
	// CPU requests of the sidecar container.
	SidecarRequestsCPUAnnotation = Prefix + "/requests-cpu"

	// SidecarRequestsMemoryAnnotation can be used to override the
	// memory requests of the sidecar container.
	SidecarRequestsMemoryAnnotation = Prefix + "/requests-memory"

	// SidecarLimitsCPUAnnotation can be used to override the
	// CPU limits of the sidecar container.
	SidecarLimitsCPUAnnotation = Prefix + "/limits-cpu"

	// SidecarLimitsMemoryAnnotation can be used to override the
	// memory limits of the sidecar container.
	SidecarLimitsMemoryAnnotation = Prefix + "/limits-memory"

	// SidecarEnvSecretAnnotation can be used to mount a secret and
	// all its keys as environment variables in the sidecar container.
	SidecarEnvSecretAnnotation = Prefix + "/secret-env"

	// SidecarEnvConfigMapAnnotation can be used to mount a ConfigMap
	// and all its keys as environment variables in the sidecar container.
	SidecarEnvConfigMapAnnotation = Prefix + "/configmap-env"

	// SidecarVolumeMountsAnnotation can be used to mount additional
	// volumes into the sidecar container. Must be in the format:
	// `{ "<volumeName>": "<mountPath>" }`
	SidecarVolumeMountsAnnotation = Prefix + "/volume-mounts"

	/*
	 *  Sidecar Container Prefix Annotations
	 */

	// SidecarEnvLiteralPrefixAnnotation can be used to add a literal
	// value as an environment variable to the sidecar container.
	SidecarEnvLiteralPrefixAnnotation = Prefix + "/env-literal-"

	// SidecarEnvFieldRefPrefixAnnoation can be used to add a Kubernetes
	// downstream API FieldRef as an environment variable to the sidecar
	// container.
	SidecarEnvFieldRefPrefixAnnoation = Prefix + "/env-fieldref-"

	// SidecarEnvSecretKeyRefPrefixAnnotation can be used to add a Secret key
	// value as an environment variable to the sidecar container.
	SidecarEnvSecretKeyRefPrefixAnnotation = Prefix + "/env-secretkeyref-"

	// SidecarEnvConfigMapKeyRefPrefixAnnotation can be used to add a ConfigMap key
	// value as an environment variable to the sidecar container.
	SidecarEnvConfigMapKeyRefPrefixAnnotation = Prefix + "/env-configmapkeyref-"

	/*
	 * Telagraf Configuration Annotations
	 */

	// TelegrafConfigClassAnnotation specifies which telegraf
	// config class to use. Classes are configured in the operator.
	TelegrafConfigClassAnnotation = Prefix + "/class"

	// TelegrafConfigMetricsPortAnnotation can be used to configure
	// the port to scrape with the Prometheus input plugin. Will be
	// appended to values of telegraf.influxdata.com/ports if both
	// are specified.
	//
	// Deprecated: This annotation will be removed in future versions,
	// use telegraf.influxdata.com/ports instead.
	TelegrafConfigMetricsPortAnnotation = Prefix + "/port"

	// TelegrafConfigMetricsPortsAnnotation can be used to configure
	// multiple ports to be scraped by the Prometheus input plugin.
	// Must be a string of comma separated values.
	TelegrafConfigMetricsPortsAnnotation = Prefix + "/ports"

	// TelegrafConfigMetricsPathAnnotation can be used to override
	// the HTTP path to be scraped by the Prometheus input plugin.
	// Applies to all ports if multiple are provided.
	// Default: "/metrics"
	TelegrafConfigMetricsPathAnnotation = Prefix + "/path"

	// TelegrafConfigMetricsSchemeAnnotation can be used to override
	// the request scheme when scraping metrics with the Prometheus
	// input plugin. Valid values are [ "http", "https" ].
	// Default: "http"
	TelegrafConfigMetricsSchemeAnnotation = Prefix + "/scheme"

	// TelegrafConfigMetricVersionAnnotation can be used to override
	// which metrics parsing version to use. Valid values are [ "1", "2"].
	// Default: "1"
	TelegrafConfigMetricVersionAnnotation = Prefix + "/metric-version"

	// TelegrafConfigMetricsNamepass can be used to configure
	// the namepass setting for the Prometheus input plugin.
	// Namepass accepts an array of glob pattern strings.
	// Only metrics whose measurement name matches a pattern
	// in this list are emitted.
	//
	// Annotation value must be specified as a comma-separated string, e.g.
	// "metric1, metric2"
	TelegrafConfigMetricsNamepass = Prefix + "/namepass"

	// TelegrafConfigIntervalAnnotation can be used to configure
	// the scraping interval. Value must be a value to Go style
	// duration, e.g. 10s, 30s, 1m.
	// Default: 10s
	TelegrafConfigIntervalAnnotation = Prefix + "/interval"

	// TelegrafConfigRawInputAnnotation can be used to configure
	// a raw telegraf input TOML block. Can be provided as a multiline
	// block of raw TOML configuration.
	// e.g.
	// telegraf.influxdata.com/inputs: |+
	//   [[inputs.redis]]
	//     servers = ["tcp://localhost:6379"]
	TelegrafConfigRawInputAnnotation = Prefix + "/inputs"

	// TelegrafConfigEnableInternalAnnotation enables the "internal"
	// telegraf plugin. Any non-empty string value is accepted.
	TelegrafConfigEnableInternalAnnotation = Prefix + "/internal"

	/*
	 * Telagraf Configuration Prefix Annotations
	 */

	// TelegrafConfigGlobalTagLiteralPrefixAnnotation can be used to a literal value
	// to the global_tags in the telegraf configuration.
	TelegrafConfigGlobalTagLiteralPrefixAnnotation = Prefix + "/global-tag-literal-"
)
