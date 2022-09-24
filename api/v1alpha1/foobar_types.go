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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FooBarSpec defines the desired state of FooBar
type FooBarSpec struct {
	Replicas int32   `json:"replicas,omitempty"`
	Foo      FooSpec `json:"foo"`
}

// FooBarStatus defines the observed state of FooBar
type FooBarStatus struct {
	Active      int32  `json:"active"`
	Idle        int32  `json:"idle"`
	CurrentHash string `json:"current_hash"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
//+kubebuilder:printcolumn:name="Idle Foos",type="integer",JSONPath=".status.idle",description="Idle Foos"
//+kubebuilder:printcolumn:name="Active Foos",type="integer",JSONPath=".status.active",description="Active Foos"
//+kubebuilder:printcolumn:name="Current Hash",type="string",JSONPath=".status.current_hash",description="Current Deployment Hash"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// FooBar is the Schema for the foobars API
type FooBar struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FooBarSpec   `json:"spec,omitempty"`
	Status FooBarStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FooBarList contains a list of FooBar
type FooBarList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FooBar `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FooBar{}, &FooBarList{})
}
