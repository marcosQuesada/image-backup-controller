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
	"github.com/marcosQuesada/image-backup-controller/pkg/registry"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/marcosQuesada/image-backup-controller/api/v1alpha1"
)

// ImageBackupReconciler reconciles a ImageBackup object
type ImageBackupReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder record.EventRecorder
	Registry registry.DockerRegistry
}

//+kubebuilder:rbac:groups=k8slab.io.k8slab.io,resources=imagebackups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8slab.io.k8slab.io,resources=imagebackups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8slab.io.k8slab.io,resources=imagebackups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ImageBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	ib := &v1alpha1.ImageBackup{}
	err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, ib)
	if err != nil && !errors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("unexpected error %w getting resource %s/%s", err, req.Namespace, req.Name)
	}

	r.Log.Info("Reconcile Image Backup", "key", req.NamespacedName, "status", ib.Status.Phase)

	if ib.Status.Phase == "" {
		now := metav1.Now()
		ib.Status.Phase = v1alpha1.PhasePending
		ib.Status.CreateAt = &now
		err = r.Status().Update(context.TODO(), ib)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	switch ib.Status.Phase {
	case v1alpha1.PhasePending:
		now := metav1.NewTime(time.Now())
		ib.Status.Phase = v1alpha1.PhaseRunning
		ib.Status.CreateAt = &now
		err = r.Status().Update(context.TODO(), ib)
		if err != nil {
			return ctrl.Result{}, err
		}
	case v1alpha1.PhaseRunning:
		if err := r.execute(ctx, ib); err != nil {
			return ctrl.Result{RequeueAfter: time.Second * 5}, err
		}
		d := metav1.Duration{Duration: time.Since(ib.Status.CreateAt.Time)}
		ib.Status.ExecutionDuration = &d
		ib.Status.Phase = v1alpha1.PhaseDone

		err = r.Status().Update(context.TODO(), ib)
		if err != nil {
			return ctrl.Result{}, err
		}
	case v1alpha1.PhaseDone:
		r.Log.Info("Image Backup Complete", "namespace", ib.Namespace, "name", ib.Name, "image", ib.Spec.Image)
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ImageBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pr := predicate.And(
		IgnoreDeleteEvents(),
		IgnoreGenericEvents(),
	)
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ImageBackup{}, builder.WithPredicates(pr)).
		Complete(r)
}

func (r *ImageBackupReconciler) execute(ctx context.Context, ib *v1alpha1.ImageBackup) error {
	newImage, err := r.Registry.BackupImageName(ib.Spec.Image)
	if err != nil {
		err = fmt.Errorf("unable to build image  %s new name, error %w", ib.Spec.Image, err)
		r.Log.Error(err, "execute", "image", ib.Spec.Image)
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, defaultExistenceCheckTimeout)
	exists, err := r.Registry.Exists(ctx, newImage)
	if err != nil {
		cancel()
		err = fmt.Errorf("unable to check image %s existence, error %w", ib.Spec.Image, err)
		r.Log.Error(err, "image", "processContainers", ib.Spec.Image, "newImage", newImage)
		return err
	}
	cancel()

	if !exists {
		r.Log.Info("Backup Image", "src", ib.Spec.Image, "dst", newImage)
		ctx, cancel := context.WithTimeout(ctx, defaultBackupTimeout)
		if err := r.Registry.Backup(ctx, ib.Spec.Image, newImage); err != nil {
			cancel()
			err = fmt.Errorf("unable to backup image %s, error %w", ib.Spec.Image, err)
			r.Log.Error(err, "image", "processContainers", ib.Spec.Image, "newImage", newImage)
			return err
		}
		cancel()
	}

	return nil
}
