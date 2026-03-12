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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OtelCollectorServiceSpec defines the desired state of OtelCollectorService
type OtelCollectorServiceSpec struct {
	// Override the default collector image from ProviderConfig
	// +optional
	CollectorImage *string `json:"collectorImage,omitempty"`

	// Override the default collector image tag from ProviderConfig
	// +optional
	CollectorVersion *string `json:"collectorVersion,omitempty"`

	// Override default CPU/memory resource requirements from ProviderConfig
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// Override the target namespace in the MCP (default from ProviderConfig, typically "observability")
	// +optional
	Namespace *string `json:"namespace,omitempty"`
}

// OtelCollectorServiceStatus defines the observed state of OtelCollectorService.
type OtelCollectorServiceStatus struct {
	// conditions represent the current state of the OtelCollectorService resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// ObservedGeneration is the generation of this resource that was last reconciled by the controller.
	ObservedGeneration int64 `json:"observedGeneration"`
	// Phase is the current phase of the resource.
	Phase string `json:"phase"`
}

// OtelCollectorService is the Schema for the otelcollectorservices API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=`.status.phase`,name="Phase",type=string
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:metadata:labels="openmcp.cloud/cluster=onboarding"
type OtelCollectorService struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of OtelCollectorService
	// +required
	Spec OtelCollectorServiceSpec `json:"spec"`

	// status defines the observed state of OtelCollectorService
	// +optional
	Status OtelCollectorServiceStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// OtelCollectorServiceList contains a list of OtelCollectorService
type OtelCollectorServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OtelCollectorService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OtelCollectorService{}, &OtelCollectorServiceList{})
}

// Finalizer returns the finalizer string for the OtelCollectorService resource
func (o *OtelCollectorService) Finalizer() string {
	return GroupVersion.Group + "/finalizer"
}

// GetStatus returns the status of the OtelCollectorService resource
func (o *OtelCollectorService) GetStatus() any {
	return o.Status
}

// GetConditions returns the conditions of the OtelCollectorService resource
func (o *OtelCollectorService) GetConditions() *[]metav1.Condition {
	return &o.Status.Conditions
}

// SetPhase sets the phase of the OtelCollectorService resource status
func (o *OtelCollectorService) SetPhase(phase string) {
	o.Status.Phase = phase
}

// SetObservedGeneration sets the observed generation of the OtelCollectorService resource
func (o *OtelCollectorService) SetObservedGeneration(gen int64) {
	o.Status.ObservedGeneration = gen
}
