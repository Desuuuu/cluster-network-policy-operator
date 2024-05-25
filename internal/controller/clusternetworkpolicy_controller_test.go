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

package controller

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	k8snetworkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	networkingv1 "github.com/Desuuuu/cluster-network-policy-operator/api/v1"
)

var _ = Describe("ClusterNetworkPolicy Controller", func() {
	const (
		timeout  = 5 * time.Second
		interval = time.Second
	)

	Context("creating a ClusterNetworkPolicy", func() {
		var (
			testNamespace     string
			conflictNamespace string
		)

		BeforeEach(func(ctx context.Context) {
			testNamespace, conflictNamespace = random("test"), random("conflict")

			err := k8sClient.Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: conflictNamespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			networkPolicy := conflictingNetworkPolicy.DeepCopy()
			networkPolicy.Namespace = conflictNamespace

			err = k8sClient.Create(ctx, networkPolicy)
			Expect(err).NotTo(HaveOccurred())

			clusterNetworkPolicy := basicClusterNetworkPolicy.DeepCopy()

			err = k8sClient.Create(ctx, clusterNetworkPolicy)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func(ctx context.Context) {
			err := k8sClient.Delete(ctx, basicClusterNetworkPolicy.DeepCopy())
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create NetworkPolicy resources", func(ctx context.Context) {
			resource := &networkingv1.ClusterNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: basicClusterNetworkPolicy.Name,
				},
			}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(resource), resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.UID).NotTo(BeEmpty())

			networkPolicy := &k8snetworkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resource.Name,
					Namespace: testNamespace,
				},
			}

			Eventually(func(g Gomega, ctx context.Context) {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(networkPolicy), networkPolicy)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(networkPolicy.OwnerReferences).To(HaveLen(1))
				g.Expect(networkPolicy.OwnerReferences[0].Controller).NotTo(BeNil())
				g.Expect(*networkPolicy.OwnerReferences[0].Controller).To(BeTrue())
				g.Expect(networkPolicy.OwnerReferences[0].UID).To(Equal(resource.UID))
				g.Expect(networkPolicy.Labels).To(Equal(basicClusterNetworkPolicy.Spec.Labels))
				g.Expect(networkPolicy.Annotations).To(Equal(basicClusterNetworkPolicy.Spec.Annotations))
				g.Expect(networkPolicy.Spec).To(Equal(basicClusterNetworkPolicy.Spec.NetworkPolicySpec))
			}, timeout, interval).WithContext(ctx).Should(Succeed())
		})

		It("should not overwrite existing NetworkPolicy resources", func(ctx context.Context) {
			resource := &networkingv1.ClusterNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: basicClusterNetworkPolicy.Name,
				},
			}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(resource), resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.UID).NotTo(BeEmpty())

			networkPolicy := &k8snetworkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resource.Name,
					Namespace: conflictNamespace,
				},
			}

			Consistently(func(g Gomega, ctx context.Context) {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(networkPolicy), networkPolicy)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(networkPolicy.OwnerReferences).To(BeEmpty())
				g.Expect(networkPolicy.Labels).To(Equal(conflictingNetworkPolicy.Labels))
				g.Expect(networkPolicy.Annotations).To(Equal(conflictingNetworkPolicy.Annotations))
				g.Expect(networkPolicy.Spec).To(Equal(conflictingNetworkPolicy.Spec))
			}, timeout, interval).WithContext(ctx).Should(Succeed())
		})

		It("should not create NetworkPolicy resources in excluded namespaces", func(ctx context.Context) {
			resource := &networkingv1.ClusterNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: basicClusterNetworkPolicy.Name,
				},
			}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(resource), resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.UID).NotTo(BeEmpty())

			networkPolicy := &k8snetworkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resource.Name,
					Namespace: "kube-system",
				},
			}

			Consistently(func(g Gomega, ctx context.Context) {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(networkPolicy), networkPolicy)
				g.Expect(err).To(HaveOccurred())
			}, timeout, interval).WithContext(ctx).Should(Succeed())
		})
	})

	Context("creating a ClusterNetworkPolicy with replace policy", func() {
		var (
			testNamespace     string
			conflictNamespace string
		)

		BeforeEach(func(ctx context.Context) {
			testNamespace, conflictNamespace = random("test"), random("conflict")

			err := k8sClient.Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: conflictNamespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			networkPolicy := conflictingNetworkPolicy.DeepCopy()
			networkPolicy.Namespace = conflictNamespace

			err = k8sClient.Create(ctx, networkPolicy)
			Expect(err).NotTo(HaveOccurred())

			clusterNetworkPolicy := basicClusterNetworkPolicy.DeepCopy()
			clusterNetworkPolicy.Annotations = map[string]string{
				"networking.desuuuu.com/conflict-policy": "replace",
			}

			err = k8sClient.Create(ctx, clusterNetworkPolicy)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func(ctx context.Context) {
			err := k8sClient.Delete(ctx, basicClusterNetworkPolicy.DeepCopy())
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create NetworkPolicy resources", func(ctx context.Context) {
			resource := &networkingv1.ClusterNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: basicClusterNetworkPolicy.Name,
				},
			}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(resource), resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.UID).NotTo(BeEmpty())

			networkPolicy := &k8snetworkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resource.Name,
					Namespace: testNamespace,
				},
			}

			Eventually(func(g Gomega, ctx context.Context) {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(networkPolicy), networkPolicy)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(networkPolicy.OwnerReferences).To(HaveLen(1))
				g.Expect(networkPolicy.OwnerReferences[0].Controller).NotTo(BeNil())
				g.Expect(*networkPolicy.OwnerReferences[0].Controller).To(BeTrue())
				g.Expect(networkPolicy.OwnerReferences[0].UID).To(Equal(resource.UID))
				g.Expect(networkPolicy.Labels).To(Equal(basicClusterNetworkPolicy.Spec.Labels))
				g.Expect(networkPolicy.Annotations).To(Equal(basicClusterNetworkPolicy.Spec.Annotations))
				g.Expect(networkPolicy.Spec).To(Equal(basicClusterNetworkPolicy.Spec.NetworkPolicySpec))
			}, timeout, interval).WithContext(ctx).Should(Succeed())
		})

		It("should overwrite existing NetworkPolicy resources", func(ctx context.Context) {
			resource := &networkingv1.ClusterNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: basicClusterNetworkPolicy.Name,
				},
			}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(resource), resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.UID).NotTo(BeEmpty())

			networkPolicy := &k8snetworkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resource.Name,
					Namespace: conflictNamespace,
				},
			}

			Eventually(func(g Gomega, ctx context.Context) {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(networkPolicy), networkPolicy)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(networkPolicy.OwnerReferences).To(HaveLen(1))
				g.Expect(networkPolicy.OwnerReferences[0].Controller).NotTo(BeNil())
				g.Expect(*networkPolicy.OwnerReferences[0].Controller).To(BeTrue())
				g.Expect(networkPolicy.OwnerReferences[0].UID).To(Equal(resource.UID))
				g.Expect(networkPolicy.Labels).To(Equal(basicClusterNetworkPolicy.Spec.Labels))
				g.Expect(networkPolicy.Annotations).To(Equal(basicClusterNetworkPolicy.Spec.Annotations))
				g.Expect(networkPolicy.Spec).To(Equal(basicClusterNetworkPolicy.Spec.NetworkPolicySpec))
			}, timeout, interval).WithContext(ctx).Should(Succeed())
		})
	})

	Context("creating a ClusterNetworkPolicy with namespace selectors", func() {
		var (
			testNamespace     string
			ignoredNamespace  string
			conflictNamespace string
		)

		BeforeEach(func(ctx context.Context) {
			testNamespace, ignoredNamespace, conflictNamespace = random("test"), random("ignored"), random("conflict")

			err := k8sClient.Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
					Labels: map[string]string{
						"create-networkpolicy": "true",
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ignoredNamespace,
					Labels: map[string]string{
						"create-networkpolicy": "false",
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: conflictNamespace,
					Labels: map[string]string{
						"create-networkpolicy": "true",
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			networkPolicy := conflictingNetworkPolicy.DeepCopy()
			networkPolicy.Namespace = conflictNamespace

			err = k8sClient.Create(ctx, networkPolicy)
			Expect(err).NotTo(HaveOccurred())

			clusterNetworkPolicy := basicClusterNetworkPolicy.DeepCopy()
			clusterNetworkPolicy.Spec.NamespaceSelector = metav1.LabelSelector{
				MatchLabels: map[string]string{
					"create-networkpolicy": "true",
				},
			}

			err = k8sClient.Create(ctx, clusterNetworkPolicy)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func(ctx context.Context) {
			err := k8sClient.Delete(ctx, basicClusterNetworkPolicy.DeepCopy())
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create NetworkPolicy resources in matching namespaces", func(ctx context.Context) {
			resource := &networkingv1.ClusterNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: basicClusterNetworkPolicy.Name,
				},
			}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(resource), resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.UID).NotTo(BeEmpty())

			networkPolicy := &k8snetworkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resource.Name,
					Namespace: testNamespace,
				},
			}

			Eventually(func(g Gomega, ctx context.Context) {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(networkPolicy), networkPolicy)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(networkPolicy.OwnerReferences).To(HaveLen(1))
				g.Expect(networkPolicy.OwnerReferences[0].Controller).NotTo(BeNil())
				g.Expect(*networkPolicy.OwnerReferences[0].Controller).To(BeTrue())
				g.Expect(networkPolicy.OwnerReferences[0].UID).To(Equal(resource.UID))
				g.Expect(networkPolicy.Labels).To(Equal(basicClusterNetworkPolicy.Spec.Labels))
				g.Expect(networkPolicy.Annotations).To(Equal(basicClusterNetworkPolicy.Spec.Annotations))
				g.Expect(networkPolicy.Spec).To(Equal(basicClusterNetworkPolicy.Spec.NetworkPolicySpec))
			}, timeout, interval).WithContext(ctx).Should(Succeed())
		})

		It("should not create NetworkPolicy resources in non-matching namespaces", func(ctx context.Context) {
			resource := &networkingv1.ClusterNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: basicClusterNetworkPolicy.Name,
				},
			}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(resource), resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.UID).NotTo(BeEmpty())

			networkPolicy := &k8snetworkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resource.Name,
					Namespace: ignoredNamespace,
				},
			}

			Consistently(func(g Gomega, ctx context.Context) {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(networkPolicy), networkPolicy)
				g.Expect(err).To(HaveOccurred())
			}, timeout, interval).WithContext(ctx).Should(Succeed())
		})

		It("should not overwrite existing NetworkPolicy resources in matching namespaces", func(ctx context.Context) {
			resource := &networkingv1.ClusterNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: basicClusterNetworkPolicy.Name,
				},
			}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(resource), resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.UID).NotTo(BeEmpty())

			networkPolicy := &k8snetworkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resource.Name,
					Namespace: conflictNamespace,
				},
			}

			Consistently(func(g Gomega, ctx context.Context) {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(networkPolicy), networkPolicy)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(networkPolicy.OwnerReferences).To(BeEmpty())
				g.Expect(networkPolicy.Labels).To(Equal(conflictingNetworkPolicy.Labels))
				g.Expect(networkPolicy.Annotations).To(Equal(conflictingNetworkPolicy.Annotations))
				g.Expect(networkPolicy.Spec).To(Equal(conflictingNetworkPolicy.Spec))
			}, timeout, interval).WithContext(ctx).Should(Succeed())
		})

		It("should not create NetworkPolicy resources in excluded namespaces", func(ctx context.Context) {
			resource := &networkingv1.ClusterNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: basicClusterNetworkPolicy.Name,
				},
			}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(resource), resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.UID).NotTo(BeEmpty())

			networkPolicy := &k8snetworkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resource.Name,
					Namespace: "kube-system",
				},
			}

			Consistently(func(g Gomega, ctx context.Context) {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(networkPolicy), networkPolicy)
				g.Expect(err).To(HaveOccurred())
			}, timeout, interval).WithContext(ctx).Should(Succeed())
		})
	})
})

