//go:build integration
// +build integration

package controllers

import (
	"context"
	"github.com/marcosQuesada/image-backup-controller/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("Run Image Backup Controller", func() {
	const timeout = time.Second * 3
	const interval = time.Millisecond * 500

	namespace := "default"
	name := "nginx"
	image := "nginx:1.14.2"
	Context("Run a new Image Backup", func() {
		It("Should create successfully", func() {
			ib := &v1alpha1.ImageBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
				},
				Spec: v1alpha1.ImageBackupSpec{
					Image: image,
				},
			}
			Expect(k8sClient.Create(context.Background(), ib)).Should(Succeed())
			key := types.NamespacedName{Namespace: namespace, Name: name}

			By("Describing Initial Image Backup State")
			Eventually(func() bool {
				r := &v1alpha1.ImageBackup{}
				k8sClient.Get(context.Background(), key, r)
				return r.Status.Phase == v1alpha1.PhasePending
			}, time.Millisecond*100, interval).Should(BeTrue())

			By("Describing Progressing Image Backup State")
			Eventually(func() bool {
				r := &v1alpha1.ImageBackup{}
				k8sClient.Get(context.Background(), key, r)
				return r.Status.Phase == v1alpha1.PhaseRunning
			}, time.Second, interval).Should(BeTrue())

			By("Describing Image Backup Completion")
			Eventually(func() bool {
				r := &v1alpha1.ImageBackup{}
				k8sClient.Get(context.Background(), key, r)
				return r.Status.Phase == v1alpha1.PhaseDone
			}, timeout, interval).Should(BeTrue())
		})
	})
})
