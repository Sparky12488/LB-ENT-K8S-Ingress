# ![Logo](docs/images/logo.svg) LB-ENT-K8S-Ingress

##LB-ENT-K8S-Ingress

An unofficial, high-performance Kubernetes Ingress Controller for Loadbalancer.org Enterprise Appliances. This controller bridges Kubernetes service discovery with hardware-based Layer 7 load balancing.

## 🌌 Features
Zero-Touch Automation: Automatically syncs K8s Ingress resources to the appliance.

Dynamic RIP Management: Adds and removes Real Server IPs (RIPs) as pods scale.

Smart Reconciliation: Uses dumpconfig to ensure the appliance matches cluster state.

Graceful Reloads: Triggers HAProxy reloads only when configuration changes are detected.

## 🛠️ Prerequisites
Loadbalancer.org Appliance (v8.x+) with API access enabled.

Kubernetes Cluster (Kind, K3s, or Bare Metal).

Network Connectivity: The K8s nodes must have a route to the Appliance API (9443) and the Appliance must have a route to the Pod Network.

## 🚀 Installation
1. Configure the Cluster
Create the namespace and configuration. Update the LB_APPLIANCE_IP to your hardware's management IP.

cluster-config.yaml

YAML
apiVersion: v1
kind: Namespace
metadata:
  name: ingress-controller
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: lb-controller-config
  namespace: ingress-controller
data:
  LB_APPLIANCE_IP: "10.30.80.70"
  VIP_POOL: "10.30.80.71" # The IP the LB will listen on
  LB_USER: "loadbalancer"


2. Add Credentials
Create a secret for your API Key and Password. Do not commit this file to Git.

secrets.yaml

YAML
apiVersion: v1
kind: Secret
metadata:
  name: lb-appliance-auth
  namespace: ingress-controller
type: Opaque
stringData:
  API_KEY: "YOUR_ENCODED_API_KEY"
  LB_PASS: "YOUR_PASSWORD"


3. Deploy the Controller
Apply the RBAC and Deployment manifests:

Bash
kubectl apply -f rbac.yaml
kubectl apply -f deployment.yaml


## 📡 Networking Note
To allow the Load Balancer to health-check the pods, ensure you have a static route on the appliance:

Destination: 10.244.0.0/16 (Your Pod CIDR)

Gateway: [Your K8s Node IP]

On the K8s Host, enable IP forwarding:

Bash
sudo sysctl -w net.ipv4.ip_forward=1
sudo iptables -t nat -A POSTROUTING -j MASQUERADE


## 🧪 Usage
To route traffic, create an Ingress with the lb-org class:

YAML
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: demo-ingress
  namespace: ingress-controller
spec:
  ingressClassName: lb-org
  rules:
  - http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: demo-service
            port:
              number: 80


## ⚖️ Disclaimer
This project is an unofficial 3rd-party integration. It is not affiliated with, maintained by, or supported by Loadbalancer.org Ltd. Use at your own risk.