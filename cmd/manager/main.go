package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/Sparky12488/LB-ENT-K8S-Ingress/internal/controller"
	"github.com/Sparky12488/LB-ENT-K8S-Ingress/internal/lbapi"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	// Tell the app how to understand standard K8S onjects like Ingress
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

func main() {
	// Setup logging so we can see what's happening in the Pod Logs
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	// Load the dynamic config from Env Vars
	cfg := controller.LoadConfig()

	// Create the manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		fmt.Printf("unable to start manager: %v\n", err)
		os.Exit(1)
	}

	// Initialize the Loadbalancer API Client
	lbClient := &lbapi.Client{
		BaseURL: fmt.Sprintf("https://%s:9443", cfg.ApplianceIP),
		APIKey:  cfg.APIKey,
		LBUser:  cfg.LBUser,
		LBPass:  cfg.LBPass,
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}

	// Register our IngressReconciler with the Manager
	if err = (&controller.IngressReconciler{
		K8sClient: mgr.GetClient(),
		LBClient:  lbClient,
		Cfg:       cfg,
	}).SetupWithManager(mgr); err != nil {
		fmt.Printf("problem running manager: %v\n", err)
		os.Exit(1)
	}

	// Start the Manager (This blocks until the app is killed)
	fmt.Println("Starting Manager....")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		fmt.Printf("problem running manager: %v\n", err)
		os.Exit(1)
	}

}
