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

	"github.com/jmickey/telegraf-sidecar-operator/internal/k8s"
)

type SidecarInjector struct {
	TelegrafImage  string
	RequestsCPU    string
	RequestsMemory string
	LimitsCPU      string
	LimitsMemory   string
}

//+kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=fail,groups=core,resources=pods,verbs=create;update,versions=v1,name=telegraf.mickey.dev,sideEffects=none,admissionReviewVersions=v1

func (s *SidecarInjector) SetupSidecarInjectorWebhookWithManager(mgr manager.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&corev1.Pod{}).
		WithDefaulter(s).
		Complete()
}

func (s *SidecarInjector) Default(ctx context.Context, obj runtime.Object) error {
	log := logf.FromContext(ctx,
		"logInstance", "injectorwebhook.injector", "func", "Default")

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected runtime.Object to be a Pod, got %T", obj)
	}

	if !s.shouldInjectContainer(pod) {
		log.V(2).Info("skipping pod, telegraf sidecar injector should not handle it",
			"generateName", pod.GetGenerateName())
		return nil
	}

	if pod.GetName() == "" {
		name := names.SimpleNameGenerator.GenerateName(pod.GetGenerateName())
		pod.SetName(name)
		log.V(2).Info("generated pod name", "pod", name)
	}

	log = log.WithValues("pod", pod.GetName())

	containerConfig, err := newContainerConfig(ctx, s, pod.GetName())
	if err != nil {
		log.Error(err, "failed to initialize container configuration")
		return err
	}
	containerConfig.applyAnnotationOverrides(pod.GetAnnotations())

	pod.Spec.Containers = append(pod.Spec.Containers, containerConfig.buildContainerSpec())
	pod.Spec.Volumes = append(pod.Spec.Volumes, s.newTelegrafConfigVolume(pod.GetName()))
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[k8s.ContainerInjectedLabel] = "true"

	log.Info("successfully injected telegraf sidecar container into pod")
	return nil
}

func (s *SidecarInjector) shouldInjectContainer(pod *corev1.Pod) bool {
	if s.hasTelegrafContainer(pod) {
		return false
	}

	for key := range pod.GetAnnotations() {
		if strings.Contains(key, k8s.Prefix) {
			return true
		}
	}

	return false
}

func (s *SidecarInjector) hasTelegrafContainer(pod *corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Name, "telegraf") {
			return true
		}
	}

	return false
}

func (s *SidecarInjector) newTelegrafConfigVolume(podName string) corev1.Volume {
	return corev1.Volume{
		Name: "telegraf-config",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: fmt.Sprintf("telegraf-config-%s", podName),
			},
		},
	}
}
