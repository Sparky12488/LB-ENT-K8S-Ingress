<p align="center">
  <img src="docs/images/Logo.png" alt="LB-ENT-K8S-Ingress Logo" width="200">
</p>

<h1 align="center">LB-ENT-K8S-Ingress</h1>

<p align="center">
  <a href="https://github.com/Sparky12488/LB-ENT-K8S-Ingress/actions/workflows/build.yml">
    <img src="https://github.com/Sparky12488/LB-ENT-K8S-Ingress/actions/workflows/build.yml/badge.svg" alt="Build Status">
  </a>
</p>

An unofficial, high-performance **Kubernetes Ingress Controller** for Loadbalancer.org Enterprise Appliances. 
This controller bridges Kubernetes service discovery with Layer 7 load balancing provided by the Loadbalancer.org appliance.

---

## 🌌 Features
* **Zero-Touch Automation:** Automatically syncs K8s Ingress resources to the appliance.

* **Dynamic RIP Management:** Adds and removes Real Server IPs (RIPs) as pods scale.

* **Smart Reconciliation:** Ensures the appliance matches cluster state.

* **Graceful Reloads:** Triggers HAProxy reloads only when configuration changes are detected.

---

## 🛠️ Prerequisites
* **Loadbalancer.org Appliance** (v8.13.X+) with API access enabled.

* **Kubernetes Cluster** (Kind, K3s, or Bare Metal).

* **Network Connectivity:** The K8s nodes must have a route to the Loadbalacner.org Appliance API and the Appliance must have a route to the Pod Network.

---

## 🚀 Installation

Example files can be found in the **deploy** folder 

### Configure the Cluster
Create the namespace and configuration. 
Update the LB_APPLIANCE_IP to your hardware's management IP.
Add a Comma separatedlist of IP's that can be used as the Frontend Service IP's 

**`cluster-config.yaml`**
```YAML
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
  LB_APPLIANCE_IP: "<LB Base IP>>" #This is th ebase IP for managing the Primary Loadbalancer 
  VIP_POOL: "<Comma separated list of IP>" # A Comma separated list of IP that the Loadbalancer can use for VIP's 
  LB_USER: "loadbalancer"
```

### Credentials
Create a secret for your API Key and Password.

**`secrets.yaml`**

```YAML
apiVersion: v1
kind: Secret
metadata:
  name: lb-appliance-auth
  namespace: ingress-controller
type: Opaque
stringData:
  API_KEY: "YOUR_API_KEY"
  LB_PASS: "YOUR_PASSWORD"
```

### 3. Deploy the Controller
Apply the RBAC and Deployment manifests:

```Bash
kubectl apply -f rbac.yaml
kubectl apply -f deployment.yaml
```
---

## 📡 High Availability Pod Routing
To ensure that the **Loadbalancer.org appliance** can always reach the Pod Network (x.x.0.0/16) even if a worker node fails, we implement a **Floating Gateway** using VRRP (Keepalived).

## 🏗️ Logic
Instead of pointing the appliance to a single node IP (which creates a Single Point of Failure), we point it to a **Virtual IP (VIP)** shared across all worker nodes.

1. **The Workers:** Run keepalived to maintain the Floating IP.

2. **The Appliance:** Uses Floating IP as the static route gateway for all container traffic.

3. **Failover:** If the primary worker node goes down, the Floating IP instantly moves to a backup worker, keeping the traffic path alive.

## 🛠️ Configuration Steps

1. Worker Node Setup
Install Keepalived on all worker nodes:

```Bash
sudo apt-get install -y keepalived
```

Create `/etc/keepalived/keepalived.conf`:

```Bash
vrrp_instance POD_GATEWAY {
    state BACKUP
    interface eth0           # Match your physical interface
    virtual_router_id 60
    priority 100             # Set higher (e.g., 110) on your preferred node
    advert_int 1
    virtual_ipaddress {
        x.x.x.x/24      # The Floating Gateway IP
    }
}
```
2. Loadbalancer.org Appliance Setup
Add the new Floating Gateway to the **Routing** config :

Local Configuration > Routing > Static Routes

**subnet:** Pod Network Address (x.x.x.x/16)


**Via Gateway:** Floating Gateway IP

---

## 🧪 Usage

To route traffic, create an Ingress with the `lb-org` class:

```YAML
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
```
---

## ⚖️ Disclaimer

This project is an unofficial 3rd-party integration. It is not affiliated with, maintained by, or supported by Loadbalancer.org Ltd. Use at your own risk.