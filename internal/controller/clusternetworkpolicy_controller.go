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
	"errors"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8snetworkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	networkingv1 "github.com/Desuuuu/cluster-network-policy-operator/api/v1"
)

// ClusterNetworkPolicyReconciler reconciles a ClusterNetworkPolicy object
type ClusterNetworkPolicyReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	Recorder           record.EventRecorder
	ExcludedNamespaces Filters
	IncludedNamespaces Filters
}

//+kubebuilder:rbac:groups=networking.desuuuu.com,resources=clusternetworkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.desuuuu.com,resources=clusternetworkpolicies/finalizers,verbs=update

//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ClusterNetworkPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Reconciliation started")

	var clusterNetworkPolicy networkingv1.ClusterNetworkPolicy
	if err := r.Get(ctx, req.NamespacedName, &clusterNetworkPolicy); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("unable to fetch ClusterNetworkPolicy: %w", err)
	}

	replaceOnConflict := clusterNetworkPolicy.Annotations[networkingv1.ConflictAnnotation] == networkingv1.ConflictReplace

	namespaces, err := r.listNamespaces(ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to list namespaces: %w", err)
	}

	selector, err := metav1.LabelSelectorAsSelector(&clusterNetworkPolicy.Spec.NamespaceSelector)
	if err != nil {
		r.Recorder.Event(&clusterNetworkPolicy, corev1.EventTypeWarning, "InvalidConfiguration", "Invalid namespace selector")

		log.Error(err, "Invalid namespace selector")

		selector = labels.Nothing()
	}

	var errs []error

	for _, ns := range namespaces {
		networkPolicy := k8snetworkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterNetworkPolicy.Name,
				Namespace: ns.Name,
			},
		}

		if !selector.Matches(labels.Set(ns.Labels)) {
			if err := r.Get(ctx, client.ObjectKeyFromObject(&networkPolicy), &networkPolicy); err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}

				errs = append(errs, fmt.Errorf("unable to fetch NetworkPolicy: %w", err))
				continue
			}

			if !metav1.IsControlledBy(&networkPolicy, &clusterNetworkPolicy) {
				continue
			}

			if err := r.Delete(ctx, &networkPolicy, client.PropagationPolicy(metav1.DeletePropagationBackground)); !apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("unable to delete NetworkPolicy from namespace %s: %w", networkPolicy.Namespace, err))
				continue
			}

			r.Recorder.Event(&clusterNetworkPolicy, corev1.EventTypeNormal, "NetworkPolicyDeleted", fmt.Sprintf("NetworkPolicy deleted from namespace %s", networkPolicy.Namespace))

			log.Info("NetworkPolicy deleted", "namespace", networkPolicy.Namespace)
			continue
		}

		res, err := controllerutil.CreateOrPatch(ctx, r.Client, &networkPolicy, func() error {
			if !replaceOnConflict && networkPolicy.UID != types.UID("") && !metav1.IsControlledBy(&networkPolicy, &clusterNetworkPolicy) {
				r.Recorder.Event(&clusterNetworkPolicy, corev1.EventTypeWarning, "NetworkPolicyConflict", fmt.Sprintf("NetworkPolicy conflict in namespace %s", networkPolicy.Namespace))

				return errors.New("conflicting NetworkPolicy detected")
			}

			if err := ctrl.SetControllerReference(&clusterNetworkPolicy, &networkPolicy, r.Scheme); err != nil {
				return err
			}

			networkPolicy.Labels = clusterNetworkPolicy.Spec.Labels
			networkPolicy.Annotations = clusterNetworkPolicy.Spec.Annotations
			networkPolicy.Spec = clusterNetworkPolicy.Spec.NetworkPolicySpec

			return nil
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to create/update NetworkPolicy in namespace %s: %w", networkPolicy.Namespace, err))
			continue
		}

		switch res {
		case controllerutil.OperationResultCreated:
			r.Recorder.Event(&clusterNetworkPolicy, corev1.EventTypeNormal, "NetworkPolicyCreated", fmt.Sprintf("NetworkPolicy created in namespace %s", networkPolicy.Namespace))

			log.Info("NetworkPolicy created", "namespace", networkPolicy.Namespace)
		case controllerutil.OperationResultUpdated:
			r.Recorder.Event(&clusterNetworkPolicy, corev1.EventTypeNormal, "NetworkPolicyUpdated", fmt.Sprintf("NetworkPolicy updated in namespace %s", networkPolicy.Namespace))

			log.Info("NetworkPolicy updated", "namespace", networkPolicy.Namespace)
		}
	}

	err = utilerrors.NewAggregate(errs)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Reconciliation successful")

	return ctrl.Result{
		RequeueAfter: 6 * time.Hour,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterNetworkPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.ClusterNetworkPolicy{}).
		Owns(&k8snetworkingv1.NetworkPolicy{}).
		Watches(
			&corev1.Namespace{},
			handler.EnqueueRequestsFromMapFunc(r.onNamespaceUpdated),
			builder.WithPredicates(namespacePredicate{}),
		).
		Complete(r)
}

// listNamespaces returns all active namespaces that match the controller's
// namespace filters.
func (r *ClusterNetworkPolicyReconciler) listNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
	var namespaceList corev1.NamespaceList
	if err := r.List(ctx, &namespaceList); err != nil {
		return nil, err
	}

	res := make([]corev1.Namespace, 0, len(namespaceList.Items))
	for _, ns := range namespaceList.Items {
		if ns.Status.Phase != corev1.NamespaceActive {
			continue
		}

		if !EvaluateFilters(r.ExcludedNamespaces, r.IncludedNamespaces, ns.Name) {
			continue
		}

		res = append(res, ns)
	}

	return res, nil
}

// onNamespaceCreated is called when a namespace is created.
func (r *ClusterNetworkPolicyReconciler) onNamespaceUpdated(ctx context.Context, namespace client.Object) []ctrl.Request {
	if !EvaluateFilters(r.ExcludedNamespaces, r.IncludedNamespaces, namespace.GetName()) {
		return nil
	}

	var clusterNetworkPolicyList networkingv1.ClusterNetworkPolicyList
	if err := r.List(ctx, &clusterNetworkPolicyList); err != nil {
		return nil
	}

	res := make([]ctrl.Request, 0, len(clusterNetworkPolicyList.Items))

	for _, clusterNetworkPolicy := range clusterNetworkPolicyList.Items {
		res = append(res, ctrl.Request{
			NamespacedName: client.ObjectKeyFromObject(&clusterNetworkPolicy),
		})
	}

	return res
}

type namespacePredicate struct {
	predicate.Funcs
}

func (namespacePredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (namespacePredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	return !reflect.DeepEqual(e.ObjectNew.GetLabels(), e.ObjectOld.GetLabels())
}
