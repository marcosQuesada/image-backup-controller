package controllers

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// IgnoreDeleteEvents predicate filters delete events
func IgnoreDeleteEvents() predicate.Predicate {
	return predicate.Funcs{
		DeleteFunc: func(ev event.DeleteEvent) bool {
			return false
		},
	}
}

// IgnoreGenericEvents Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
// external trigger request - e.g. reconcile Autoscaling, or a Webhook.
func IgnoreGenericEvents() predicate.Predicate {
	return predicate.Funcs{
		GenericFunc: func(ev event.GenericEvent) bool {
			return false
		},
	}
}

func IgnoreRestrictedNamespaces(restricted []string) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(ev event.CreateEvent) bool {
			return !isRestrictedNamespace(restricted, ev.Object.GetNamespace())
		},
		UpdateFunc: func(ev event.UpdateEvent) bool {
			return !isRestrictedNamespace(restricted, ev.ObjectNew.GetNamespace())
		},
	}
}

func isRestrictedNamespace(restricted []string, namespace string) bool {
	for _, s := range restricted {
		if namespace == s {
			return true
		}
	}

	return false
}

func DeploymentReady() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(ev event.CreateEvent) bool {
			return isDeploymentReady(ev.Object)
		},
		UpdateFunc: func(ev event.UpdateEvent) bool {
			return isDeploymentReady(ev.ObjectNew)
		},
	}
}

func isDeploymentReady(o runtime.Object) bool {
	d, ok := o.(*appsv1.Deployment)
	if !ok {
		return false
	}

	if d.Status.Replicas == 0 {
		return false
	}

	return d.Status.Replicas == d.Status.ReadyReplicas
}

func DaemonSetReady() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(ev event.CreateEvent) bool {
			return isDaemonSetReady(ev.Object)
		},
		UpdateFunc: func(ev event.UpdateEvent) bool {
			return isDaemonSetReady(ev.ObjectNew)
		},
	}
}

func isDaemonSetReady(o runtime.Object) bool {
	d, ok := o.(*appsv1.DaemonSet)
	if !ok {
		return false
	}

	if d.Status.DesiredNumberScheduled == 0 {
		return false
	}

	return d.Status.DesiredNumberScheduled == d.Status.NumberReady
}

func DeploymentHasNonBackupImage(isNonBackupImage func(string) bool) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(ev event.CreateEvent) bool {
			return hasDeploymentNonBackupImage(ev.Object, isNonBackupImage)
		},
		UpdateFunc: func(ev event.UpdateEvent) bool {
			return hasDeploymentNonBackupImage(ev.ObjectNew, isNonBackupImage)
		},
	}
}

func hasDeploymentNonBackupImage(o runtime.Object, isNonBackupImage func(string) bool) bool {
	d, ok := o.(*appsv1.Deployment)
	if !ok {
		return false
	}

	for _, container := range d.Spec.Template.Spec.InitContainers {
		if isNonBackupImage(container.Image) {
			return true
		}
	}

	for _, container := range d.Spec.Template.Spec.Containers {
		if isNonBackupImage(container.Image) {
			return true
		}
	}

	return false
}

func DaemonSetHasNonBackupImage(isNonBackupImage func(string) bool) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(ev event.CreateEvent) bool {
			return hasDaemonSetNonBackupImage(ev.Object, isNonBackupImage)
		},
		UpdateFunc: func(ev event.UpdateEvent) bool {
			return hasDaemonSetNonBackupImage(ev.ObjectNew, isNonBackupImage)
		},
	}
}

func hasDaemonSetNonBackupImage(o runtime.Object, isNonBackupImage func(string) bool) bool {
	d, ok := o.(*appsv1.DaemonSet)
	if !ok {
		return false
	}

	for _, container := range d.Spec.Template.Spec.InitContainers {
		if isNonBackupImage(container.Image) {
			return true
		}
	}

	for _, container := range d.Spec.Template.Spec.Containers {
		if isNonBackupImage(container.Image) {
			return true
		}
	}

	return false
}
