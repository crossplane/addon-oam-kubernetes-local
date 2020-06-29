/*

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

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	oamapi "github.com/crossplane/oam-kubernetes-runtime/apis/core"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	oamcore "github.com/crossplane/oam-controllers/pkg/controller/core"
	"github.com/crossplane/oam-controllers/pkg/webhooks"
	// +kubebuilder:scaffold:imports
)

var scheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = oamapi.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var enableWebhook bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&enableWebhook, "enable-webhook", true, "Enable webhooks")
	flag.Parse()

	ctrl.SetLogger(zap.New(func(o *zap.Options) {
		o.Development = true
	}))

	oamLog := ctrl.Log.WithName("oam controller")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "oam-controller-runtime",
		CertDir:            webhooks.Cert_mount_path, // has to be the same as helm value
		Port:               9443,
	})
	if err != nil {
		oamLog.Error(err, "unable to create a controller manager")
		os.Exit(1)
	}

	if err = oamcore.Setup(mgr, logging.NewLogrLogger(oamLog)); err != nil {
		oamLog.Error(err, "unable to setup oam core controller")
		os.Exit(1)
	}

	if enableWebhook {
		if err = (&webhooks.ManualScalerTraitValidator{
			Log: ctrl.Log.WithName("validator webhook").WithName("ManualScalerTrait"),
		}).SetupWebhookWithManager(mgr); err != nil {
			oamLog.Error(err, "unable to create webhook", "webhook name", "ManualScalerTraitValidator")
			os.Exit(1)
		}
		if err = (&webhooks.ManualScalerTraitMutater{
			Log: ctrl.Log.WithName("mutate webhook").WithName("ManualScalerTrait"),
		}).SetupWebhookWithManager(mgr); err != nil {
			oamLog.Error(err, "unable to create webhook", "webhook name", "ManualScalerTraitMutater")
			os.Exit(1)
		}
	}
	// +kubebuilder:scaffold:builder

	oamLog.Info("starting the OAM controller manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		oamLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
