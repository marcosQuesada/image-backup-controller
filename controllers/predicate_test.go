package controllers

import (
	"github.com/marcosQuesada/image-backup-controller/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"strings"
	"testing"
)

func TestIgnoreDeleteEvents(t *testing.T) {
	p := IgnoreDeleteEvents()

	if !p.Create(event.CreateEvent{
		Object: nil,
	}) {
		t.Error("Expected call")
	}
	if p.Delete(event.DeleteEvent{
		Object: nil,
	}) {
		t.Error("Not expected call")
	}
}

func TestIgnoreGenericEvents(t *testing.T) {
	p := IgnoreGenericEvents()

	if !p.Create(event.CreateEvent{
		Object: nil,
	}) {
		t.Error("Expected call")
	}
	if p.Generic(event.GenericEvent{
		Object: nil,
	}) {
		t.Error("Not expected call")
	}
}

func TestIgnoreRestrictedNamespaces(t *testing.T) {
	restrictedNamespace := "foo"
	restricted := []string{restrictedNamespace}

	p := IgnoreRestrictedNamespaces(restricted)
	if !p.Create(event.CreateEvent{
		Object: &v1alpha1.ImageBackup{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "xxx",
			},
		},
	}) {
		t.Error("Expected call")
	}
	if p.Create(event.CreateEvent{
		Object: &v1alpha1.ImageBackup{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: restrictedNamespace,
			},
		},
	}) {
		t.Error("Not expected call")
	}
}

func TestDeploymentHasNonBackupImage(t *testing.T) {
	backupRegistry := "foo"
	fn := func(img string) bool {
		return !strings.HasPrefix(img, backupRegistry)
	}
	p := DeploymentHasNonBackupImage(fn)
	if !p.Create(event.CreateEvent{
		Object: getFakePod("default", "goo", "goo/bar:1.2.3"),
	}) {
		t.Error("Expected call")
	}

	if p.Create(event.CreateEvent{
		Object: getFakePod("default", "goo", backupRegistry+"/bar:1.2.3"),
	}) {
		t.Error("Not expected call")
	}

}

func TestDaemonSetHasNonBackupImage(t *testing.T) {
	backupRegistry := "foo"
	fn := func(img string) bool {
		return !strings.HasPrefix(img, backupRegistry)
	}
	p := DaemonSetHasNonBackupImage(fn)
	if !p.Create(event.CreateEvent{
		Object: getFakeDaemonSet("default", "goo", "goo/bar:1.2.3"),
	}) {
		t.Error("Expected call")
	}

	if p.Create(event.CreateEvent{
		Object: getFakeDaemonSet("default", "goo", backupRegistry+"/bar:1.2.3"),
	}) {
		t.Error("Not expected call")
	}

}

func getFakePod(ns, name, img string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: img,
						},
					},
				},
			},
		},
	}
}

func getFakeDaemonSet(ns, name, img string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: img,
						},
					},
				},
			},
		},
	}
}
