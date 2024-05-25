/*
MIT License

Copyright (c) 2024 Desuuuu

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package v1

import (
	k8snetworkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConflictAnnotation = "networking.desuuuu.com/conflict-policy"
	ConflictReplace    = "replace"
)

// ClusterNetworkPolicySpec defines the desired state of ClusterNetworkPolicy
type ClusterNetworkPolicySpec struct {
	// Labels to apply to the NetworkPolicy resources.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations to apply to the NetworkPolicy resources.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// NamespaceSelector restricts the list of namespaces in which the
	// NetworkPolicy resources will be created. An empty namespaceSelector
	// matches all namespaces.
	// +optional
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector"`

	k8snetworkingv1.NetworkPolicySpec `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster

// ClusterNetworkPolicy is the Schema for the clusternetworkpolicies API
type ClusterNetworkPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterNetworkPolicySpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterNetworkPolicyList contains a list of ClusterNetworkPolicy
type ClusterNetworkPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterNetworkPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterNetworkPolicy{}, &ClusterNetworkPolicyList{})
}
