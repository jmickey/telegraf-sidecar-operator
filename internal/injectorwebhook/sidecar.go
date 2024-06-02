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

package injectorwebhook

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/jmickey/telegraf-sidecar-operator/internal/metadata"
)

const (
	containerName = "telegraf"
)

// TODO(@jmickey): find a better way of surfacing
// non-fatal errors than embedding a logger here.
type containerConfig struct {
	requestsCPU    resource.Quantity
	requestsMemory resource.Quantity
	limitsCPU      resource.Quantity
	limitsMemory   resource.Quantity
	log            logr.Logger
	image          string
	env            []corev1.EnvVar
	envFrom        []corev1.EnvFromSource
}

func newContainerConfig(ctx context.Context, s *SidecarInjector, podName string) (*containerConfig, error) {
	var err error
	c := &containerConfig{
		image: s.TelegrafImage,
		log:   logf.FromContext(ctx, "logInstance", "injectorwebhook.sidecar", "pod", podName),
	}

	// Setup default environment variables for the sidecar
	c.env = []corev1.EnvVar{
		{
			Name: "PODNAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: "NODENAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		{
			Name: "NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
	}

	if c.requestsCPU, err = resource.ParseQuantity(s.RequestsCPU); err != nil {
		return nil, fmt.Errorf("failed to parse CPU requests with value: %s, error: %w", s.RequestsCPU, err)
	}
	if c.requestsMemory, err = resource.ParseQuantity(s.RequestsMemory); err != nil {
		return nil, fmt.Errorf("failed to parse memory requests with value: %s, error: %w", s.RequestsMemory, err)
	}
	if c.limitsCPU, err = resource.ParseQuantity(s.LimitsCPU); err != nil {
		return nil, fmt.Errorf("failed to parse CPU limits with value: %s, error: %w", s.LimitsCPU, err)
	}
	if c.limitsMemory, err = resource.ParseQuantity(s.LimitsMemory); err != nil {
		return nil, fmt.Errorf("failed to parse memory limits with value: %s, error: %w", s.LimitsMemory, err)
	}

	return c, nil
}

func (c *containerConfig) applyAnnotationOverrides(annotations map[string]string) {
	if override, ok := annotations[metadata.SidecarCustomImageAnnotation]; ok {
		c.image = override
	}

	if override, ok := annotations[metadata.SidecarRequestsCPUAnnotation]; ok {
		q, err := resource.ParseQuantity(override)
		if err != nil {
			c.log.Error(err,
				"failed to parse override resource value for requests.CPU, using default value",
				"invalidValue", override)
		} else {
			c.requestsCPU = q
		}
	}

	if override, ok := annotations[metadata.SidecarRequestsMemoryAnnotation]; ok {
		q, err := resource.ParseQuantity(override)
		if err != nil {
			c.log.Error(err,
				"failed to parse override resource value for requests.memory, using default value",
				"invalidValue", override)
		} else {
			c.requestsMemory = q
		}
	}

	if override, ok := annotations[metadata.SidecarLimitsCPUAnnotation]; ok {
		q, err := resource.ParseQuantity(override)
		if err != nil {
			c.log.Error(err,
				"failed to parse override resource value for limits.CPU, using default value",
				"invalidValue", override)
		} else {
			c.limitsCPU = q
		}
	}

	if override, ok := annotations[metadata.SidecarLimitsMemoryAnnotation]; ok {
		q, err := resource.ParseQuantity(override)
		if err != nil {
			c.log.Error(err,
				"failed to parse override resource value for limits.memory, using default value",
				"invalidValue", override)
		} else {
			c.limitsMemory = q
		}
	}

	if secretEnv, ok := annotations[metadata.SidecarEnvSecretAnnotation]; ok {
		c.envFrom = append(c.envFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretEnv,
				},
				Optional: func(x bool) *bool { return &x }(true),
			},
		})
	}

	envLiterals := metadata.GetAnnotationsWithPrefix(annotations, metadata.SidecarEnvLiteralPrefixAnnotation)
	for name, value := range envLiterals {
		c.env = append(c.env, corev1.EnvVar{
			Name:  name,
			Value: value,
		})
	}

	envFieldRefs := metadata.GetAnnotationsWithPrefix(annotations, metadata.SidecarEnvFieldRefPrefixAnnoation)
	for name, value := range envFieldRefs {
		c.env = append(c.env, corev1.EnvVar{
			Name: name,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: value,
				},
			},
		})
	}

	envConfigMapKeyRefs := metadata.GetAnnotationsWithPrefix(annotations,
		metadata.SidecarEnvConfigMapKeyRefPrefixAnnotation)
	for name, value := range envConfigMapKeyRefs {
		selector := strings.SplitN(value, ".", 2)
		if len(selector) == 2 {
			c.env = append(c.env, corev1.EnvVar{
				Name: name,
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: selector[0],
						},
						Key: selector[1],
					},
				},
			})
		} else {
			c.log.Info("failed to parse configmapref for %s, invalid value: %s", name, value)
		}
	}

	envSecretKeyRefs := metadata.GetAnnotationsWithPrefix(annotations, metadata.SidecarEnvConfigMapKeyRefPrefixAnnotation)
	for name, value := range envSecretKeyRefs {
		selector := strings.SplitN(value, ".", 2)
		if len(selector) == 2 {
			c.env = append(c.env, corev1.EnvVar{
				Name: name,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: selector[0],
						},
						Key: selector[1],
					},
				},
			})
		} else {
			c.log.Info("failed to parse secretkeyref for %s, invalid value: %s", name, value)
		}
	}
}

func (c *containerConfig) buildContainerSpec() corev1.Container {
	container := corev1.Container{
		Name:  containerName,
		Image: c.image,
		Command: []string{
			"telegraf",
			"--config",
			"/etc/telegraf/telegraf.conf",
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    c.requestsCPU,
				corev1.ResourceMemory: c.requestsMemory,
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    c.limitsCPU,
				corev1.ResourceMemory: c.limitsMemory,
			},
		},
		Env:     c.env,
		EnvFrom: c.envFrom,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      fmt.Sprintf("%s-config", containerName),
				MountPath: "/etc/telegraf",
			},
		},
	}

	return container
}
