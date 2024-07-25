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
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jmickey/telegraf-sidecar-operator/internal/metadata"
)

const (
	timeout   = time.Second * 10
	interval  = time.Millisecond * 250
	namespace = "default"
)

var (
	testCtx = context.Background()
)

var _ = Describe("Sidecar injector webhook", func() {
	When("Creating a pod under the defaulting webhook", func() {
		Context("And there is no telegraf annotation", func() {
			It("Should allow the pod admission and not inject the telegraf container", func() {
				podName := "no-annotations"
				pod := newTestPod(podName, nil)
				Expect(k8sClient.Create(testCtx, pod)).To(Succeed())

				pod = &corev1.Pod{}
				lookupKey := types.NamespacedName{Name: podName, Namespace: namespace}
				Expect(k8sClient.Get(testCtx, lookupKey, pod)).To(Succeed())
				for _, c := range pod.Spec.Containers {
					Expect(c.Name).NotTo(Equal(containerName))
				}

				cleanUpPod(pod.GetName())
			})
		})

		Context("And there is a telegraf annotation", func() {
			It("Should inject the telegraf container and config volume with default settings", func() {
				podName := "sidecar-defaults"

				pod := newTestPod(podName, map[string]string{
					metadata.TelegrafConfigClassAnnotation: "default",
				})
				Expect(k8sClient.Create(testCtx, pod)).To(Succeed())

				pod = &corev1.Pod{}
				lookupKey := types.NamespacedName{Name: podName, Namespace: namespace}
				Expect(k8sClient.Get(testCtx, lookupKey, pod)).To(Succeed())
				Expect(pod.GetLabels()[metadata.SidecarInjectedLabel]).To(Equal("true"))
				Expect(len(pod.Spec.Containers)).To(Equal(2))
				Expect(len(pod.Spec.Volumes)).To(Equal(1))

				var found bool
				for _, container := range pod.Spec.Containers {
					if container.Name == containerName {
						found = true
						Expect(container.Image).To(Equal(defaultTelegrafImage))
						Expect(container.Resources.Requests.Cpu().String()).To(Equal(defaultRequestsCPU))
						Expect(container.Resources.Requests.Memory().String()).To(Equal(defaultRequestsMemory))
						Expect(container.Resources.Limits.Cpu().String()).To(Equal(defaultLimitsCPU))
						Expect(container.Resources.Limits.Memory().String()).To(Equal(defaultLimitsMemory))
					}
				}
				Expect(found).To(BeTrue())

				cleanUpPod(pod.GetName())
			})

			It("Should truncate the secret name if the pod name is too long", func() {
				podName := "long-pod-name-5yzuhd7fknyq24yfy9kquaj0aknw9vvu1fynqn08"

				pod := newTestPod(podName, map[string]string{
					metadata.TelegrafConfigClassAnnotation: "default",
				})
				Expect(k8sClient.Create(testCtx, pod)).To(Succeed())

				var found bool
				for _, volume := range pod.Spec.Volumes {
					if volume.Name == "telegraf-config" {
						found = true
						Expect(len(volume.VolumeSource.Secret.SecretName)).To(Equal(63))
					}
				}
				Expect(found).To(BeTrue())

				cleanUpPod(pod.GetName())
			})

			It("Should correctly generate secret name using pod generate name", func() {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Labels:    nil,
						Annotations: map[string]string{
							metadata.TelegrafConfigClassAnnotation: "default",
						},
						GenerateName: "pod-generate-name-",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "test-container",
								Image: "ubuntu:latest",
							},
						},
					},
				}
				Expect(k8sClient.Create(testCtx, pod)).To(Succeed())

				var found bool
				for _, volume := range pod.Spec.Volumes {
					if volume.Name == "telegraf-config" {
						found = true
						Expect(strings.HasPrefix(volume.VolumeSource.Secret.SecretName, "telegraf-pod-generate-name-")).To(BeTrue())
					}
				}
				Expect(found).To(BeTrue())

				cleanUpPod(pod.GetName())
			})

			It("Should proceed with injection using defaults if resource annotations are invalid", func() {
				podName := "invalid-resource-values"
				invalidValue := "1000x"

				pod := newTestPod(podName, map[string]string{
					metadata.SidecarRequestsCPUAnnotation:    invalidValue,
					metadata.SidecarLimitsCPUAnnotation:      invalidValue,
					metadata.SidecarRequestsMemoryAnnotation: invalidValue,
					metadata.SidecarLimitsMemoryAnnotation:   invalidValue,
				})
				Expect(k8sClient.Create(testCtx, pod)).To(Succeed())

				pod = &corev1.Pod{}
				lookupKey := types.NamespacedName{Name: podName, Namespace: namespace}
				Expect(k8sClient.Get(testCtx, lookupKey, pod)).To(Succeed())
				Expect(len(pod.Spec.Containers)).To(Equal(2))
				Expect(len(pod.Spec.Volumes)).To(Equal(1))

				var found bool
				for _, container := range pod.Spec.Containers {
					if container.Name == containerName {
						found = true
						Expect(container.Resources.Requests.Cpu().String()).To(Equal(defaultRequestsCPU))
						Expect(container.Resources.Requests.Memory().String()).To(Equal(defaultRequestsMemory))
						Expect(container.Resources.Limits.Cpu().String()).To(Equal(defaultLimitsCPU))
						Expect(container.Resources.Limits.Memory().String()).To(Equal(defaultLimitsMemory))
					}
				}
				Expect(found).To(BeTrue())

				cleanUpPod(pod.GetName())
			})

			It("Should proceed overriding container resources with annotation values", func() {
				var (
					podName                = "sidecar-override-resources"
					overrideRequestsCPU    = "500m"
					overrideRequestsMemory = "500Mi"
					overrideLimitsCPU      = "800m"
					overrideLimitsMemory   = "800Mi"
				)

				pod := newTestPod(podName, map[string]string{
					metadata.SidecarRequestsCPUAnnotation:    overrideRequestsCPU,
					metadata.SidecarRequestsMemoryAnnotation: overrideRequestsMemory,
					metadata.SidecarLimitsCPUAnnotation:      overrideLimitsCPU,
					metadata.SidecarLimitsMemoryAnnotation:   overrideLimitsMemory,
				})
				Expect(k8sClient.Create(testCtx, pod)).To(Succeed())

				pod = &corev1.Pod{}
				lookupKey := types.NamespacedName{Name: podName, Namespace: namespace}
				Expect(k8sClient.Get(testCtx, lookupKey, pod)).To(Succeed())
				Expect(len(pod.Spec.Containers)).To(Equal(2))
				Expect(len(pod.Spec.Volumes)).To(Equal(1))

				var found bool
				for _, container := range pod.Spec.Containers {
					if container.Name == containerName {
						found = true
						Expect(container.Resources.Requests.Cpu().String()).To(Equal(overrideRequestsCPU))
						Expect(container.Resources.Requests.Memory().String()).To(Equal(overrideRequestsMemory))
						Expect(container.Resources.Limits.Cpu().String()).To(Equal(overrideLimitsCPU))
						Expect(container.Resources.Limits.Memory().String()).To(Equal(overrideLimitsMemory))
					}
				}
				Expect(found).To(BeTrue())

				cleanUpPod(pod.GetName())
			})

			It("Should add envFrom with secretRef if `secret-env` annotation is present", func() {
				podName := "sidecar-secret-env"
				secretName := "secret-env"

				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: namespace,
					},
					StringData: map[string]string{
						"TEST_VAR": "test_value",
					},
				}
				Expect(k8sClient.Create(testCtx, secret)).To(Succeed())

				pod := newTestPod(podName, map[string]string{
					metadata.SidecarEnvSecretAnnotation: secret.GetName(),
				})
				Expect(k8sClient.Create(testCtx, pod)).To(Succeed())

				pod = &corev1.Pod{}
				lookupKey := types.NamespacedName{Name: podName, Namespace: namespace}
				Expect(k8sClient.Get(testCtx, lookupKey, pod)).To(Succeed())
				Expect(len(pod.Spec.Containers)).To(Equal(2))

				var found bool
				for _, container := range pod.Spec.Containers {
					if container.Name == containerName {
						found = true
						Expect(container.EnvFrom[0].SecretRef.LocalObjectReference.Name).To(Equal(secretName))
					}
				}
				Expect(found).To(BeTrue())

				cleanUpPod(pod.GetName())
				cleanUpObject(&corev1.Secret{}, types.NamespacedName{Name: secretName, Namespace: namespace})
			})

			It("Should add envFrom with configMapRef if `configmap-env` annotation is present", func() {
				podName := "sidecar-configmap-env"
				configMapName := "configmap-env"

				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName,
						Namespace: namespace,
					},
					Data: map[string]string{
						"TEST_VAR": "test_value",
					},
				}
				Expect(k8sClient.Create(testCtx, configMap)).To(Succeed())

				pod := newTestPod(podName, map[string]string{
					metadata.SidecarEnvConfigMapAnnotation: configMap.GetName(),
				})
				Expect(k8sClient.Create(testCtx, pod)).To(Succeed())
				Expect(len(pod.Spec.Containers)).To(Equal(2))

				pod = &corev1.Pod{}
				lookupKey := types.NamespacedName{Name: podName, Namespace: namespace}
				Expect(k8sClient.Get(testCtx, lookupKey, pod)).To(Succeed())
				Expect(len(pod.Spec.Containers)).To(Equal(2))

				var found bool
				for _, container := range pod.Spec.Containers {
					if container.Name == containerName {
						found = true
						Expect(len(container.EnvFrom)).To(Equal(1))
						Expect(container.EnvFrom[0].ConfigMapRef.LocalObjectReference.Name).To(Equal(configMapName))
					}
				}
				Expect(found).To(BeTrue())

				cleanUpPod(pod.GetName())
				cleanUpObject(&corev1.ConfigMap{}, types.NamespacedName{Name: configMapName, Namespace: namespace})
			})

			It("Should add an environment variable literal value if `env-literal-` annotation exists", func() {
				podName := "sidecar-env-literal"
				envVarKey := "LITERAL_VAR"
				envVarValue := "literal_value"

				pod := newTestPod(podName, map[string]string{
					metadata.SidecarEnvLiteralPrefixAnnotation + envVarKey: envVarValue,
				})
				Expect(k8sClient.Create(testCtx, pod)).To(Succeed())

				pod = &corev1.Pod{}
				lookupKey := types.NamespacedName{Name: podName, Namespace: namespace}
				Expect(k8sClient.Get(testCtx, lookupKey, pod)).To(Succeed())
				Expect(len(pod.Spec.Containers)).To(Equal(2))

				var found bool
				for _, container := range pod.Spec.Containers {
					if container.Name == containerName {
						for _, envVar := range container.Env {
							if envVar.Name == envVarKey {
								found = true
								Expect(envVar.Value).To(Equal(envVarValue))
							}
						}
					}
				}
				Expect(found).To(BeTrue())

				cleanUpPod(pod.GetName())
			})

			It("Should add an environment variable FieldRef value if `/env-fieldref-` annotation exists", func() {
				podName := "sidecar-env-fieldref"

				pod := newTestPod(podName, map[string]string{
					metadata.SidecarEnvFieldRefPrefixAnnoation + "POD_IP": "status.podIP",
				})
				Expect(k8sClient.Create(testCtx, pod)).To(Succeed())

				pod = &corev1.Pod{}
				lookupKey := types.NamespacedName{Name: podName, Namespace: namespace}
				Expect(k8sClient.Get(testCtx, lookupKey, pod)).To(Succeed())
				Expect(len(pod.Spec.Containers)).To(Equal(2))

				var found bool
				for _, container := range pod.Spec.Containers {
					if container.Name == containerName {
						for _, envVar := range container.Env {
							if envVar.Name == "POD_IP" {
								found = true
								Expect(envVar.ValueFrom.FieldRef.FieldPath).To(Equal("status.podIP"))
							}
						}

					}
				}
				Expect(found).To(BeTrue())

				cleanUpPod(pod.GetName())
			})

			It("Should add an environment variable ConfigMapRef value if `/env-configmapkeyref-` annotation exists", func() {
				podName := "sidecar-env-configmapkeyref"
				configMapName := "configmap-env-configmapkeyref"
				envVarKey := "CONFIGMAP_VAR"
				envVarValue := "configmap_value"

				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName,
						Namespace: namespace,
					},
					Data: map[string]string{
						envVarKey: envVarValue,
					},
				}
				Expect(k8sClient.Create(testCtx, configMap)).To(Succeed())

				pod := newTestPod(podName, map[string]string{
					metadata.SidecarEnvConfigMapKeyRefPrefixAnnotation + envVarKey: configMapName + "." + envVarKey,
				})
				Expect(k8sClient.Create(testCtx, pod)).To(Succeed())

				pod = &corev1.Pod{}
				lookupKey := types.NamespacedName{Name: podName, Namespace: namespace}
				Expect(k8sClient.Get(testCtx, lookupKey, pod)).To(Succeed())
				Expect(len(pod.Spec.Containers)).To(Equal(2))

				var found bool
				for _, container := range pod.Spec.Containers {
					if container.Name == containerName {
						for _, envVar := range container.Env {
							if envVar.Name == envVarKey {
								found = true
								Expect(envVar.ValueFrom.ConfigMapKeyRef.Name).To(Equal(configMapName))
								Expect(envVar.ValueFrom.ConfigMapKeyRef.Key).To(Equal(envVarKey))
							}
						}
					}
				}
				Expect(found).To(BeTrue())

				cleanUpPod(pod.GetName())
				cleanUpObject(&corev1.ConfigMap{}, types.NamespacedName{Name: configMapName, Namespace: namespace})
			})

			It("Should add an environment variable ConfigMapRef value if `/env-secretkeyref-` annotation exists", func() {
				podName := "sidecar-env-secretkeyref"
				secretName := "secret-env-secretkeyref"
				envVarKey := "SECRET_VAR"
				envVarValue := "secret_value"

				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: namespace,
					},
					StringData: map[string]string{
						envVarKey: envVarValue,
					},
				}
				Expect(k8sClient.Create(testCtx, secret)).To(Succeed())

				pod := newTestPod(podName, map[string]string{
					metadata.SidecarEnvSecretKeyRefPrefixAnnotation + envVarKey: secretName + "." + envVarKey,
				})
				Expect(k8sClient.Create(testCtx, pod)).To(Succeed())

				pod = &corev1.Pod{}
				lookupKey := types.NamespacedName{Name: podName, Namespace: namespace}
				Expect(k8sClient.Get(testCtx, lookupKey, pod)).To(Succeed())
				Expect(len(pod.Spec.Containers)).To(Equal(2))

				var found bool
				for _, container := range pod.Spec.Containers {
					if container.Name == containerName {
						for _, envVar := range container.Env {
							if envVar.Name == envVarKey {
								found = true
								Expect(envVar.ValueFrom.SecretKeyRef.Name).To(Equal(secretName))
								Expect(envVar.ValueFrom.SecretKeyRef.Key).To(Equal(envVarKey))
							}
						}
					}
				}
				Expect(found).To(BeTrue())

				cleanUpPod(pod.GetName())
				cleanUpObject(&corev1.Secret{}, types.NamespacedName{Name: secretName, Namespace: namespace})
			})

			It("Should add additional volume mounts when `volume-mounts` annotation exists", func() {
				podName := "sidecar-volume-mounts"

				pod := newTestPod(podName, map[string]string{
					metadata.SidecarVolumeMountsAnnotation: "{ \"log-vol\": \"/var/log/app\" }",
				})
				pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
					Name: "log-vol",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				})
				Expect(k8sClient.Create(testCtx, pod)).To(Succeed())

				var found bool
				for _, container := range pod.Spec.Containers {
					if container.Name == containerName {
						for _, mount := range container.VolumeMounts {
							if mount.Name == "log-vol" {
								found = true
								Expect(mount.MountPath).To(Equal("/var/log/app"))
							}
						}
					}
				}
				Expect(found).To(BeTrue())

				cleanUpPod(pod.GetName())
			})
		})
	})
})

func newTestPod(name string, annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      nil,
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "ubuntu:latest",
				},
			},
		},
	}
}

func cleanUpPod(name string) {
	podKey := types.NamespacedName{Name: name, Namespace: namespace}
	Eventually(func() error {
		p := &corev1.Pod{}
		Expect(k8sClient.Get(testCtx, podKey, p)).Should(Succeed())
		return k8sClient.Delete(testCtx, p)
	}, timeout, interval).Should(Succeed())

	// Ensure delete has completed successfully
	Eventually(func() error {
		p := &corev1.Pod{}
		return k8sClient.Get(testCtx, podKey, p)
	}, timeout, interval).ShouldNot(Succeed())
}

func cleanUpObject(object client.Object, lookupKey types.NamespacedName) {
	Eventually(func() error {
		Expect(k8sClient.Get(testCtx, lookupKey, object)).Should(Succeed())
		return k8sClient.Delete(testCtx, object)
	}, timeout, interval).Should(Succeed())

	Eventually(func() error {
		return k8sClient.Get(testCtx, lookupKey, object)
	}, timeout, interval).ShouldNot(Succeed())
}
