package injectorwebhook

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/jmickey/telegraf-sidecar-operator/internal/k8s"
)

var (
	testCtx = context.Background()
)

var _ = Describe("Sidecar injector webhook", func() {
	AfterEach(func() {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
		}
		err := k8sClient.Delete(testCtx, pod)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
	})

	When("Creating a pod under the defaulting webhook", func() {
		Context("And there is no telegraf annoation", func() {
			var pod *corev1.Pod
			BeforeEach(func() {
				pod = &corev1.Pod{
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
			})

			It("Should allow the pod admission and not inject the telegraf container", func() {
				err := k8sClient.Create(testCtx, pod)
				Expect(err).NotTo(HaveOccurred())

				pod = &corev1.Pod{}
				name := types.NamespacedName{
					Namespace: "default",
					Name:      "test-pod",
				}
				err = k8sClient.Get(testCtx, name, pod)
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
				for _, c := range pod.Spec.Containers {
					Expect(c.Name).NotTo(Equal("telegraf"))
				}
			})
		})
	})

	Context("And there is a telegraf annotation", func() {
		var pod *corev1.Pod
		BeforeEach(func() {
			pod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
					Annotations: map[string]string{
						k8s.TelegrafConfigIntervalAnnotation: "10s",
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
		})

		It("Should inject the telegraf container and config volume with default settings", func() {
			err := k8sClient.Create(testCtx, pod)
			Expect(err).NotTo(HaveOccurred())

			pod = &corev1.Pod{}
			name := types.NamespacedName{
				Name:      "test-pod",
				Namespace: "default",
			}
			err = k8sClient.Get(testCtx, name, pod)
			Expect(err).NotTo(HaveOccurred())
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
		})

		It("Should proceed with injection using defaults if resource annotations are invalid", func() {
			invalidValue := "1000x"
			pod.Annotations[k8s.SidecarRequestsCPUAnnotation] = invalidValue
			pod.Annotations[k8s.SidecarLimitsCPUAnnotation] = invalidValue
			pod.Annotations[k8s.SidecarRequestsMemoryAnnotation] = invalidValue
			pod.Annotations[k8s.SidecarLimitsMemoryAnnotation] = invalidValue

			err := k8sClient.Create(testCtx, pod)
			Expect(err).NotTo(HaveOccurred())

			pod = &corev1.Pod{}
			name := types.NamespacedName{
				Name:      "test-pod",
				Namespace: "default",
			}
			err = k8sClient.Get(testCtx, name, pod)
			Expect(err).NotTo(HaveOccurred())
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
		})

		It("Should proceed override container resources with annotation values", func() {
			var (
				overrideRequestsCPU    = "500m"
				overrideRequestsMemory = "500Mi"
				overrideLimitsCPU      = "800m"
				overrideLimitsMemory   = "800Mi"
			)
			pod.Annotations[k8s.SidecarRequestsCPUAnnotation] = overrideRequestsCPU
			pod.Annotations[k8s.SidecarRequestsMemoryAnnotation] = overrideRequestsMemory
			pod.Annotations[k8s.SidecarLimitsCPUAnnotation] = overrideLimitsCPU
			pod.Annotations[k8s.SidecarLimitsMemoryAnnotation] = overrideLimitsMemory

			err := k8sClient.Create(testCtx, pod)
			Expect(err).NotTo(HaveOccurred())

			pod = &corev1.Pod{}
			name := types.NamespacedName{
				Name:      "test-pod",
				Namespace: "default",
			}
			err = k8sClient.Get(testCtx, name, pod)
			Expect(err).NotTo(HaveOccurred())
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
		})
	})
})
