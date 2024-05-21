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
	"time"

	"github.com/jmickey/telegraf-sidecar-operator/internal/metadata"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	testCtx = context.Background()
)

var _ = Describe("Pod Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	When("A pod is created", func() {
		AfterEach(func() {
			podKey := types.NamespacedName{Name: "test-pod", Namespace: "default"}
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

			secretKey := types.NamespacedName{Name: "telegraf-config-test-pod", Namespace: "default"}
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
		})

		Context("And there is no telegraf.influxdata.com/injected label", func() {
			It("should not reconcile the object", func() {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
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
				Expect(k8sClient.Create(testCtx, pod)).Should(Succeed())

				secretKey := types.NamespacedName{
					Name:      fmt.Sprintf("telegraf-config-%s", pod.GetName()),
					Namespace: "default",
				}
				// The secret should not exist, so the success condition is a NotFound error
				Consistently(func() (int, error) {
					secret := &corev1.Secret{}
					err := k8sClient.Get(testCtx, secretKey, secret)
					if err != nil {
						if apierrors.IsNotFound(err) {
							return 0, nil
						}
						return -1, err
					}
					return 1, nil
				}, timeout, interval).Should(Equal(0))
			})
		})

		Context("And the telegraf config secret already exists", func() {
			It("Should skip further reconciliation", func() {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "telegraf-config-test-pod",
						Namespace: "default",
					},
				}
				Eventually(func() error {
					err := k8sClient.Create(testCtx, secret)
					Expect(err).Should(BeNil())
					s := &corev1.Secret{}
					key := types.NamespacedName{Name: secret.GetName(), Namespace: secret.GetNamespace()}
					return k8sClient.Get(testCtx, key, s)
				}, timeout, interval).Should(Succeed())

				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						Labels: map[string]string{
							metadata.SidecarInjectedLabel: "true",
						},
						Annotations: map[string]string{
							metadata.TelegrafConfigMetricsPortsAnnotation: "8080",
						},
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
				Expect(k8sClient.Create(testCtx, pod)).Should(Succeed())

				secretKey := types.NamespacedName{
					Name:      "telegraf-config-test-pod",
					Namespace: "default",
				}
				Eventually(func() bool {
					secret := &corev1.Secret{}
					Expect(k8sClient.Get(testCtx, secretKey, secret)).Should(Succeed())
					_, ok := secret.GetLabels()[metadata.SecretManagedByLabelKey]
					return ok
				}, timeout, interval).Should(BeFalse())
			})
		})

		Context("And the telegraf secret does not already exist", func() {
			It("Should complete the reconciliation successfully", func() {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						Labels: map[string]string{
							metadata.SidecarInjectedLabel: "true",
						},
						Annotations: map[string]string{
							metadata.TelegrafConfigMetricsPortsAnnotation: "8080",
						},
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
				Expect(k8sClient.Create(testCtx, pod)).Should(Succeed())
				Eventually(func() error {
					p := &corev1.Pod{}
					key := types.NamespacedName{Name: pod.GetName(), Namespace: pod.GetNamespace()}
					return k8sClient.Get(testCtx, key, p)
				}, timeout, interval).Should(Succeed())

				Eventually(func() error {
					s := &corev1.Secret{}
					key := types.NamespacedName{
						Name:      fmt.Sprintf("telegraf-config-%s", pod.GetName()),
						Namespace: pod.GetNamespace(),
					}
					return k8sClient.Get(testCtx, key, s)
				}, timeout, interval).Should(Succeed())
			})
		})
	})
})
