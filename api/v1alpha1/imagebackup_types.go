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

const (
	PhasePending = "PENDING"
	PhaseRunning = "RUNNING"
	PhaseDone    = "DONE"

	DeploymentResourceType = "deployment"
	DaemonSetResourceType  = "daemonset"
)

// ImageBackupSpec defines the desired state of ImageBackup
type ImageBackupSpec struct {
	Image        string `json:"image,omitempty"`
	ResourceName string `json:"resource-name,omitempty"`
	ResourceType string `json:"resource-type,omitempty"`
}

// ImageBackupStatus defines the observed state of ImageBackup
type ImageBackupStatus struct {
	Phase             string           `json:"phase,omitempty"`
	CreateAt          *metav1.Time     `json:"create_at,omitempty"`
	ExecutionDuration *metav1.Duration `json:"duration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ImageBackup is the Schema for the imagebackups API
type ImageBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageBackupSpec   `json:"spec,omitempty"`
	Status ImageBackupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ImageBackupList contains a list of ImageBackup
type ImageBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImageBackup{}, &ImageBackupList{})
}