var networkPolicySpec = k8snetworkingv1.NetworkPolicySpec{
	PodSelector: metav1.LabelSelector{
		MatchLabels: map[string]string{
			"role": "db",
		},
	},
	PolicyTypes: []k8snetworkingv1.PolicyType{
		k8snetworkingv1.PolicyTypeIngress,
		k8snetworkingv1.PolicyTypeEgress,
	},
	Ingress: []k8snetworkingv1.NetworkPolicyIngressRule{
		{
			From: []k8snetworkingv1.NetworkPolicyPeer{
				{
					IPBlock: &k8snetworkingv1.IPBlock{
						CIDR:   "172.17.0.0/16",
						Except: []string{"172.17.1.0/24"},
					},
				},
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"project": "myproject",
						},
					},
				},
				{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"role": "frontend",
						},
					},
				},
			},
			Ports: []k8snetworkingv1.NetworkPolicyPort{
				{
					Protocol: ptr(corev1.ProtocolTCP),
					Port:     ptr(intstr.FromInt(6379)),
				},
			},
		},
	},
	Egress: []k8snetworkingv1.NetworkPolicyEgressRule{
		{
			To: []k8snetworkingv1.NetworkPolicyPeer{
				{
					IPBlock: &k8snetworkingv1.IPBlock{
						CIDR: "10.0.0.0/24",
					},
				},
			},
			Ports: []k8snetworkingv1.NetworkPolicyPort{
				{
					Protocol: ptr(corev1.ProtocolTCP),
					Port:     ptr(intstr.FromInt(5978)),
				},
			},
		},
	},
}

var basicClusterNetworkPolicy = &networkingv1.ClusterNetworkPolicy{
	ObjectMeta: metav1.ObjectMeta{
		Name: "test-clusternetworkpolicy",
	},
	Spec: networkingv1.ClusterNetworkPolicySpec{
		Labels: map[string]string{
			"my-label": "label-value1",
		},
		Annotations: map[string]string{
			"my-annotation": "annotation-value1",
		},
		NetworkPolicySpec: networkPolicySpec,
	},
}

var conflictingNetworkPolicy = &k8snetworkingv1.NetworkPolicy{
	ObjectMeta: metav1.ObjectMeta{
		Name: basicClusterNetworkPolicy.Name,
	},
	Spec: k8snetworkingv1.NetworkPolicySpec{
		PolicyTypes: []k8snetworkingv1.PolicyType{
			k8snetworkingv1.PolicyTypeIngress,
		},
		Ingress: []k8snetworkingv1.NetworkPolicyIngressRule{
			{
				From: []k8snetworkingv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{},
					},
				},
			},
		},
	},
}

func ptr[T any](v T) *T {
	return &v
}

func random(prefix string) string {
	r, err := rand.Int(rand.Reader, big.NewInt(999999))
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s-%06d", prefix, r.Int64())
}
