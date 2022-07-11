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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	appsv1alpha1 "github.com/undeadops/fooOperator/api/v1alpha1"
)

// FooReconciler reconciles a Foo object
type FooReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps.undeadops.xyz,resources=foos,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.undeadops.xyz,resources=foos/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.undeadops.xyz,resources=foos/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Foo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *FooReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	foo := &appsv1alpha1.Foo{}

	existingDeployment := &appsv1.Deployment{}
	existingService := &corev1.Service{}
	existingIngress := &netv1.Ingress{}

	log.Info("⚡️ Event received! ⚡️")
	log.Info("Request: ", "req", req)

	// CR deleted : check if  the Deployment and the Service must be deleted
	err := r.Get(ctx, req.NamespacedName, foo)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Foo resource not found, check if a Pod must be deleted.")

			// delete Pod
			err = r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, existingDeployment)
			if err != nil {
				if errors.IsNotFound(err) {
					log.Info("Nothing to do, no Foo Deployment found.")
					return ctrl.Result{}, nil
				} else {
					log.Error(err, "❌ Failed to get Foo Deployment")
					return ctrl.Result{}, err
				}
			} else {
				log.Info("☠️ Foo Deployment exists: delete it. ☠️")
				r.Delete(ctx, existingDeployment)
			}
			// delete service
			err = r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, existingService)
			if err != nil {
				if errors.IsNotFound(err) {
					log.Info("Nothing to do, no Foo Service found.")
					return ctrl.Result{}, nil
				} else {
					log.Error(err, "❌ Failed to get Foo Service")
					return ctrl.Result{}, err
				}
			} else {
				log.Info("☠️ Foo Service exists: delete it. ☠️")
				r.Delete(ctx, existingService)
			}

			// delete ingress
			err = r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, existingIngress)
			if err != nil {
				if errors.IsNotFound(err) {
					log.Info("Nothing to do, no Foo Ingress found.")
					return ctrl.Result{}, nil
				} else {
					log.Error(err, " Failed to get Foo Ingress")
					return ctrl.Result{}, err
				}
			} else {
				log.Info("Foo Ingress exists: delete it.")
				r.Delete(ctx, existingIngress)
				return ctrl.Result{}, nil
			}
		}
	} else {
		log.Info("ℹ️  CR state ℹ️", "Foo.Name", foo.Name, " Foo.Namespace", foo.Namespace)

		// Check if the foo already exists, if not: create a new one.
		err = r.Get(ctx, types.NamespacedName{Name: foo.Name, Namespace: foo.Namespace}, existingDeployment)
		if err != nil && errors.IsNotFound(err) {
			// Define a new deployment
			newFoo := r.createPod(foo)
			log.Info("✨ Creating a new Foo", "Foo.Namespace", newFoo.Namespace, "Foo.Name", newFoo.Name)

			err = r.Create(ctx, newFoo)
			if err != nil {
				log.Error(err, "❌ Failed to create new Foo", "Foo.Namespace", newFoo.Namespace, "Foo.Name", newFoo.Name)
				return ctrl.Result{}, err
			}
		} else if err == nil {
			log.Info("🔁 Foo exits, update the Foo! 🔁")
			// Pod exists!
			// Updates are Replacements!
			existingDeployment.Spec.Template.Spec.Containers[0].Image = foo.Spec.Pod.Image
			existingDeployment.Spec.Template.Spec.Containers[0].Env = foo.Spec.Pod.Env
			existingDeployment.Spec.Template.Spec.Conteiners[0].Resources = foo.Spec.Pod.Resources
			existingDeployment.Spec.Template.Spec.Containers[0].
				err = r.Update(ctx, existingDeployment)
			if err != nil {
				log.Error(err, "Failed to create replacement Foo", "Foo.Namespace", foo.Namespace, "Foo.Name", foo.Name)
				return ctrl.Result{}, err
			}
			log.Info("Replacement Foo Has been Started!", "Foo.Namespace", foo.Namespace, "Foo.Name", foo.Name)

		} else if err != nil {
			log.Error(err, "Failed to get Deployment")
			return ctrl.Result{}, err
		}

		// Check if the service already exists, if not: create a new one
		err = r.Get(ctx, types.NamespacedName{Name: foo.Name, Namespace: foo.Namespace}, existingService)
		if err != nil && errors.IsNotFound(err) {
			// Create the Service
			newService := r.createService(foo)
			log.Info("✨ Creating a new Service", "Service.Namespace", newService.Namespace, "Service.Name", newService.Name)
			err = r.Create(ctx, newService)
			if err != nil {
				log.Error(err, "❌ Failed to create new Service", "Service.Namespace", newService.Namespace, "Service.Name", newService.Name)
				return ctrl.Result{}, err
			}
		} else if err == nil {
			// Service exists, check if the port has to be updated.
			var port = int32(foo.Spec.Ingress.ServicePort)
			if existingService.Spec.Ports[0].Port != port {
				log.Info("🔁 Port number changes, update the service! 🔁")
				existingService.Spec.Ports[0].Port = port
				err = r.Update(ctx, existingService)
				if err != nil {
					log.Error(err, "❌ Failed to update Service", "Service.Namespace", existingService.Namespace, "Service.Name", existingService.Name)
					return ctrl.Result{}, err
				}
			}
		} else if err != nil {
			log.Error(err, "Failed to get Service")
			return ctrl.Result{}, err
		}

		// Check if ingress already exists, if not: create a new one
		err = r.Get(ctx, types.NamespacedName{Name: foo.Name, Namespace: foo.Namespace}, existingIngress)
		if err != nil && errors.IsNotFound(err) {
			// Create the Ingress
			newIngress := r.createIngress(foo)
			log.Info(" Creating a new Ingress", "Ingress.Namespace", newIngress.Namespace, "Ingress.Name", newIngress.Name)
			err = r.Create(ctx, newIngress)
			if err != nil {
				log.Error(err, "Failed to create new Service", "Ingress.Namespace", newIngress.Namespace, "Ingress.Name", newIngress.Name)
				return ctrl.Result{}, err
			}
		} else if err == nil {
			// Ingress exists, Update accordingly
			recIngress := existingIngress.DeepCopy()
			recIngress.Labels = foo.Labels
			recIngress.Annotations = foo.Spec.Ingress.Annotations
			//recIngress.Spec = getIngressSpec(foo)
			// Update Ingress...
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FooReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.Foo{}).
		Complete(r)
}

// Create Pod Object for Foo
func (r *FooReconciler) createPod(foo *appsv1alpha1.Foo) *appsv1.Deployment {
	// Changing to create deployment instead of pod, because, updates are easier
	labels := make(map[string]string)
	labels["app"] = foo.Name
	var replicas int32
	replicas = 1

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

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      foo.Name,
			Namespace: foo.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
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
			},
		},
	}
	return deployment
}

// Create Service Object for Foo
func (r *FooReconciler) createService(foo *appsv1alpha1.Foo) *corev1.Service {
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

var pathPrefix netv1.PathType = netv1.PathTypePrefix

// Create Ingress Object for Foo
func (r *FooReconciler) createIngress(foo *appsv1alpha1.Foo) *netv1.Ingress {
	labels := make(map[string]string)
	labels["app"] = foo.Name

	ingress := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        foo.Name,
			Namespace:   foo.Namespace,
			Labels:      labels,
			Annotations: foo.Spec.Ingress.Annotations,
		},
		Spec: netv1.IngressSpec{
			IngressClassName: &foo.Spec.Ingress.IngressClassName,
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
											Name: foo.Name,
											Port: netv1.ServiceBackendPort{
												Number: foo.Spec.Ingress.ServicePort,
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
