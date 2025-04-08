/*
Copyright 2024 Josh Michielsen.

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
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	goruntime "runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/jmickey/telegraf-sidecar-operator/internal/classdata"
	"github.com/jmickey/telegraf-sidecar-operator/internal/controller"
	"github.com/jmickey/telegraf-sidecar-operator/internal/injectorwebhook"
	"github.com/jmickey/telegraf-sidecar-operator/internal/version"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	defaultTelegrafSecretNamePrefix = "telegraf-config"
	defaultTelegrafImage            = "docker.io/library/telegraf:1.30-alpine"
	defaultTelegrafRequestsCPU      = "100m"
	defaultTelegrafRequestsMemory   = "100Mi"
	defaultTelegrafLimitsCPU        = "200m"
	defaultTelegrafLimitsMemory     = "300Mi"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var enableNativeSidecars bool

	var telegrafClassesDirectory string
	var telegrafDefaultClass string
	var telegrafEnableIntervalPlugin bool
	var telegrafSecretNamePrefix string
	var telegrafImage string
	var telegrafRequestsCPU string
	var telegrafRequestsMemory string
	var telegrafLimitsCPU string
	var telegrafLimitsMemory string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", false,
		"If set the metrics endpoint is served securely")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers.")
	flag.BoolVar(&enableNativeSidecars, "enable-native-sidecars", false,
		"If set, kubernetes v1.28 native sidecars will be enabled.")
	flag.StringVar(&telegrafClassesDirectory, "telegraf-classes-directory", "/etc/config/classes",
		"Path to the directory containing telegraf class files.")
	flag.StringVar(&telegrafDefaultClass, "telegraf-default-class", "default",
		"Default telegraf class to use.")
	flag.BoolVar(&telegrafEnableIntervalPlugin, "telegraf-enable-internal-plugin", false,
		"Enable the telegraf internal plugin in for all sidecar containers. "+
			"If disabled, can be overwritten using pod annotation.")
	flag.StringVar(&telegrafImage, "telegraf-image", defaultTelegrafImage,
		"Telegraf image to inject as a sidecar container.")
	flag.StringVar(&telegrafRequestsCPU, "telegraf-requests-cpu", defaultTelegrafRequestsCPU,
		"Default CPU requests for the telegraf sidecar.")
	flag.StringVar(&telegrafRequestsMemory, "telegraf-requests-memory", defaultTelegrafRequestsMemory,
		"Default memory requests for the telegraf sidecar.")
	flag.StringVar(&telegrafLimitsCPU, "telegraf-limits-cpu", defaultTelegrafLimitsCPU,
		"Default CPU limits for the telegraf sidecar.")
	flag.StringVar(&telegrafLimitsMemory, "telegraf-limits-memory", defaultTelegrafLimitsMemory,
		"Default memory limits for the telegraf sidecar.")
	flag.StringVar(&telegrafSecretNamePrefix, "telegraf-secret-name-prefix", defaultTelegrafSecretNamePrefix,
		"Set the telegraf configuration secret name prefix, defaults to 'telegraf-config'")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	tlsOpts := []func(*tls.Config){}
	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	if err := validateRequestsAndLimits([]string{
		telegrafRequestsCPU,
		telegrafRequestsMemory,
		telegrafLimitsCPU,
		telegrafLimitsMemory,
	}); err != nil {
		setupLog.Error(err, "failed to validate telegraf resource flag values")
		os.Exit(1)
	}

	classDataHandler, err := classdata.NewDirectoryHandler(telegrafClassesDirectory)
	if err != nil {
		setupLog.Error(err, "failed to initialize class data handler")
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress:   metricsAddr,
			SecureServing: secureMetrics,
			TLSOpts:       tlsOpts,
		},
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "b20b1aee.mickey.dev",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controller.PodReconciler{
		Client:               mgr.GetClient(),
		Scheme:               mgr.GetScheme(),
		Recorder:             mgr.GetEventRecorderFor("telegraf-sidecar-injector"),
		ClassDataHandler:     classDataHandler,
		DefaultClass:         telegrafDefaultClass,
		EnableInternalPlugin: telegrafEnableIntervalPlugin,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Pod")
		os.Exit(1)
	}

	admission := &injectorwebhook.SidecarInjector{
		SecretNamePrefix:     telegrafSecretNamePrefix,
		TelegrafImage:        telegrafImage,
		RequestsCPU:          telegrafRequestsCPU,
		RequestsMemory:       telegrafRequestsMemory,
		LimitsCPU:            telegrafLimitsCPU,
		LimitsMemory:         telegrafLimitsMemory,
		EnableNativeSidecars: enableNativeSidecars,
	}

	if err = admission.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create sidecar injector webhook", "component", "injectorwebhook")
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

	setupLog.Info("starting Telegraf Sidecar Operator",
		"manager-version", version.Version, "git-commit", version.GitCommit, "go-version", goruntime.Version())
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func validateRequestsAndLimits(resources []string) error {
	for _, val := range resources {
		if val != "" {
			_, err := resource.ParseQuantity(val)
			if err != nil {
				return fmt.Errorf("failed to parse resource value: %s, err: %w", val, err)
			}
		}
	}

	return nil
}
