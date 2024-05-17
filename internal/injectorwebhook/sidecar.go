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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/jmickey/telegraf-sidecar-operator/internal/metadata"
)

const (
	containerName = "telegraf"
)

type containerConfig struct {
	image          string
	requestsCPU    resource.Quantity
	requestsMemory resource.Quantity
	limitsCPU      resource.Quantity
	limitsMemory   resource.Quantity

	log logr.Logger
}

func newContainerConfig(ctx context.Context, s *SidecarInjector, podName string) (*containerConfig, error) {
	var err error
	c := &containerConfig{
		image: s.TelegrafImage,
		log:   logf.FromContext(ctx, "logInstance", "injectorwebhook.sidecar", "pod", podName),
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
		// TODO(@jmickey): Support custom resources annotation, and parse correctly.
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
		Env: []corev1.EnvVar{
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
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      fmt.Sprintf("%s-config", containerName),
				MountPath: "/etc/telegraf",
			},
		},
	}

	return container
}

// func GetAnnotationsWithPrefix(annotations map[string]string, prefix string) map[string]string {}
