// Copyright Contributors to the Open Cluster Management project

package fakepolicy

import (
	"context"
	"flag"
	"os"

	"github.com/go-logr/zapr"
	"github.com/stolostron/go-log-utils/zaputil"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	fakev1beta1 "open-cluster-management.io/governance-policy-nucleus/test/fakepolicy/api/v1beta1"
	"open-cluster-management.io/governance-policy-nucleus/test/fakepolicy/controllers"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
	utilruntime.Must(fakev1beta1.AddToScheme(scheme))
}

//nolint:unused
func main() {
	if err := Run(context.Background(), nil); err != nil {
		os.Exit(1)
	}
}

func Run(parentCtx context.Context, cfg *rest.Config) error {
	zflags := zaputil.NewFlagConfig()
	zflags.Bind(flag.CommandLine)
	klog.InitFlags(flag.CommandLine)

	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string

	flag.StringVar(&metricsAddr, "metrics-bind-address", "0",
		"The address the metric endpoint binds to. Disabled by default, but conventionally :8080")
	flag.StringVar(&probeAddr, "health-probe-bind-address", "0",
		"The address the probe endpoint binds to. Disabled by default, but conventionally :8081")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	flag.Parse()

	ctrlZap, err := zflags.BuildForCtrl()
	if err != nil {
		return err
	}

	ctrl.SetLogger(zapr.NewLogger(ctrlZap))
	setupLog := ctrl.Log.WithName("setup")

	klogZap, err := zaputil.BuildForKlog(zflags.GetConfig(), flag.CommandLine)
	if err != nil {
		return err
	}

	klog.SetLogger(zapr.NewLogger(klogZap).WithName("klog"))

	if cfg == nil {
		var err error

		cfg, err = ctrl.GetConfig()
		if err != nil {
			setupLog.Error(err, "unable to get Kubernetes config")

			return err
		}
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		Metrics:                server.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "8b5e65ab.open-cluster-management.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")

		return err
	}

	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		setupLog.Error(err, "unable to create dynamic client")

		return err
	}

	if err = (&controllers.FakePolicyReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		DynamicClient: dynClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "FakePolicy")

		return err
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")

		return err
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")

		return err
	}

	setupLog.Info("starting manager")

	managerCtx, cancel := context.WithCancel(parentCtx)

	go func() {
		// It would be nicer if this could take a parent context,
		// but this will work to cancel the manager on those signals.
		<-ctrl.SetupSignalHandler().Done()
		cancel()
	}()

	if err := mgr.Start(managerCtx); err != nil {
		setupLog.Error(err, "problem running manager")

		return err
	}

	return nil
}
