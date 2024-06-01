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
	ControllerName               = "telegraf-sidecar-operator"
	SecretCreatedByLabelKey      = "app.kubernetes.io/created-by"
	SecretManagedByLabelKey      = "app.kubernetes.io/managed-by"
	SidecarInjectedLabel         = Prefix + "/injected"
	TelegrafSecretClassNameLabel = Prefix + "/class"
	TelegrafSecretPodLabel       = Prefix + "/pod"
)
