package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppSpec   `json:"spec"`
	Status AppStatus `json:"status"`
}

type DeploymentSpec struct {
	Name     string `json:"name"`
	Image    string `json:"image"`
	Replicas int32  `json:"replicas"`
}

type ServiceSpec struct {
	Name string `json:"name"`
}

type IngressSpec struct {
	Name string `json:"name"`
}

type AppSpec struct {
	Deployment DeploymentSpec `json:"deployment"`
	Service    ServiceSpec    `json:"service"`
	Ingress    IngressSpec    `json:"ingress"`
}

type AppStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []App `json:"items"`
}
