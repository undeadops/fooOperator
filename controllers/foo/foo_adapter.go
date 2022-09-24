package foo

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"github.com/undeadops/fooOperator/api/v1alpha1"
	appsv1alpha1 "github.com/undeadops/fooOperator/api/v1alpha1"
	"github.com/undeadops/fooOperator/controllers"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type FooAdapter struct {
	foo      *appsv1alpha1.Foo
	logger   logr.Logger
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

func NewFooAdapter(foo *appsv1alpha1.Foo, logger logr.Logger, client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder) *FooAdapter {
	return &FooAdapter{
		foo:      foo,
		logger:   logger,
		client:   client,
		scheme:   scheme,
		recorder: recorder,
	}
}

func (f *FooAdapter) FinalizeFoo() (controllers.OperationResult, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(2))
	defer cancel()

	var err error

	var updated bool = false

	if f.foo.Status.State == "Busy" {
		if !controllerutil.ContainsFinalizer(f.foo, FooFinalizer) {
			controllerutil.AddFinalizer(f.foo, FooFinalizer)
		}
	}

	if f.foo.Status.State != "Busy" {
		if controllerutil.ContainsFinalizer(f.foo, FooFinalizer) {
			controllerutil.RemoveFinalizer(f.foo, FooFinalizer)
		}
	}

	if updated {
		err = f.client.Update(ctx, f.foo)
		if err != nil {
			return controllers.RequeueWithError(fmt.Errorf("failed to update Foo resource with finalizer: %v", err))
		}
		return controllers.Requeue()
	}
	return controllers.ContinueProcessing()
}

// Update Status for Foo
func (f *FooAdapter) UpdateStatus() (controllers.OperationResult, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(2))
	defer cancel()

	var err error
	pod := &corev1.Pod{}

	err = f.client.Get(ctx, types.NamespacedName{Namespace: f.foo.GetNamespace(), Name: f.foo.Name}, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			return controllers.ContinueProcessing()
		}
		return controllers.RequeueWithError(fmt.Errorf("failed to retrieve pod info for StatusUpdate: %v", err))
	}

	var fooStatus string = "Pending"
	labels := f.foo.GetLabels()
	if pod.Status.Phase == "Running" {
		if labels[taskLabel] != "" {
			fooStatus = "Busy"
		} else {
			fooStatus = "Idle"
		}
	}

	// Resource Creation Complete
	status := v1alpha1.FooStatus{
		State:     fooStatus,
		TaskId:    labels[taskLabel],
		Pod:       pod.Status.PodIP,
		PodStatus: pod.Status.Phase,
	}

	if !reflect.DeepEqual(f.foo.Status, status) {
		f.foo.Status = status
		f.foo.Status.LastStateChange = metav1.Now()
	}

	err = f.client.Status().Update(ctx, f.foo)
	if err != nil {
		// Failure here isn't critical, requeue the update
		return controllers.Requeue()
	}

	return controllers.ContinueProcessing()
}

// Create Pod Object for Foo
func (f *FooAdapter) CreatePod() (controllers.OperationResult, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(2))
	defer cancel()

	newPod := f.newPod()
	_ = ctrl.SetControllerReference(f.foo, newPod, f.scheme)

	err := f.client.Create(ctx, newPod)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return controllers.ContinueProcessing()
		}
		return controllers.RequeueWithError(fmt.Errorf("failed to create Foo Pod - Requeueing"))
	}
	return controllers.ContinueProcessing()
}

