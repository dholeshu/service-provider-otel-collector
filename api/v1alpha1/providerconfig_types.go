/*
Copyright 2025.

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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProviderConfigSpec defines the desired state of ProviderConfig
type ProviderConfigSpec struct {
	// +optional
	// +kubebuilder:default:="1m"
	// +kubebuilder:validation:Format=duration
	PollInterval *metav1.Duration `json:"pollInterval,omitempty"`

	// Default OTEL collector container image
	// +optional
	// +kubebuilder:default:="otel/opentelemetry-collector-contrib"
	DefaultImage *string `json:"defaultImage,omitempty"`

	// Default OTEL collector image tag
	// +optional
	// +kubebuilder:default:="0.146.1"
	DefaultVersion *string `json:"defaultVersion,omitempty"`

	// Image pull secrets on platform cluster, synced to MCP
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Default compute resource requirements
	// +optional
	DefaultResources *corev1.ResourceRequirements `json:"defaultResources,omitempty"`

	// Default target namespace in MCP
	// +optional
	// +kubebuilder:default:="observability"
	DefaultNamespace *string `json:"defaultNamespace,omitempty"`
}

// ProviderConfigStatus defines the observed state of ProviderConfig.
type ProviderConfigStatus struct {
	// conditions represent the current state of the ProviderConfig resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ProviderConfig is the Schema for the providerconfigs API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:metadata:labels="openmcp.cloud/cluster=platform"
type ProviderConfig struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of ProviderConfig
	// +required
	Spec ProviderConfigSpec `json:"spec"`

	// status defines the observed state of ProviderConfig
	// +optional
	Status ProviderConfigStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// ProviderConfigList contains a list of ProviderConfig
type ProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProviderConfig{}, &ProviderConfigList{})
}

// PollInterval returns the poll interval duration from the spec.
func (o *ProviderConfig) PollInterval() time.Duration {
	if o.Spec.PollInterval == nil {
		return time.Minute
	}
	return o.Spec.PollInterval.Duration
}
