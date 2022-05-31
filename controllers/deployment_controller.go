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
	"fmt"
	"github.com/go-logr/logr"
	"github.com/marcosQuesada/image-backup-controller/api/v1alpha1"
	"github.com/marcosQuesada/image-backup-controller/pkg/registry"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const defaultExistenceCheckTimeout = time.Second * 10
const defaultBackupTimeout = time.Second * 300

// ImagePredicateFilter filters non image backup
type ImagePredicateFilter interface {
	IsNonImageBackup(image string) bool
}

// DeploymentReconciler reconciles a Deployment object
type DeploymentReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder record.EventRecorder
	Registry registry.DockerRegistry
}

//+kubebuilder:rbac:groups="";apps,resources=deployments,verbs=get;list;update;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("Deployment Reconcile", "signal", req.NamespacedName, "type", "deployment")

	dpl := &appsv1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, dpl); err != nil {
		if errors.IsNotFound(err) {
			// Deployment has been deleted, skip
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("unable to get deploy %s error %v", req.NamespacedName, err)
	}

	processing, newInitContainersUpdated, err := r.processContainers(ctx, req.Namespace, req.Name, dpl.Spec.Template.Spec.InitContainers)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to process initcontainers, error %w", err)
	}

	if processing {
		return ctrl.Result{RequeueAfter: time.Second * 2}, nil
	}

	processing, newContainersUpdated, err := r.processContainers(ctx, req.Namespace, req.Name, dpl.Spec.Template.Spec.Containers)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to process containers, error %w", err)
	}

	if processing {
		return ctrl.Result{RequeueAfter: time.Second * 2}, nil
	}

	r.Log.Info("Deployment", "key", req.NamespacedName.String(), "image", dpl.Spec.Template.Spec.Containers[0].Image)

	//	if !needsUpdate {
	if !newInitContainersUpdated && !newContainersUpdated {
		return ctrl.Result{}, nil
	}

	r.Log.Info("Rolling out updated Backup image", "resource", req.NamespacedName)
	if err := r.Update(ctx, dpl); err != nil {
		if errors.IsNotFound(err) {
			// Deployment has been deleted, skip
			r.Log.Info("deployment has been deleted before update", "resource", req.NamespacedName)
			return ctrl.Result{}, nil
		}

		if errors.IsConflict(err) {
			// On conflict wait 1 second and retry
			r.Log.Info("deployment update conflict, requeue", "resource", req.NamespacedName)
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}

		r.Log.Error(err, "unexpected error", "resource", req.NamespacedName)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager, fn ImagePredicateFilter, banNs []string) error {
	pr := predicate.And(
		IgnoreDeleteEvents(),
		IgnoreGenericEvents(),
		IgnoreRestrictedNamespaces(banNs),
		DeploymentReady(),
		DeploymentHasNonBackupImage(fn.IsNonImageBackup),
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}, builder.WithPredicates(pr)).
		Complete(r)
}

func (r *DeploymentReconciler) processContainers(ctx context.Context, ns, name string, cs []corev1.Container) (bool, bool, error) {
	needsUpdate := false
	processing := false
	for i, container := range cs {
		if !r.Registry.IsNonImageBackup(container.Image) {
			continue
		}

		ib := &v1alpha1.ImageBackup{}
		ibName := v1alpha1.ImageBackupNameFromImage(container.Image)
		err := r.Get(ctx, types.NamespacedName{Namespace: imageBackupNamespace, Name: ibName}, ib)
		if err != nil && !errors.IsNotFound(err) {
			return false, false, fmt.Errorf("unexpected error %w getting resource %s/%s", err, ns, name)
		}

		if errors.IsNotFound(err) {
			r.Log.Info("No ImageBackup found, create it", "key", ns+"/"+name)
			dpl := &appsv1.Deployment{}
			if err := r.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, dpl); err != nil {
				if errors.IsNotFound(err) {
					// Deployment has been deleted, skip
					return false, false, nil
				}

				return true, false, fmt.Errorf("unexpected error getting deploy %s error %w", ns+"/"+name, err)
			}

			ib = newImageBackup(imageBackupNamespace, ibName, container.Image, ns+"/"+name, v1alpha1.DeploymentResourceType)
			if err := r.Create(ctx, ib); err != nil {
				if errors.IsAlreadyExists(err) {
					r.Log.Info("ImageBackup already exists", "key", ibName)
					return true, false, nil
				}
				return true, false, fmt.Errorf("unable to create resource %s error %w", ns+"/"+name, err)
			}

			processing = true
			continue
		}

		if ib.Status.Phase != v1alpha1.PhaseDone {
			processing = true
			continue
		}

		newImage, err := r.Registry.BackupImageName(container.Image)
		if err != nil {
			err = fmt.Errorf("unable to build image  %s new name, error %w", container.Image, err)
			r.Log.Error(err, "imageName", "processContainers", container.Image, "newImage", newImage)
			return false, false, err
		}

		r.Log.Info("Updating image", "deployment", ns+"/"+name, "from", cs[i].Image, "to", newImage)
		cs[i].Image = newImage
		needsUpdate = true
	}

	return processing, needsUpdate, nil
}

func newImageBackup(ns, name, img, rn, rt string) *v1alpha1.ImageBackup {
	return &v1alpha1.ImageBackup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
		Spec: v1alpha1.ImageBackupSpec{
			Image:        img,
			ResourceName: rn,
			ResourceType: rt,
		},
	}
}
