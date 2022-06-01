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

package main

import (
	"errors"
	"flag"
	"github.com/marcosQuesada/image-backup-controller/pkg/registry"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	k8slabiov1alpha1 "github.com/marcosQuesada/image-backup-controller/api/v1alpha1"
	"github.com/marcosQuesada/image-backup-controller/controllers"
	//+kubebuilder:scaffold:imports
)

const kubeSystemNamespace = "kube-system"

var (
	scheme         = runtime.NewScheme()
	setupLog       = ctrl.Log.WithName("setup")
	backupRegistry string
	username       string
	token          string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(k8slabiov1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	syncPeriod := time.Second * 30 // @TODO: DEV
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "6d243b47.k8slab.io",
		SyncPeriod:             &(syncPeriod), // @TODO: HERE
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	bannedNamespaces := []string{kubeSystemNamespace, "ingress-nginx", "image-backup"} // @TODO:
	dr := registry.NewDockerRegistry(backupRegistry, username, token)
	g := &controllers.GenericReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("generic"),
		Registry: dr,
	}
	if err = (&controllers.DeploymentReconciler{
		GenericReconciler: g,
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		Log:               ctrl.Log.WithName("controllers").WithName("deployment"),
	}).SetupWithManager(mgr, dr, bannedNamespaces); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Deployment")
		os.Exit(1)
	}

	if err = (&controllers.DaemonSetReconciler{
		GenericReconciler: g,
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		Log:               ctrl.Log.WithName("controllers").WithName("daemonSet"),
	}).SetupWithManager(mgr, dr, bannedNamespaces); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DaemonSet")
		os.Exit(1)
	}

	if err = (&controllers.ImageBackupReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Log:      ctrl.Log.WithName("controllers").WithName("imageBackup"),
		Registry: dr,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ImageBackup")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func init() {
	errBadConfig := errors.New("bad config")
	reg := os.Getenv("BACKUP_REPOSITORY")
	if reg == "" {
		setupLog.Error(errBadConfig, "empty backup registry")
		os.Exit(1)
	}
	backupRegistry = reg

	u := os.Getenv("BACKUP_REPOSITORY_USERNAME")
	if u == "" {
		setupLog.Error(errBadConfig, "empty registry username")
		os.Exit(1)
	}

	username = u

	pass := os.Getenv("BACKUP_REPOSITORY_PASSWORD")
	if pass == "" {
		setupLog.Error(errBadConfig, "empty backup password")
		os.Exit(1)
	}

	token = pass
}
