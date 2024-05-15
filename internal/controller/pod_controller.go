/*
Copyright 2024.

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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/jmickey/telegraf-sidecar-operator/internal/k8s"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme               *runtime.Scheme
	Recorder             record.EventRecorder
	DefaultClass         string
	EnableInternalPlugin bool
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	obj := &corev1.Pod{}
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		log.Error(err, "failed to fetch pod")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !r.shouldAttemptReconcilation(obj) {
		log.Info("reconciliation skipped, pod doesn't have telegraf container")
		return ctrl.Result{}, nil
	}

	secret := &corev1.Secret{}
	name := types.NamespacedName{
		Name:      fmt.Sprintf("telegraf-config-%s", obj.GetName()),
		Namespace: req.Namespace,
	}
	if err := r.Get(ctx, name, secret); err == nil {
		log.Info("reconciliation skipped, telegraf-config secret for pod already exists")
		return ctrl.Result{}, nil
	}

	return r.reconcile(ctx, obj)
}
func (r *PodReconciler) reconcile(ctx context.Context, obj *corev1.Pod) (ctrl.Result, error) {
	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

func (r *PodReconciler) shouldAttemptReconcilation(pod *corev1.Pod) bool {
	for key := range pod.GetLabels() {
		if key == k8s.ContainerInjectedLabel {
			return true
		}
	}

	return false
}
