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

package k8s

const (
	/*
	 * Telagraf Sidecar Container Configuration
	 */

	// SidecarCustomImageAnnotation can be used to override
	// the telegraf sidecar image
	SidecarCustomImageAnnotation = Prefix + "/image"

	// SidecarRequestsCPUAnnotation can be used to override the
	// CPU requests of the sidecar container
	SidecarRequestsCPUAnnotation = Prefix + "/requests-cpu"

	// SidecarRequestsMemoryAnnotation can be used to override the
	// memory requests of the sidecar container
	SidecarRequestsMemoryAnnotation = Prefix + "/requests-memory"

	// SidecarLimitsCPUAnnotation can be used to override the
	// CPU limits of the sidecar container
	SidecarLimitsCPUAnnotation = Prefix + "/limits-cpu"

	// SidecarLimitsMemoryAnnotation can be used to override the
	// memory limits of the sidecar container
	SidecarLimitsMemoryAnnotation = Prefix + "/limits-memory"

	/*
	 * Telagraf Configuration
	 */

	// TelegrafConfigClassAnnotation
	TelegrafConfigClassAnnotation = Prefix + "/class"

	// TelegrafConfigMetricsPortAnnotation
	TelegrafConfigMetricsPortAnnotation = Prefix + "/port"

	// TelegrafConfigMetricsPortsAnnotation
	TelegrafConfigMetricsPortsAnnotation = Prefix + "/ports"

	// TelegrafConfigMetricsPathAnnotation
	TelegrafConfigMetricsPathAnnotation = Prefix + "/path"

	// TelegrafConfigMetricsSchemeAnnotation
	TelegrafConfigMetricsSchemeAnnotation = Prefix + "/scheme"

	// TelegrafConfigMetricVersionAnnotation
	TelegrafConfigMetricVersionAnnotation = Prefix + "/metric-version"

	// TelegrafConfigMetricsNamepass
	TelegrafConfigMetricsNamepass = Prefix + "/namepass"

	// TelegrafConfigIntervalAnnotation
	TelegrafConfigIntervalAnnotation = Prefix + "/interval"

	// TelegrafConfigRawInputAnnotation
	TelegrafConfigRawInputAnnotation = Prefix + "/inputs"

	// TelegrafConfigEnableInternalAnnotation
	TelegrafConfigEnableInternalAnnotation = Prefix + "/internal"
)
