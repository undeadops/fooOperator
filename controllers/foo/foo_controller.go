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

package foo

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	"github.com/undeadops/fooOperator/api/v1alpha1"
	"github.com/undeadops/fooOperator/controllers"
)

const taskLabel string = "taskId"
const FooFinalizer string = "finalize.foo.undeadops.xyz"

// FooReconciler reconciles a Foo object
type FooReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=apps.undeadops.xyz,resources=foos,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.undeadops.xyz,resources=foos/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.undeadops.xyz,resources=foos/finalizers,verbs=update

func (r *FooReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var err error

	// Fetching Latest Foo Instance
	foo := &v1alpha1.Foo{}
	err = r.Get(ctx, req.NamespacedName, foo)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	fooAdapter := NewFooAdapter(foo, r.Log, r.Client, r.Scheme, r.Recorder)
	result, err := r.ReconcileFooHandler(fooAdapter)
	return result, err
}

type ReconcileOperation func() (controllers.OperationResult, error)

func (r *FooReconciler) ReconcileFooHandler(adapter *FooAdapter) (ctrl.Result, error) {
	operations := []ReconcileOperation{
		adapter.FinalizeFoo,
		adapter.CreatePod,
		adapter.CreateService,
		adapter.CreateIngress,
		adapter.UpdateStatus,
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

// Create Pod Object for Foo
func (r *FooReconciler) createPod(foo *v1alpha1.Foo) *corev1.Pod {
	// Changing to create deployment instead of pod, because, updates are easier
	labels := make(map[string]string)
	labels["app"] = foo.Name

	// Create Ports Definition
	var ports []corev1.ContainerPort
	for _, p := range foo.Spec.Pod.Ports {
		ports = append(ports, corev1.ContainerPort{
			Name:          p.Name,
			HostPort:      p.HostPort,
			ContainerPort: p.ContainerPort,
			Protocol:      corev1.Protocol(p.Protocol),
		})
	}

	// Create Env Definition
	var envs []corev1.EnvVar
	for _, e := range foo.Spec.Pod.Env {
		envs = append(envs, corev1.EnvVar{
			Name:  e.Name,
			Value: e.Value,
		})
	}

	// Create Resources Def
	resources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceName("cpu"):    resource.MustParse(foo.Spec.Pod.Resources.Limits.Cpu),
			corev1.ResourceName("memory"): resource.MustParse(foo.Spec.Pod.Resources.Limits.Memory),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceName("cpu"):    resource.MustParse(foo.Spec.Pod.Resources.Requests.Cpu),
			corev1.ResourceName("memory"): resource.MustParse(foo.Spec.Pod.Resources.Requests.Memory),
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      foo.Name,
			Namespace: foo.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Image:     foo.Spec.Pod.Image,
				Name:      "foo",
				Command:   foo.Spec.Pod.Command,
				Ports:     ports,
				Env:       envs,
				Resources: resources,
			}},
		},
	}
	return pod
}

// Create Service Object for Foo
func (r *FooReconciler) createService(foo *v1alpha1.Foo) *corev1.Service {
	labels := make(map[string]string)
	labels["app"] = foo.Name

	// Create Ports Definition
	var ports []corev1.ServicePort
	for _, p := range foo.Spec.Pod.Ports {
		ports = append(ports, corev1.ServicePort{
			Name:       p.Name,
			Protocol:   corev1.Protocol(p.Protocol),
			Port:       int32(p.ContainerPort),
			TargetPort: intstr.FromInt(int(p.ContainerPort)),
		})
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      foo.Name,
			Namespace: foo.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": foo.Name,
			},
			Ports: ports,
			Type:  corev1.ServiceTypeClusterIP,
		},
	}
	return service
}

// SetupWithManager sets up the controller with the Manager.
func (r *FooReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Foo{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Owns(&netv1.Ingress{}).
		Complete(r)
}
