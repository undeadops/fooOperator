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
	"flag"
	"os"
	"strconv"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/record"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	appsv1alpha1 "github.com/undeadops/fooOperator/api/v1alpha1"
	"github.com/undeadops/fooOperator/controllers/foo"
	"github.com/undeadops/fooOperator/controllers/foobar"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(appsv1alpha1.AddToScheme(scheme))
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

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "fcfe743e.undeadops.xyz",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	kubeConfig := ctrl.GetConfigOrDie()
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)

	eventBroadcaster := record.NewBroadcasterWithCorrelatorOptions(record.CorrelatorOptions{
		BurstSize: getEventBurstSize(),
		QPS:       1,
	})

	// Start events processing pipeline.
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	defer eventBroadcaster.Shutdown()

	if err = (&foobar.FooBarReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Log:      ctrl.Log.WithName("controllers").WithName("FooBar"),
		Recorder: eventBroadcaster.NewRecorder(mgr.GetScheme(), v1.EventSource{Component: "foobar-controller"}),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "FooBar")
		os.Exit(1)
	}
	if err = (&foo.FooReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Log:      ctrl.Log.WithName("controllers").WithName("Foo"),
		Recorder: eventBroadcaster.NewRecorder(mgr.GetScheme(), v1.EventSource{Component: "foo-controller"}),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Foo")
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

// getEventBurstSize returns the burst size of events that can be handled in the cluster
func getEventBurstSize() int {
	// EventBurstSizeEnvVar is the constant for env variable EVENT_BURST_SIZE
	// An empty value means the default burst size which is 150.
	var eventBurstSizeEnvVar = "EVENT_BURST_SIZE"
	var eventBurstSize = 150
	burstSize, found := os.LookupEnv(eventBurstSizeEnvVar)
	if found {
		eventBurstSizeInt, err := strconv.Atoi(burstSize)
		if err != nil {
			setupLog.Info("Invalid EVENT_BURST_SIZE value: using default 150")
		} else {
			eventBurstSize = eventBurstSizeInt
		}
	}
	return eventBurstSize
}
