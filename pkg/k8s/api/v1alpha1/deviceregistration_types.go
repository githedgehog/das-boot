/*
Copyright 2023 The Hedgehog Authors.

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

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DeviceRegistrationSpec defines the properties of a device registration process
type DeviceRegistrationSpec struct {
	LocationUUID string `json:"locationUUID,omitempty"`
	CSR          []byte `json:"csr,omitempty"`
}

// DeviceRegistrationStatus defines the observed state of the device registration process
type DeviceRegistrationStatus struct {
	Certificate []byte `json:"certificate,omitempty"`
}

type RequestConditionType string

// These are the possible conditions for a certificate request.
const (
	CertificateApproved RequestConditionType = "Approved"
	CertificateDenied   RequestConditionType = "Denied"
	CertificateFailed   RequestConditionType = "Failed"
)

type CertificateSigningRequestCondition struct {
	// type of the condition. Known conditions include "Approved", "Denied", and "Failed".
	Type RequestConditionType
	// Status of the condition, one of True, False, Unknown.
	// Approved, Denied, and Failed conditions may not be "False" or "Unknown".
	// If unset, should be treated as "True".
	// +optional
	Status corev1.ConditionStatus
	// brief reason for the request state
	// +optional
	Reason string
	// human readable message with details about the request state
	// +optional
	Message string
	// timestamp for the last update to this condition
	// +optional
	LastUpdateTime metav1.Time
	// lastTransitionTime is the time the condition last transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=hedgehog;fabric,shortName=devreg;dr
// +kubebuilder:printcolumn:name="Location",type=string,JSONPath=`.spec.locationUUID`,priority=0
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,priority=0
// DeviceRegistration is the Schema for the device registration within DAS BOOT
type DeviceRegistration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeviceRegistrationSpec   `json:"spec,omitempty"`
	Status DeviceRegistrationStatus `json:"status,omitempty"`
}

const KindDeviceRegistration = "DeviceRegistration"

//+kubebuilder:object:root=true

// DeviceRegistrationList contains a list of DeviceRegistration
type DeviceRegistrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeviceRegistration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeviceRegistration{}, &DeviceRegistrationList{})
}
