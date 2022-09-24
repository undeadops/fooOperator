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

package foobar

import (
	"context"

	"k8s.io/client-go/tools/record"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	appsv1alpha1 "github.com/undeadops/fooOperator/api/v1alpha1"
	"github.com/undeadops/fooOperator/controllers"
)

// FooBarReconciler reconciles a FooBar object
type FooBarReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=apps.undeadops.xyz,resources=foobars,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.undeadops.xyz,resources=foobars/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.undeadops.xyz,resources=foobars/finalizers,verbs=update

func (r *FooBarReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var err error

	fooBar := &appsv1alpha1.FooBar{}

	// Check if resource deleted
	err = r.Get(ctx, req.NamespacedName, fooBar)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	barAdapter := NewFooBarAdapter(fooBar, r.Log, r.Client, r.Scheme, r.Recorder)
	result, err := r.ReconcileHandler(barAdapter)

	return result, err
}

type ReconcileOperation func() (controllers.OperationResult, error)

func (r *FooBarReconciler) ReconcileHandler(adapter *FooBarAdapter) (ctrl.Result, error) {
	operations := []ReconcileOperation{
		adapter.UpdateStatus,
		adapter.ScaleUp,
		adapter.ScaleDown,
	}

	for _, operation := range operations {
		result, err := operation()
		if err != nil || result.RequeueRequest {
			return ctrl.Result{RequeueAfter: result.RequeueDelay}, err
		}
		if result.CancelRequest {
			return ctrl.Result{}, nil
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FooBarReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.FooBar{}).
		Owns(&appsv1alpha1.Foo{}).
		Complete(r)
}
