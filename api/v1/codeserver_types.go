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

package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CodeServerSpec defines the desired state of CodeServer
type CodeServerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// image
	Image string `json:"image,omitempty"`
	//resource name
	Name string `json:"name,omitempty"`

	RecycleAfterSeconds  *int64 `json:"recycleAfterSeconds,omitempty"`
	InactiveAfterSeconds *int64 `json:"inactiveAfterSeconds,omitempty"`
	//add
	Add  bool        `json:"add,omitempty" description:"add"`
	Envs []v1.EnvVar `json:"envs,omitempty"`
}

// CodeServerStatus defines the observed state of CodeServer
type CodeServerStatus struct {
	//Server conditions
	Conditions []ServerCondition `json:"conditions,omitempty" protobuf:"bytes,1,opt,name=conditions"`
}

// ServerConditionType describes the type of state of code server condition
type ServerConditionType string

const (
	// ServerCreated means the code server has been accepted by the system.
	ServerCreated ServerConditionType = "ServerCreated"
	// ServerReady means the code server has been ready for usage.
	ServerReady ServerConditionType = "ServerReady"
	// ServerBound means the code server has been bound to user.
	ServerBound ServerConditionType = "ServerBound"
	// ServerRecycled means the code server has been recycled totally.
	ServerRecycled ServerConditionType = "ServerRecycled"
	// ServerInactive means the code server will be marked inactive if `InactiveAfterSeconds` elapsed
	ServerInactive ServerConditionType = "ServerInactive"
	// ServerErrored means failed to reconcile code server.
	ServerErrored ServerConditionType = "ServerErrored"
)

// ServerCondition describes the state of the code server at a certain point.
type ServerCondition struct {
	// Type of code server condition.
	Type ServerConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human-readable message indicating details about the transition.
	Message map[string]string `json:"message,omitempty"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CodeServer is the Schema for the codeservers API
type CodeServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CodeServerSpec   `json:"spec,omitempty"`
	Status CodeServerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CodeServerList contains a list of CodeServer
type CodeServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CodeServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CodeServer{}, &CodeServerList{})
}
