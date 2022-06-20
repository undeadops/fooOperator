/*
Copyright 2022.

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

package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	log "sigs.k8s.io/controller-runtime/pkg/log"

	appsv1alpha1 "github.com/undeadops/fooOperator/api/v1alpha1"
)

// FooBarReconciler reconciles a FooBar object
type FooBarReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps.undeadops.xyz,resources=foobars,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.undeadops.xyz,resources=foobars/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.undeadops.xyz,resources=foobars/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FooBar object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *FooBarReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctrllog := log.FromContext(ctx)

	fooBar := &appsv1alpha1.FooBar{}
	pod := &corev1.Pod{}
	svc := &corev1.Service{}

	ctrllog.Info("⚡️ Event received! ⚡️")
	ctrllog.Info("Request: ", "req", req)

	// Check if resource deleted
	err := r.Get(ctx, req.NamespacedName, fooBar)
	if err != nil {
		if errors.IsNotFound(err) {
			ctrllog.Info("FooBar resource not found, check if Foos need to be Deleted.")

			err = r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, pod)
			if err != nil {
				if errors.IsNotFound(err) {
					ctrllog.Info("Nothing to do, no FooBars found")
					return ctrl.Result{}, nil
				} else {
					ctrllog.Error(err, "❌ Failed to get Pod")
					return ctrl.Result{}, err
				}
			}

			err = r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, svc)
			if err != nil {
				if errors.IsNotFound(err) {
					ctrllog.Info("Nothing to do, no servie found.")
					return ctrl.Result{}, nil
				} else {
					ctrllog.Error(err, "❌ Failed to get service")
					return ctrl.Result{}, err
				}
			} else {
				ctrllog.Info("☠️ Service exists: delete it. ☠️")
				r.Delete(ctx, svc)
			}
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FooBarReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.FooBar{}).
		Complete(r)
}
