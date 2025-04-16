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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage/names"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/jmickey/telegraf-sidecar-operator/internal/metadata"
)

type SidecarInjector struct {
	SecretNamePrefix     string
	TelegrafImage        string
	RequestsCPU          string
	RequestsMemory       string
	LimitsCPU            string
	LimitsMemory         string
	EnableNativeSidecars bool
}

//+kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=ignore,groups=core,resources=pods,verbs=create;update,versions=v1,name=telegraf.mickey.dev,sideEffects=none,admissionReviewVersions=v1

func (s *SidecarInjector) SetupWithManager(mgr manager.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&corev1.Pod{}).
		WithDefaulter(s).
		Complete()
}

func (s *SidecarInjector) Default(ctx context.Context, obj runtime.Object) error {
	log := logf.FromContext(ctx).WithName("webhook.injector")

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected runtime.Object to be a Pod, got %T", obj)
	}
	log = log.WithValues("generateName", pod.GetGenerateName())

	if !s.shouldInjectContainer(pod) {
		log.V(2).Info("skipping pod, telegraf sidecar injector should not handle it")
		return nil
	}

	containerConfig, err := newContainerConfig(ctx, s, pod.GetName())
	if err != nil {
		log.Error(err, "failed to initialize container configuration")
		return err
	}
	containerConfig.applyAnnotationOverrides(pod.GetAnnotations())
	container := containerConfig.buildContainerSpec()
	if s.EnableNativeSidecars {
		policy := corev1.ContainerRestartPolicyAlways
		container.RestartPolicy = &policy
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, container)
	} else {
		pod.Spec.Containers = append(pod.Spec.Containers, container)
	}

	// If the pod does not have a name (the API server will generate one), then randomise
	// secret name using the name generation prefix and 5 random letters/numbers.
	// Communicate the required secret name to the controller via a label.
	secretName := s.generateSecretName(pod)
	telegrafVol := corev1.Volume{
		Name: "telegraf-config",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, telegrafVol)

	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[metadata.SidecarInjectedLabel] = "true"
	pod.Labels[metadata.SidecarSecretNameLabel] = secretName

	log.Info("successfully injected telegraf sidecar container into pod", "secretName", secretName)
	return nil
}

func (s *SidecarInjector) shouldInjectContainer(pod *corev1.Pod) bool {
	if s.hasTelegrafContainer(pod) {
		return false
	}

	for key := range pod.GetAnnotations() {
		if strings.Contains(key, metadata.Prefix) {
			return true
		}
	}

	return false
}

func (s *SidecarInjector) hasTelegrafContainer(pod *corev1.Pod) bool {
	if s.EnableNativeSidecars {
		for _, container := range pod.Spec.InitContainers {
			if strings.Contains(container.Name, "telegraf") {
				return true
			}
		}
	} else {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Name, "telegraf") {
				return true
			}
		}
	}

	return false
}

func (s *SidecarInjector) generateSecretName(pod *corev1.Pod) string {
	podName := pod.GetName()
	if podName == "" {
		podName = pod.GetGenerateName()
	}

	name := fmt.Sprintf("%s-%s-", s.SecretNamePrefix, strings.TrimSuffix(podName, "-"))
	if len(name) > 57 {
		name = name[:57] + "-"
	}

	return names.SimpleNameGenerator.GenerateName(name)
}
