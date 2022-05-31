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
	"github.com/marcosQuesada/image-backup-controller/pkg/registry"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const daemonSetControllerName = "image-backup-daemonset-controller"

// DaemonSetReconciler reconciles a DaemonSet object
type DaemonSetReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder record.EventRecorder
	Registry registry.DockerRegistry
}

// @TODO: core groups ?
//+kubebuilder:rbac:groups="";apps,resources=daemonsets,verbs=get;update;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	r.Log.Info("Reconcile DaemonSet", "key", req.NamespacedName)

	return ctrl.Result{}, nil
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
