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

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const daemonSetControllerName = "image-backup-daemonset-controller"

// DaemonSetReconciler reconciles a DaemonSet object
type DaemonSetReconciler struct {
	*GenericReconciler
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder record.EventRecorder
}

// @TODO: core groups ?
//+kubebuilder:rbac:groups="";apps,resources=daemonsets,verbs=get;update;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("Reconcile DaemonSet", "key", req.NamespacedName)

	dms := &appsv1.DaemonSet{}
	return r.reconcile(ctx, req, dms)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DaemonSetReconciler) SetupWithManager(mgr ctrl.Manager, fn ImagePredicateFilter, banNs []string) error {
	pr := predicate.And(
		IgnoreDeleteEvents(),
		IgnoreGenericEvents(),
		IgnoreRestrictedNamespaces(banNs),
		DaemonSetReady(),
		DaemonSetHasNonBackupImage(fn.IsNonImageBackup),
	)
	return ctrl.NewControllerManagedBy(mgr).
		Named(daemonSetControllerName).
		For(&appsv1.DaemonSet{}, builder.WithPredicates(pr)).
		Complete(r)
}
