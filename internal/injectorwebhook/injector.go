package injectorwebhook

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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

	return nil
}
