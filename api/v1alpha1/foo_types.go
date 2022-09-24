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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FooSpec defines the desired state of Foo
type FooSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Pod     FooPodSpec     `json:"pod"`
	Ingress FooIngressSpec `json:"ingress,omitempty"`
}

// FooPodSpec defines the desired config of our pod
type FooPodSpec struct {
	Annotations        map[string]string `json:"annotations,omitempty"`
	Labels             map[string]string `json:"labels,omitempty"`
	ServiceAccountName string            `json:"serviceAccountName,omitempty"`
	Image              string            `json:"image"`
	Command            []string          `json:"command,omitempty"`
	Env                []PodEnv          `json:"env,omitempty"`
	Ports              []PodPorts        `json:"ports,omitempty"`
	Resources          PodResources      `json:"resources,omitempty"`
	NodeSelector       map[string]string `json:"nodeselector,omitempty"`
}

// FooIngressSpec defines the desired config of our Ingress
type FooIngressSpec struct {
	Annotations      map[string]string `json:"annotations,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	ServicePort      int32             `json:"servicePort"`
	IngressClassName string            `json:"ingressClassName,omitempty"`
}

// PodEnv defines environment variables passed to running Container
type PodEnv struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// PodPorts defines the Ports exposed
type PodPorts struct {
	Name          string `json:"name,omitempty"`
	HostPort      int32  `json:"hostPort,omitempty"`
	ContainerPort int32  `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
}

// PodResources defines the cpu/memory request limits
type PodResources struct {
	Requests PodResourceList `json:"requests,omitempty"`
	Limits   PodResourceList `json:"limits,omitempty"`
}

// PodResourceList
type PodResourceList struct {
	Cpu    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// FooStatus defines the observed state of Foo
type FooStatus struct {
	State           string          `json:"state,omitempty"`
	TaskId          string          `json:"taskId,omitempty"`
	Pod             string          `json:"pod,omitempty"`
	PodStatus       corev1.PodPhase `json:"podStatus,omitempty"`
	LastStateChange metav1.Time     `json:"lastStateChange"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Foo is the Schema for the foos API
type Foo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FooSpec   `json:"spec,omitempty"`
	Status FooStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FooList contains a list of Foo
type FooList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Foo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Foo{}, &FooList{})
}
