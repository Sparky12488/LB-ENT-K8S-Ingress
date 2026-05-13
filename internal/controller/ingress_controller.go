package controller

import (
	"context"
	"fmt"

	"github.com/Sparky12488/LB-ENT-K8S-Ingress/internal/lbapi"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type IngressReconciler struct {
	K8sClient client.Client
	LBClient  *lbapi.Client
	Cfg       *Config
}

func (r *IngressReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	fmt.Printf("\n--- Reconcile Triggered for: %s ---\n", req.Name)

	ingress := &networkingv1.Ingress{}
	err := r.K8sClient.Get(ctx, req.NamespacedName, ingress)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	fmt.Println("Step 1: Ingress object fetched")

	vipName := fmt.Sprintf("k8s_%s", ingress.Name)
	finalizerName := "lb.org/finalizer"

	// Track if we need to reload HAProxy at the end
	needsReload := false

	// 1. HANDLE DELETION
	if !ingress.ObjectMeta.DeletionTimestamp.IsZero() {
		fmt.Println("Step 2a: Handling Deletion")
		if containsString(ingress.GetFinalizers(), finalizerName) {
			fmt.Printf("Cleaning up Appliance for VIP: %s\n", vipName)
			r.LBClient.DeleteVIP(vipName)
			r.LBClient.ReloadConfig() // Hard reload on delete is usually fine

			ingress.SetFinalizers(removeString(ingress.GetFinalizers(), finalizerName))
			r.K8sClient.Update(ctx, ingress)
		}
		return reconcile.Result{}, nil
	}

	// 2. REGISTER FINALIZER
	if !containsString(ingress.GetFinalizers(), finalizerName) {
		fmt.Println("Step 2b: Registering Finalizer")
		ingress.SetFinalizers(append(ingress.GetFinalizers(), finalizerName))
		if err := r.K8sClient.Update(ctx, ingress); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// 3. GET CONFIG
	if len(r.Cfg.VIPPool) == 0 || r.Cfg.VIPPool[0] == "" {
		return reconcile.Result{}, fmt.Errorf("VIP pool is empty")
	}
	targetVIP := r.Cfg.VIPPool[0]

	// 4. RECONCILE RULES
	for _, rule := range ingress.Spec.Rules {
		for _, path := range rule.HTTP.Paths {
			// Ensure VIP exists
			vipAction := map[string]string{
				"action": "add-vip", "layer": "7", "vip": vipName,
				"ip": targetVIP, "ports": "80", "mode": "http", "forwarding": "proxy",
			}
			// We could check if VIP exists, but add-vip usually just returns "exists"
			r.LBClient.SendAction(vipAction)

			// Get current state from Appliance (using our new dumpconfig parser)
			existingIPs, _ := r.LBClient.ListVirtual(vipName)

			// Get desired state from K8S
			serviceName := path.Backend.Service.Name
			endpoints := &corev1.Endpoints{}
			err = r.K8sClient.Get(ctx, client.ObjectKey{Namespace: ingress.Namespace, Name: serviceName}, endpoints)
			if err != nil {
				continue
			}

			var desiredIPs []string
			for _, subset := range endpoints.Subsets {
				for _, addr := range subset.Addresses {
					desiredIPs = append(desiredIPs, addr.IP)
				}
			}

			// Sync RIPs: Delete stale
			for _, existing := range existingIPs {
				if !containsString(desiredIPs, existing) {
					fmt.Printf("Removing stale RIP: %s\n", existing)
					r.LBClient.DeleteRIP(vipName, fmt.Sprintf("rip_%s", existing))
					needsReload = true
				}
			}

			// Sync RIPs: Add new
			for _, desired := range desiredIPs {
				if !containsString(existingIPs, desired) {
					fmt.Printf("Adding new RIP: %s\n", desired)
					ripAction := map[string]string{
						"action": "add-rip", "vip": vipName,
						"rip": fmt.Sprintf("rip_%s", desired), "ip": desired, "weight": "100",
					}
					r.LBClient.SendAction(ripAction)
					needsReload = true
				}
			}
		}
	}

	// 5. COMMIT (Only if changes were made)
	if needsReload {
		fmt.Println("Step 7: Changes detected. Triggering HAProxy Reload.")
		r.LBClient.ReloadConfig()
	}

	return reconcile.Result{}, nil
}

func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.Ingress{}).
		Watches(
			&corev1.Endpoints{},
			handler.EnqueueRequestsFromMapFunc(r.findIngressForEndpoints), // Add this logic
		).
		Complete(r)
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

func (r *IngressReconciler) findIngressForEndpoints(ctx context.Context, obj client.Object) []reconcile.Request {
	endpoints, ok := obj.(*corev1.Endpoints)
	if !ok {
		return nil
	}

	var ingresses networkingv1.IngressList
	if err := r.K8sClient.List(ctx, &ingresses, client.InNamespace(endpoints.Namespace)); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, ing := range ingresses.Items {
		for _, rule := range ing.Spec.Rules {
			for _, path := range rule.HTTP.Paths {
				if path.Backend.Service.Name == endpoints.Name {
					requests = append(requests, reconcile.Request{
						NamespacedName: client.ObjectKey{
							Namespace: ing.Namespace,
							Name:      ing.Name,
						},
					})
				}
			}
		}
	}
	return requests
}