func (f *FooAdapter) newPod() *corev1.Pod {
	// Changing to create deployment instead of pod, because, updates are easier
	labels := make(map[string]string)
	labels["app"] = f.foo.Name

	// Create Ports Definition
	var ports []corev1.ContainerPort
	for _, p := range f.foo.Spec.Pod.Ports {
		ports = append(ports, corev1.ContainerPort{
			Name:          p.Name,
			HostPort:      p.HostPort,
			ContainerPort: p.ContainerPort,
			Protocol:      corev1.Protocol(p.Protocol),
		})
	}

	// Create Env Definition
	var envs []corev1.EnvVar
	for _, e := range f.foo.Spec.Pod.Env {
		envs = append(envs, corev1.EnvVar{
			Name:  e.Name,
			Value: e.Value,
		})
	}

	// Create Resources Def
	resources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceName("cpu"):    resource.MustParse(f.foo.Spec.Pod.Resources.Limits.Cpu),
			corev1.ResourceName("memory"): resource.MustParse(f.foo.Spec.Pod.Resources.Limits.Memory),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceName("cpu"):    resource.MustParse(f.foo.Spec.Pod.Resources.Requests.Cpu),
			corev1.ResourceName("memory"): resource.MustParse(f.foo.Spec.Pod.Resources.Requests.Memory),
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.foo.Name,
			Namespace: f.foo.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Image:     f.foo.Spec.Pod.Image,
				Name:      "foo",
				Command:   f.foo.Spec.Pod.Command,
				Ports:     ports,
				Env:       envs,
				Resources: resources,
			}},
		},
	}
	return pod
}

// Create Service Object for Foo
func (f *FooAdapter) CreateService() (controllers.OperationResult, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(2))
	defer cancel()

	newSvc := f.newService()
	_ = ctrl.SetControllerReference(f.foo, newSvc, f.scheme)

	err := f.client.Create(ctx, newSvc)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return controllers.ContinueProcessing()
		}
		return controllers.RequeueWithError(fmt.Errorf("failed to create Service - Requeueing"))
	}
	return controllers.ContinueProcessing()
}

func (f *FooAdapter) newService() *corev1.Service {
	labels := make(map[string]string)
	labels["app"] = f.foo.Name

	// Create Ports Definition
	var ports []corev1.ServicePort
	for _, p := range f.foo.Spec.Pod.Ports {
		ports = append(ports, corev1.ServicePort{
			Name:       p.Name,
			Protocol:   corev1.Protocol(p.Protocol),
			Port:       int32(p.ContainerPort),
			TargetPort: intstr.FromInt(int(p.ContainerPort)),
		})
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.foo.Name,
			Namespace: f.foo.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": f.foo.Name,
			},
			Ports: ports,
			Type:  corev1.ServiceTypeClusterIP,
		},
	}
	return service
}

// Create Ingress Object for Foo
func (f *FooAdapter) CreateIngress() (controllers.OperationResult, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(2))
	defer cancel()

	newIng := f.newIngress()
	_ = ctrl.SetControllerReference(f.foo, newIng, f.scheme)

	err := f.client.Create(ctx, newIng)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return controllers.ContinueProcessing()
		}
		return controllers.RequeueWithError(fmt.Errorf("failed to create Ingress for Foo - Requeuing"))
	}

	return controllers.ContinueProcessing()
}

var pathPrefix netv1.PathType = netv1.PathTypePrefix

func (f *FooAdapter) newIngress() *netv1.Ingress {
	labels := make(map[string]string)
	labels["app"] = f.foo.Name

	ingress := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        f.foo.Name,
			Namespace:   f.foo.Namespace,
			Labels:      labels,
			Annotations: f.foo.Spec.Ingress.Annotations,
		},
		Spec: netv1.IngressSpec{
			IngressClassName: &f.foo.Spec.Ingress.IngressClassName,
			Rules: []netv1.IngressRule{
				{
					IngressRuleValue: netv1.IngressRuleValue{
						HTTP: &netv1.HTTPIngressRuleValue{
							Paths: []netv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathPrefix,
									Backend: netv1.IngressBackend{
										Service: &netv1.IngressServiceBackend{
											Name: f.foo.Name,
											Port: netv1.ServiceBackendPort{
												Number: f.foo.Spec.Ingress.ServicePort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return ingress
}
