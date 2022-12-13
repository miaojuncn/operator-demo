package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AppSpec struct {
	Image         string `json:"image"`
	Replicas      int32  `json:"replicas"`
	EnableService bool   `json:"enable_service"`
	// +kubebuilder:default:enable_ingress=false
	EnableIngress bool `json:"enable_ingress"`
}

type AppStatus struct {
	DeploymentName string `json:"deployment_name"`
	// +kubebuilder:default:service_name=""
	ServiceName string `json:"service_name"`
	// +kubebuilder:default:ingress_name=""
	IngressName string `json:"ingress_name"`
}

// +kubebuilder:printcolumn:name:"deployment",type:string,JSONPath=`.status.deployment_name`
// +kubebuilder:printcolumn:name:"service",type:string,JSONPath=`.status.service_name`
// +kubebuilder:printcolumn:name:"ingress",type:string,JSONPath=`.status.ingress_name`
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// App is the Schema for the apps API
type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppSpec   `json:"spec,omitempty"`
	Status AppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AppList contains a list of App
type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []App `json:"items"`
}

func init() {
	SchemeBuilder.Register(&App{}, &AppList{})
}
