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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const defaultRequeueDuration = time.Second * 5

// ImagePredicateFilter filters non image backup
type ImagePredicateFilter interface {
	IsNonImageBackup(image string) bool
}

// GenericReconciler reconciles workload objects
type GenericReconciler struct {
	client.Client
	Log      logr.Logger
	Registry registry.DockerRegistry
}

func (r *GenericReconciler) reconcile(ctx context.Context, req ctrl.Request, obj client.Object) (ctrl.Result, error) {
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			// resource has been deleted, skip
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("unable to get resource %s error %v", req.NamespacedName, err)
	}

	processing, newInitContainersUpdated, err := r.processContainers(ctx, req.Namespace, req.Name, initContainers(obj), obj)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to process initcontainers, error %w", err)
	}

	if processing {
		return ctrl.Result{RequeueAfter: defaultRequeueDuration}, nil
	}

	processing, newContainersUpdated, err := r.processContainers(ctx, req.Namespace, req.Name, containers(obj), obj)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to process containers, error %w", err)
	}

	if processing {
		return ctrl.Result{RequeueAfter: defaultRequeueDuration}, nil
	}

	if !newInitContainersUpdated && !newContainersUpdated {
		return ctrl.Result{}, nil
	}

	r.Log.Info("Rolling out updated Backup image", "resource", req.NamespacedName)
	if err := r.Update(ctx, obj); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("resource has been deleted before updating", "resource", req.NamespacedName)
			return ctrl.Result{}, nil
		}

		if errors.IsConflict(err) {
			// On conflict wait 1 second and retry
			r.Log.Info("resource update conflict, requeue", "resource", req.NamespacedName)
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}

		r.Log.Error(err, "unexpected error", "resource", req.NamespacedName)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *GenericReconciler) processContainers(ctx context.Context, ns, name string, cs []corev1.Container, obj client.Object) (processing bool, needsUpdate bool, err error) {
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
			if err := r.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, obj); err != nil {
				if errors.IsNotFound(err) {
					// Workload has been deleted, skip
					return false, false, nil
				}

				return true, false, fmt.Errorf("unexpected error getting resource %s error %w", ns+"/"+name, err)
			}

			ib = newImageBackup(imageBackupNamespace, ibName, container.Image)
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

		r.Log.Info("Updating image", "resource", ns+"/"+name, "from", cs[i].Image, "to", newImage) // @TODO: Fire event?
		cs[i].Image = newImage
		needsUpdate = true
	}

	return processing, needsUpdate, nil
}

func newImageBackup(ns, name, img string) *v1alpha1.ImageBackup {
	return &v1alpha1.ImageBackup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
		Spec: v1alpha1.ImageBackupSpec{
			Image: img,
		},
	}
}

func initContainers(obj client.Object) []corev1.Container {
	switch o := obj.(type) {
	case *appsv1.Deployment:
		return o.Spec.Template.Spec.InitContainers
	case *appsv1.DaemonSet:
		return o.Spec.Template.Spec.InitContainers
	default:
		return []corev1.Container{}
	}
}

func containers(obj client.Object) []corev1.Container {
	switch o := obj.(type) {
	case *appsv1.Deployment:
		return o.Spec.Template.Spec.Containers
	case *appsv1.DaemonSet:
		return o.Spec.Template.Spec.Containers
	default:
		return []corev1.Container{}
	}
}
