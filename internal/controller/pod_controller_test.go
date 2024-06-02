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
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jmickey/telegraf-sidecar-operator/internal/metadata"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	timeout   = time.Second * 10
	duration  = time.Second * 5
	interval  = time.Millisecond * 250
	namespace = "default"
)

var (
	testCtx = context.Background()
)

var _ = Describe("Pod Controller", func() {
	When("A pod is created", func() {
		Context("And there is no telegraf.influxdata.com/injected label", func() {
			It("should not reconcile the object", func() {
				pod := newTestPod("no-label", nil, nil)
				Expect(k8sClient.Create(testCtx, pod)).Should(Succeed())

				secretKey := types.NamespacedName{
					Name:      fmt.Sprintf("telegraf-config-%s", pod.GetName()),
					Namespace: namespace,
				}

				By("Expecting the secret to not be created with NotFound error")
				Consistently(func() bool {
					secret := &corev1.Secret{}
					err := k8sClient.Get(testCtx, secretKey, secret)
					if err != nil {
						return apierrors.IsNotFound(err)
					}
					return false
				}, duration, interval).Should(BeTrue())

				cleanUpPod(pod.GetName())
			})
		})

		Context("And the telegraf.influxdata.com/injected label exists", func() {
			Context("And the telegraf config secret already exists", func() {
				It("Should skip further reconciliation", func() {
					secret := &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "telegraf-config-secret-already-exists",
							Namespace: namespace,
						},
					}
					Eventually(func() error {
						err := k8sClient.Create(testCtx, secret)
						Expect(err).Should(BeNil())
						s := &corev1.Secret{}
						key := types.NamespacedName{Name: secret.GetName(), Namespace: secret.GetNamespace()}
						return k8sClient.Get(testCtx, key, s)
					}, timeout, interval).Should(Succeed())

					pod := newTestPod(
						"secret-already-exists",
						map[string]string{metadata.SidecarInjectedLabel: "true"},
						map[string]string{metadata.TelegrafConfigMetricsPortsAnnotation: "8080"},
					)
					Expect(k8sClient.Create(testCtx, pod)).Should(Succeed())

					secretKey := types.NamespacedName{
						Name:      secret.GetName(),
						Namespace: namespace,
					}
					By("Confirming that the pod hasn't been updated")
					Eventually(func() bool {
						secret := &corev1.Secret{}
						Expect(k8sClient.Get(testCtx, secretKey, secret)).Should(Succeed())
						_, ok := secret.GetLabels()[metadata.SecretManagedByLabelKey]
						return ok
					}, timeout, interval).Should(BeFalse())

					cleanUpPod(pod.GetName())
					cleanUpSecret(secret.GetName())
				})
			})

			Context("And the telegraf secret does not already exist", func() {
				It("Should reconcile successfully with minimum configuration", func() {
					pod := newTestPod(
						"minimum-config",
						map[string]string{metadata.SidecarInjectedLabel: "true"},
						map[string]string{metadata.TelegrafSecretClassNameLabel: "testclass"},
					)
					Expect(k8sClient.Create(testCtx, pod)).Should(Succeed())
					Eventually(func() error {
						p := &corev1.Pod{}
						key := types.NamespacedName{Name: pod.GetName(), Namespace: pod.GetNamespace()}
						return k8sClient.Get(testCtx, key, p)
					}, timeout, interval).Should(Succeed())

					secret := &corev1.Secret{}
					Eventually(func() error {
						key := types.NamespacedName{
							Name:      fmt.Sprintf("telegraf-config-%s", pod.GetName()),
							Namespace: pod.GetNamespace(),
						}
						return k8sClient.Get(testCtx, key, secret)
					}, timeout, interval).Should(Succeed())

					val, ok := secret.GetLabels()[metadata.TelegrafSecretClassNameLabel]
					Expect(ok).To(BeTrue())
					Expect(val).To(Equal("testclass"))

					val, ok = secret.GetLabels()[metadata.TelegrafSecretPodLabel]
					Expect(ok).To(BeTrue())
					Expect(val).To(Equal(pod.GetName()))
				})

				It("Should reconcile successfully with single port annotation", func() {
					pod := newTestPod(
						"single-port-annotation",
						map[string]string{metadata.SidecarInjectedLabel: "true"},
						map[string]string{metadata.TelegrafConfigMetricsPortsAnnotation: "8080"},
					)
					Expect(k8sClient.Create(testCtx, pod)).Should(Succeed())
					Eventually(func() error {
						p := &corev1.Pod{}
						key := types.NamespacedName{Name: pod.GetName(), Namespace: pod.GetNamespace()}
						return k8sClient.Get(testCtx, key, p)
					}, timeout, interval).Should(Succeed())

					secret := &corev1.Secret{}
					Eventually(func() error {
						key := types.NamespacedName{
							Name:      fmt.Sprintf("telegraf-config-%s", pod.GetName()),
							Namespace: pod.GetNamespace(),
						}
						return k8sClient.Get(testCtx, key, secret)
					}, timeout, interval).Should(Succeed())

					fixture, err := os.ReadFile("../../config/testdata/fixtures/single-port.toml")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(string(secret.Data["telegraf.conf"])).Should(Equal(string(fixture)))

					cleanUpPod(pod.GetName())
					cleanUpSecret(secret.GetName())
				})

				It("Should complete the reconciliation successfully with multiple ports annotation", func() {
					pod := newTestPod(
						"multiple-ports-annotation",
						map[string]string{metadata.SidecarInjectedLabel: "true"},
						map[string]string{metadata.TelegrafConfigMetricsPortsAnnotation: "8080, 9090, 9091"},
					)
					Expect(k8sClient.Create(testCtx, pod)).Should(Succeed())
					Eventually(func() error {
						p := &corev1.Pod{}
						key := types.NamespacedName{Name: pod.GetName(), Namespace: pod.GetNamespace()}
						return k8sClient.Get(testCtx, key, p)
					}, timeout, interval).Should(Succeed())

					secret := &corev1.Secret{}
					Eventually(func() error {
						key := types.NamespacedName{
							Name:      fmt.Sprintf("telegraf-config-%s", pod.GetName()),
							Namespace: pod.GetNamespace(),
						}
						return k8sClient.Get(testCtx, key, secret)
					}, timeout, interval).Should(Succeed())

					fixture, err := os.ReadFile("../../config/testdata/fixtures/multiple-ports.toml")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(string(secret.Data["telegraf.conf"])).Should(Equal(string(fixture)))

					cleanUpPod(pod.GetName())
					cleanUpSecret(secret.GetName())
				})

				It("Should reconcile successfully with prometheus plugin overrides", func() {
					pod := newTestPod(
						"prometheus-plugin-overrides",
						map[string]string{metadata.SidecarInjectedLabel: "true"},
						map[string]string{
							metadata.TelegrafConfigMetricsPortsAnnotation:  "8080",
							metadata.TelegrafConfigMetricsSchemeAnnotation: "https",
							metadata.TelegrafConfigMetricsPathAnnotation:   "/test-path",
							metadata.TelegrafConfigIntervalAnnotation:      "30s",
							metadata.TelegrafConfigMetricVersionAnnotation: "2",
							metadata.TelegrafConfigMetricsNamepass:         "['metric1']",
						},
					)
					Expect(k8sClient.Create(testCtx, pod)).Should(Succeed())
					Eventually(func() error {
						p := &corev1.Pod{}
						key := types.NamespacedName{Name: pod.GetName(), Namespace: pod.GetNamespace()}
						return k8sClient.Get(testCtx, key, p)
					}, timeout, interval).Should(Succeed())

					secret := &corev1.Secret{}
					Eventually(func() error {
						key := types.NamespacedName{
							Name:      fmt.Sprintf("telegraf-config-%s", pod.GetName()),
							Namespace: pod.GetNamespace(),
						}
						return k8sClient.Get(testCtx, key, secret)
					}, timeout, interval).Should(Succeed())

					fixture, err := os.ReadFile("../../config/testdata/fixtures/prometheus-overrides.toml")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(string(secret.Data["telegraf.conf"])).Should(Equal(string(fixture)))

					cleanUpPod(pod.GetName())
					cleanUpSecret(secret.GetName())
				})

				It("Should reconcile successfully with raw input annotation", func() {
					pod := newTestPod(
						"raw-input",
						map[string]string{metadata.SidecarInjectedLabel: "true"},
						map[string]string{
							metadata.TelegrafConfigRawInputAnnotation: `
[[inputs.influxdb_listener]]
  service_address = ":8186"
  max_body_size = 0
  max_line_size = 0
  read_timeout = "8s"
  write_timeout = "8s"
[inputs.influxdb_listener.tags]
  collectiontype = "application"
`,
						},
					)
					Expect(k8sClient.Create(testCtx, pod)).Should(Succeed())
					Eventually(func() error {
						p := &corev1.Pod{}
						key := types.NamespacedName{Name: pod.GetName(), Namespace: pod.GetNamespace()}
						return k8sClient.Get(testCtx, key, p)
					}, timeout, interval).Should(Succeed())

					secret := &corev1.Secret{}
					Eventually(func() error {
						key := types.NamespacedName{
							Name:      fmt.Sprintf("telegraf-config-%s", pod.GetName()),
							Namespace: pod.GetNamespace(),
						}
						return k8sClient.Get(testCtx, key, secret)
					}, timeout, interval).Should(Succeed())

					fixture, err := os.ReadFile("../../config/testdata/fixtures/raw-input.toml")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(string(secret.Data["telegraf.conf"])).Should(Equal(string(fixture)))

					cleanUpPod(pod.GetName())
					cleanUpSecret(secret.GetName())
				})

				It("Should reconcile successfully with internal plugin enabled", func() {
					pod := newTestPod(
						"internal-plugin-enabled",
						map[string]string{metadata.SidecarInjectedLabel: "true"},
						map[string]string{
							metadata.TelegrafConfigEnableInternalAnnotation: "yes",
						},
					)
					Expect(k8sClient.Create(testCtx, pod)).Should(Succeed())
					Eventually(func() error {
						p := &corev1.Pod{}
						key := types.NamespacedName{Name: pod.GetName(), Namespace: pod.GetNamespace()}
						return k8sClient.Get(testCtx, key, p)
					}, timeout, interval).Should(Succeed())

					secret := &corev1.Secret{}
					Eventually(func() error {
						key := types.NamespacedName{
							Name:      fmt.Sprintf("telegraf-config-%s", pod.GetName()),
							Namespace: pod.GetNamespace(),
						}
						return k8sClient.Get(testCtx, key, secret)
					}, timeout, interval).Should(Succeed())

					fixture, err := os.ReadFile("../../config/testdata/fixtures/internal-plugin.toml")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(string(secret.Data["telegraf.conf"])).Should(Equal(string(fixture)))

					cleanUpPod(pod.GetName())
					cleanUpSecret(secret.GetName())
				})

				It("Should reconcile successfully with global tags annotation", func() {
					pod := newTestPod(
						"global-tags",
						map[string]string{metadata.SidecarInjectedLabel: "true"},
						map[string]string{
							metadata.TelegrafConfigGlobalTagLiteralPrefixAnnotation + "my_tag": "my-value",
						},
					)
					Expect(k8sClient.Create(testCtx, pod)).Should(Succeed())
					Eventually(func() error {
						p := &corev1.Pod{}
						key := types.NamespacedName{Name: pod.GetName(), Namespace: pod.GetNamespace()}
						return k8sClient.Get(testCtx, key, p)
					}, timeout, interval).Should(Succeed())

					secret := &corev1.Secret{}
					Eventually(func() error {
						key := types.NamespacedName{
							Name:      fmt.Sprintf("telegraf-config-%s", pod.GetName()),
							Namespace: pod.GetNamespace(),
						}
						return k8sClient.Get(testCtx, key, secret)
					}, timeout, interval).Should(Succeed())

					fixture, err := os.ReadFile("../../config/testdata/fixtures/global-tags.toml")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(string(secret.Data["telegraf.conf"])).Should(Equal(string(fixture)))

					cleanUpPod(pod.GetName())
					cleanUpSecret(secret.GetName())
				})

				It("Should fail reconciliation if raw input is invalid toml", func() {
					pod := newTestPod(
						"invalid-raw-input",
						map[string]string{metadata.SidecarInjectedLabel: "true"},
						map[string]string{metadata.TelegrafConfigRawInputAnnotation: "[[inputs.exec]]invalid1"},
					)
					Expect(k8sClient.Create(testCtx, pod)).Should(Succeed())
					Eventually(func() error {
						p := &corev1.Pod{}
						key := types.NamespacedName{Name: pod.GetName(), Namespace: pod.GetNamespace()}
						return k8sClient.Get(testCtx, key, p)
					}, timeout, interval).Should(Succeed())

					secretKey := types.NamespacedName{
						Name:      fmt.Sprintf("telegraf-config-%s", pod.GetName()),
						Namespace: namespace,
					}

					By("Expecting the secret to not be created with NotFound error")
					Consistently(func() bool {
						secret := &corev1.Secret{}
						err := k8sClient.Get(testCtx, secretKey, secret)
						if err != nil {
							return apierrors.IsNotFound(err)
						}
						return false
					}, duration, interval).Should(BeTrue())

					cleanUpPod(pod.GetName())
				})
			})
		})
	})
})

func newTestPod(name string, labels map[string]string, annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
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

	// Ensure the delete has completed successfully
	Eventually(func() error {
		p := &corev1.Pod{}
		return k8sClient.Get(testCtx, podKey, p)
	}, timeout, interval).ShouldNot(Succeed())
}

func cleanUpSecret(name string) {
	secretKey := types.NamespacedName{Name: name, Namespace: namespace}
	Eventually(func() error {
		s := &corev1.Secret{}
		err := k8sClient.Get(testCtx, secretKey, s)
		if apierrors.IsNotFound(err) {
			return nil
		}
		return k8sClient.Delete(testCtx, s)
	}, timeout, interval).Should(Succeed())

	// Ensure the delete has completed successfully
	Eventually(func() error {
		s := &corev1.Secret{}
		return k8sClient.Get(testCtx, secretKey, s)
	}, timeout, interval).ShouldNot(Succeed())
}
