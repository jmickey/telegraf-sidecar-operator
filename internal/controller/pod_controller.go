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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/jmickey/telegraf-sidecar-operator/internal/classdata"
	"github.com/jmickey/telegraf-sidecar-operator/internal/metadata"
	k8s "github.com/jmickey/telegraf-sidecar-operator/internal/metadata"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme               *runtime.Scheme
	Recorder             record.EventRecorder
	ClassDataHandler     classdata.Handler
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
		Owns(&corev1.Secret{}).
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
	} else {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "failed to lookup secret from kubernetes api")
			return ctrl.Result{}, err
		}
	}

	return r.reconcile(ctx, obj)
}

func (r *PodReconciler) reconcile(ctx context.Context, obj *corev1.Pod) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	telegrafConfig := newTelegrafConfig(r.ClassDataHandler, r.DefaultClass, r.EnableInternalPlugin)
	if err := telegrafConfig.applyAnnotationOverrides(obj.GetAnnotations()); err != nil {
		msg := fmt.Sprintf("one or more warnings were generated when applying telegraf pod annotations: [ %s ]", err.Error())
		r.Recorder.Event(obj, corev1.EventTypeWarning, "InvalidAnnotationFormat", msg)
		log.Info(msg)
	}

	configData, err := telegrafConfig.buildConfigData()
	if err != nil {
		msg := fmt.Sprintf("error building telegraf config: %s", err.Error())
		r.Recorder.Event(obj, corev1.EventTypeWarning, "InvalidTelegrafConfiguration", msg)
		log.Info(msg)

		return ctrl.Result{}, fmt.Errorf("error building telegraf configuration: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("telegraf-config-%s", obj.GetName()),
			Namespace: obj.GetNamespace(),
			Labels: map[string]string{
				metadata.TelegrafSecretClassNameLabel: telegrafConfig.class,
				metadata.TelegrafSecretPodLabel:       obj.GetName(),
				metadata.SecretManagedByLabelKey:      metadata.SecretManagedByLabelValue,
			},
		},
		Type: "Opaque",
		StringData: map[string]string{
			"telegraf.conf": configData,
		},
	}

	if err := controllerutil.SetOwnerReference(obj, secret, r.Scheme); err != nil {
		r.Recorder.Eventf(obj, corev1.EventTypeWarning, "SetOwnerReferenceError",
			"failed to set owner reference for secret: %s: %s", secret.GetName(), err.Error())
		log.Error(err, "failed to set owner reference for secret")
		return ctrl.Result{}, fmt.Errorf("failed to set owner reference for secret: %w", err)
	}

	if err := r.Client.Create(ctx, secret); err != nil {
		r.Recorder.Eventf(obj, corev1.EventTypeWarning, "CreateSecretInClusterError",
			"failed to create secret: %s in cluster: %s", secret.GetName(), err.Error())
		log.Error(err, "failed to create secret in cluster", "secret", secret.GetName())
		return ctrl.Result{}, fmt.Errorf("failed to create secret: %s in cluster: %w", secret.GetName(), err)
	}

	msg := fmt.Sprintf("successfully create telegraf config secret: %s", secret.GetName())
	r.Recorder.Event(obj, corev1.EventTypeNormal, "TelegrafConfigCreateSuccessful", msg)

	return ctrl.Result{}, nil
}

func (r *PodReconciler) shouldAttemptReconcilation(pod *corev1.Pod) bool {
	for key := range pod.GetLabels() {
		if key == k8s.SidecarInjectedLabel {
			return true
		}
	}

	return false
}
