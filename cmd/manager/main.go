package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	zapctrl "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/jwks-operator/jwks-operator/api/v1alpha1"
	"github.com/jwks-operator/jwks-operator/pkg/config"
	"github.com/jwks-operator/jwks-operator/pkg/controller"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var configPath string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&configPath, "config", "config.yaml", "Path to configuration file")

	opts := zapctrl.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zapctrl.New(zapctrl.UseFlagOptions(&opts)))

	// Get namespace from Kubernetes environment
	// This is typically set via downward API in Pod spec
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		// Fallback: try to read from service account namespace file
		if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
			namespace = string(data)
			namespace = strings.TrimSpace(namespace)
			setupLog.Info("Namespace read from service account file", "namespace", namespace)
		} else {
			setupLog.Info("Could not read namespace from service account file", "error", err)
		}
	} else {
		setupLog.Info("Namespace from POD_NAMESPACE environment", "namespace", namespace)
	}
	if namespace == "" {
		setupLog.Error(fmt.Errorf("namespace not found"), "POD_NAMESPACE environment variable or service account namespace file is required")
		os.Exit(1)
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); err != nil {
		setupLog.Info("Config file not found, using defaults", "path", configPath, "error", err)
	} else {
		setupLog.Info("Config file found", "path", configPath)
	}

	// Load configuration
	cfg, err := config.Load(configPath, namespace)
	if err != nil {
		setupLog.Error(err, "Failed to load configuration", "configPath", configPath, "namespace", namespace)
		os.Exit(1)
	}

	setupLog.Info("Loaded configuration", "namespace", cfg.Namespace)

	// Create logger
	logger := createLogger(cfg)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "jwks-operator.example.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Create controller
	jwksReconciler := controller.NewJWKSReconciler(mgr.GetClient(), mgr.GetScheme(), cfg, logger)

	if err = jwksReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "JWKS")
		os.Exit(1)
	}

	// Setup health checks
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

// createLogger creates a zap logger based on configuration
func createLogger(cfg *config.Config) *zap.Logger {
	var zapConfig zap.Config

	if cfg.Logging.Format == "json" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
	}

	// Set log level
	switch cfg.Logging.Level {
	case "debug":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	logger, err := zapConfig.Build()
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	return logger
}
