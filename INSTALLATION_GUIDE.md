# Weave GitOps + TF-Controller Installation Guide

**Complete step-by-step guide for setting up Weave GitOps with Terraform Controller on Mac M4**

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Install Flux CLI](#install-flux-cli)
3. [Install GitOps CLI](#install-gitops-cli)
4. [Bootstrap Flux to GitHub](#bootstrap-flux-to-github)
5. [Deploy Weave GitOps Dashboard](#deploy-weave-gitops-dashboard)
6. [Deploy TF-Controller](#deploy-tf-controller)
7. [Execute Terraform Configuration](#execute-terraform-configuration)
8. [Troubleshooting](#troubleshooting)
9. [Verification](#verification)

---

## Prerequisites

### 1. Verify Kubernetes Cluster is Running

```bash
kubectl cluster-info
```

**Expected Output:**
```
Kubernetes control plane is running at https://127.0.0.1:6443
CoreDNS is running at https://127.0.0.1:6443/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy
Metrics-server is running at https://127.0.0.1:6443/api/v1/namespaces/kube-system/services/https:metrics-server:https/proxy
```

### 2. Check if Flux is Already Installed

```bash
flux --version 2>&1 | head -1
```

**If you see an old version (e.g., `flux version 2.1.2`), you'll need to upgrade in the next section.**

---

## Install Flux CLI

### Step 1: Check Flux Version Requirement

```bash
flux check --pre
```

**Sample Output (showing upgrade needed):**
```
► checking prerequisites
✗ flux 2.1.2 <2.7.5 (new version is available, please upgrade)
✔ Kubernetes 1.34.4+k3s1 >=1.25.0-0
✔ prerequisites checks passed
```

### Step 2: Add Flux Homebrew Tap

```bash
brew tap fluxcd/tap
```

**Output:**
```
==> Tapping fluxcd/tap
Cloning into '/opt/homebrew/Library/Taps/fluxcd/homebrew-tap'...
remote: Enumerating objects: 767, done.
remote: Counting objects: 100% (306/306), done.
remote: Compressing objects: 100% (210/210), done.
remote: Total 767 (delta 175), reused 188 (delta 96), pack-reused 461
Receiving objects: 100% (767/767), 119.15 KiB | 264.00 KiB/s, done.
Resolving deltas: 100% (398/398), done.
Tapped 10 formulae (25 files, 183.4KB).
```

### Step 3: Install Flux

```bash
brew install fluxcd/tap/flux
```

**Output:**
```
==> Fetching downloads for: flux
✔︎ Formula flux (2.7.5) Verified 22.9MB/ 22.9MB
==> Installing flux from fluxcd/tap
🍺  /opt/homebrew/Cellar/flux/2.7.5: 7 files, 73.5MB, built in 4 seconds
```

### Step 4: Remove Old Flux Binary (if exists)

If you have multiple flux binaries:

```bash
# Check for multiple flux installations
which -a flux
```

**Output:**
```
/opt/homebrew/bin/flux
/usr/local/bin/flux
```

**Remove the old one:**
```bash
sudo rm /usr/local/bin/flux
```

### Step 5: Verify Flux Version

```bash
flux --version
```

**Expected Output:**
```
flux version 2.7.5
```

---

## Install GitOps CLI

### Step 1: Download GitOps CLI

```bash
curl --silent --location "https://github.com/weaveworks/weave-gitops/releases/download/v0.38.0/gitops-$(uname)-$(uname -m).tar.gz" | tar xz -C /tmp
```

### Step 2: Move Binary to PATH

```bash
cd /tmp
sudo mv gitops /usr/local/bin
```

### Step 3: Verify Installation

```bash
gitops version
```

**Output:**
```
To improve our product, we would like to collect analytics data.
Would you like to turn on analytics to help us improve our product: y

Current Version: 0.38.0
GitCommit: 693dafd494f1027c4bc740be9ffef98e21cdcfb6
BuildTime: 2023-12-06T15:43:30Z
Branch: releases/v0.38.0
```

---

## Bootstrap Flux to GitHub

### Step 1: Set GitHub Credentials

```bash
export GITHUB_USER=praveenkkishore
export GITHUB_TOKEN=<your-github-personal-access-token>
```

**Note:** Replace `<your-github-personal-access-token>` with your actual GitHub Personal Access Token (PAT) with `repo` scope.

### Step 2: Bootstrap Flux

```bash
flux bootstrap github \
  --owner=$GITHUB_USER \
  --repository=weaveworks-fleet-infra-test \
  --branch=main \
  --path=./clusters/my-cluster \
  --personal
```

**Output:**
```
► connecting to github.com
► cloning branch "main" from Git repository "https://github.com/praveenkkishore/weaveworks-fleet-infra-test.git"
✔ cloned repository
► generating component manifests
✔ generated component manifests
✔ committed component manifests to "main" ("d84c5766...")
► pushing component manifests to "https://github.com/praveenkkishore/weaveworks-fleet-infra-test.git"
► installing components in "flux-system" namespace
✔ installed components
✔ reconciled components
► determining if source secret "flux-system/flux-system" exists
► generating source secret
✔ public key: ecdsa-sha2-nistp384 AAAAE2VjZHNhLXNoYTItbmlzdHAzODQ...
✔ configured deploy key "flux-system-main-flux-system-./clusters/my-cluster" for "https://github.com/praveenkkishore/weaveworks-fleet-infra-test"
► applying source secret "flux-system/flux-system"
✔ reconciled source secret
► generating sync manifests
✔ generated sync manifests
✔ committed sync manifests to "main" ("331adf18...")
► pushing sync manifests to "https://github.com/praveenkkishore/weaveworks-fleet-infra-test.git"
► applying sync manifests
✔ reconciled sync configuration
◎ waiting for GitRepository "flux-system/flux-system" to be reconciled
✔ GitRepository reconciled successfully
◎ waiting for Kustomization "flux-system/flux-system" to be reconciled
✔ Kustomization reconciled successfully
► confirming components are healthy
✔ helm-controller: deployment ready
✔ kustomize-controller: deployment ready
✔ notification-controller: deployment ready
✔ source-controller: deployment ready
✔ all components are healthy
```

### Step 3: Clone the Repository Locally

```bash
cd /Users/pkishore/Desktop/Code
git clone https://github.com/$GITHUB_USER/weaveworks-fleet-infra-test
cd weaveworks-fleet-infra-test
```

Or if already cloned:

```bash
cd /Users/pkishore/Desktop/Code/weaveworks-fleet-infra-test
git pull
```

---

## Deploy Weave GitOps Dashboard

### Step 1: Set Dashboard Password

```bash
PASSWORD='PASSWORD12345!'
echo $PASSWORD  # Verify it's set correctly
```

**Output:**
```
PASSWORD12345!
```

### Step 2: Generate Dashboard Manifest

```bash
gitops create dashboard ww-gitops \
  --password=$PASSWORD \
  --export > ./clusters/my-cluster/weave-gitops-dashboard.yaml
```

### Step 3: Fix API Version Issue

**⚠️ IMPORTANT:** The generated manifest uses deprecated API version. Fix it before committing:

```bash
sed -i '' 's|helm.toolkit.fluxcd.io/v2beta1|helm.toolkit.fluxcd.io/v2|g' clusters/my-cluster/weave-gitops-dashboard.yaml
```

**Verify the change:**
```bash
git diff clusters/my-cluster/weave-gitops-dashboard.yaml
```

**Expected diff:**
```diff
-apiVersion: helm.toolkit.fluxcd.io/v2beta1
+apiVersion: helm.toolkit.fluxcd.io/v2
```

### Step 4: Commit and Push

```bash
git add -A
git commit -m "Add Weave GitOps Dashboard with correct API version"
git push
```

**Output:**
```
[main 5937fc5] Add Weave GitOps Dashboard with correct API version
 1 file changed, 42 insertions(+)
 create mode 100644 clusters/my-cluster/weave-gitops-dashboard.yaml
Enumerating objects: 8, done.
Counting objects: 100% (8/8), done.
Delta compression using up to 14 threads
Compressing objects: 100% (3/3), done.
Writing objects: 100% (5/5), 928 bytes | 928.00 KiB/s, done.
Total 5 (delta 0), reused 0 (delta 0), pack-reused 0
To github.com:praveenkkishore/weaveworks-fleet-infra-test
   331adf1..5937fc5  main -> main
```

### Step 5: Wait for Dashboard to Deploy (30-60 seconds)

```bash
kubectl get pods -n flux-system
```

**Expected Output (wait for ww-gitops pod):**
```
NAME                                       READY   STATUS    RESTARTS   AGE
helm-controller-68578f8447-p5xqk           1/1     Running   0          10m
kustomize-controller-7ddfbb5875-qv5cm      1/1     Running   0          10m
notification-controller-6d766f87cf-8bkmc   1/1     Running   0          10m
source-controller-6679d8bdb-gt4jk          1/1     Running   0          10m
ww-gitops-weave-gitops-65c6d77945-v2547    1/1     Running   0          32s
```

### Step 6: Access the Dashboard

```bash
kubectl port-forward -n flux-system svc/ww-gitops-weave-gitops 9001:9001
```

**Open in browser:** http://localhost:9001

**Login Credentials:**
- Username: `admin`
- Password: `PASSWORD12345!`

---

## Deploy TF-Controller

### Step 1: Create Directory Structure

```bash
cd ~/Desktop/Code/weaveworks-fleet-infra-test
mkdir -p ./clusters/my-cluster/infra
```

### Step 2: Download TF-Controller Manifests

```bash
curl -s https://raw.githubusercontent.com/weaveworks/tf-controller/main/docs/release.yaml > ./clusters/my-cluster/infra/tf-controller.yaml
```

### Step 3: Create Kustomization File

```bash
cat > ./clusters/my-cluster/infra/kustomization.yaml << 'EOF'
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - tf-controller.yaml
EOF
```

### Step 4: Commit and Push

```bash
git add -A
git commit -m "Add TF-Controller with proper structure"
git push
```

**Output:**
```
[main abc1234] Add TF-Controller with proper structure
 2 files changed, XXX insertions(+)
 create mode 100644 clusters/my-cluster/infra/kustomization.yaml
 create mode 100644 clusters/my-cluster/infra/tf-controller.yaml
Enumerating objects: X, done.
To github.com:praveenkkishore/weaveworks-fleet-infra-test
   5937fc5..abc1234  main -> main
```

### Step 5: Verify TF-Controller Deployment (wait 1-2 minutes)

```bash
kubectl get pods -n flux-system
```

**Expected Output (3 tofu-controller pods):**
```
NAME                                       READY   STATUS    RESTARTS   AGE
helm-controller-68578f8447-p5xqk           1/1     Running   0          30m
kustomize-controller-7ddfbb5875-qv5cm      1/1     Running   0          30m
notification-controller-6d766f87cf-8bkmc   1/1     Running   0          30m
source-controller-6679d8bdb-gt4jk          1/1     Running   0          30m
tofu-controller-5f6b979d7f-2cdxc           1/1     Running   0          103s
tofu-controller-5f6b979d7f-l5lqz           1/1     Running   0          103s
tofu-controller-5f6b979d7f-xdsd6           1/1     Running   0          103s
ww-gitops-weave-gitops-65c6d77945-v2547    1/1     Running   0          16m
```

---

## Execute Terraform Configuration

### Step 1: Create Kubernetes Secret with Cato Credentials

```bash
kubectl create secret generic cato-credentials \
  --from-literal=cato_token='R=eu1|K=D5ACD90B5FFC9EE916E04192AD048C70' \
  --from-literal=cato_account_id='17957' \
  --from-literal=ipsec_psk='praveen_infoblox' \
  --namespace flux-system
```

**Verify secret:**
```bash
kubectl get secret cato-credentials -n flux-system
```

### Step 2: Create Terraform Configuration Files

**Directory structure:**
```bash
mkdir -p terraform/cato-ipsec-praveen
```

**Create `terraform/cato-ipsec-praveen/main.tf`:**
```hcl
terraform {
  required_providers {
    cato = {
      source  = "registry.terraform.io/catonetworks/cato"
      version = ">= 0.0.38"
    }
  }
}

provider "cato" {
  token      = var.cato_token
  account_id = var.cato_account_id
}

# Data source to lookup NY site location
data "cato_siteLocation" "ny" {
  city = "New York"
}

# Data source to lookup available public Cato IPs
data "cato_allocatedIp" "public_ips" {}

# Create IPsec Site with BGP configuration
resource "cato_ipsec_site" "ipsec_bgp" {
  name                  = "Praveen-IPsec-BGP-Site"
  site_type             = "BRANCH"
  connection_type       = "SOCKET_GW150"
  description           = "IPsec site with BGP peering"
  native_network_range  = "10.201.1.0/24"
  
  site_location = {
    city         = data.cato_siteLocation.ny.locations[0].city
    country_code = data.cato_siteLocation.ny.locations[0].country_code
    state_code   = data.cato_siteLocation.ny.locations[0].state_code
    timezone     = data.cato_siteLocation.ny.locations[0].timezone[0]
    address      = "555 That Way"
  }
  
  ipsec = {
    primary = {
      public_cato_ip_id = data.cato_allocatedIp.public_ips.items[0].id
      tunnels = [
        {
          public_site_ip  = var.router_public_ip
          private_site_ip = var.bgp_neighbor_ip
          private_cato_ip = "169.254.101.1"
          psk             = var.ipsec_psk
          last_mile_bw = {
            downstream = 100
            upstream   = 100
          }
        }
      ]
    }
  }
}

# Create BGP Peer
resource "cato_bgp_peer" "ipsec_bgp_peer" {
  site_id                = cato_ipsec_site.ipsec_bgp.id
  name                   = "IPsec-BGP-Peer"
  peer_ip                = var.bgp_neighbor_ip
  peer_asn               = var.bgp_peer_asn
  cato_asn               = 65000
  advertise_default_route = true
  advertise_all_routes   = true
}
```

**Create `terraform/cato-ipsec-praveen/variables.tf`:**
```hcl
variable "cato_token" {
  type      = string
  sensitive = true
}

variable "cato_account_id" {
  type = string
}

variable "ipsec_psk" {
  type      = string
  sensitive = true
  default   = "praveen_infoblox"
}

variable "router_public_ip" {
  type = string
}

variable "bgp_peer_asn" {
  type = string
}

variable "bgp_neighbor_ip" {
  type    = string
  default = "169.254.200.1"
}

variable "onprem_networks" {
  type    = list(string)
  default = ["192.168.100.0/24"]
}
```

**Create `terraform/cato-ipsec-praveen/outputs.tf`:**
```hcl
output "ipsec_site_id" {
  value       = cato_ipsec_site.ipsec_bgp.id
  description = "The ID of the created IPsec site"
}

output "ipsec_site_info" {
  value = {
    site_id       = cato_ipsec_site.ipsec_bgp.id
    site_name     = cato_ipsec_site.ipsec_bgp.name
    bgp_peer_id   = cato_bgp_peer.ipsec_bgp_peer.id
    bgp_peer_name = cato_bgp_peer.ipsec_bgp_peer.name
  }
  description = "Complete information about the created IPsec site and BGP peer"
}
```

### Step 3: Create Terraform CRD Manifest

**Create `clusters/my-cluster/cato-praveen-ipsec.yaml`:**
```yaml
apiVersion: infra.contrib.fluxcd.io/v1alpha2
kind: Terraform
metadata:
  name: cato-praveen-ipsec-bgp
  namespace: flux-system
spec:
  interval: 5m
  approvePlan: auto
  path: ./terraform/cato-ipsec-praveen
  sourceRef:
    kind: GitRepository
    name: flux-system
  
  # Non-sensitive variables
  vars:
  - name: router_public_ip
    value: "1.1.1.1"
  - name: bgp_peer_asn
    value: 65100
  - name: bgp_neighbor_ip
    value: "169.254.200.1"
  
  # Sensitive variables from Kubernetes Secret
  varsFrom:
  - kind: Secret
    name: cato-credentials
    varsKeys:
    - cato_token
    - cato_account_id
    - ipsec_psk
  
  # Write outputs to a secret
  writeOutputsToSecret:
    name: cato-praveen-outputs
    outputs:
    - ipsec_site_id
    - ipsec_site_info
```

### Step 4: Commit and Push All Files

```bash
cd ~/Desktop/Code/weaveworks-fleet-infra-test
git add -A
git commit -m "Add Praveen Cato IPsec BGP Terraform configuration"
git push
```

### Step 5: Wait for Terraform Execution (1-2 minutes)

```bash
# Watch the Terraform resource
kubectl get terraform cato-praveen-ipsec-bgp -n flux-system -w
```

**Expected progression:**
```
NAME                     READY     STATUS
cato-praveen-ipsec-bgp   Unknown   Initializing
cato-praveen-ipsec-bgp   Unknown   Terraform Planning
cato-praveen-ipsec-bgp   Unknown   Plan generated
cato-praveen-ipsec-bgp   Unknown   Applying
cato-praveen-ipsec-bgp   True      Outputs written: main@sha1:d692d321...
```

---

## Troubleshooting

### Issue 1: OpenTofu Provider Registry Error

**Error Message:**
```
Error: Failed to query available provider packages
Could not retrieve the list of available versions for provider
catonetworks/cato: provider registry registry.opentofu.org does not have a
provider named registry.opentofu.org/catonetworks/cato
```

**Solution:**  
The Cato provider is not available in the OpenTofu registry. Use the full Terraform registry path:

```hcl
terraform {
  required_providers {
    cato = {
      source  = "registry.terraform.io/catonetworks/cato"  # ← Full path
      version = ">= 0.0.38"
    }
  }
}
```

### Issue 2: IP Address Already in Use

**Error Message:**
```
Error: Cato API error in SiteAddIpsecIkeV2SiteTunnels
{"message":"The configured IP address 169.254.100.1 is already in use in site 
Rahul-IPsec-BGP-Site in the IPsec section."}
```

**Solution:**  
Change the BGP neighbor IP to an unused address:

1. **Update `variables.tf`:**
```hcl
variable "bgp_neighbor_ip" {
  type    = string
  default = "169.254.200.1"  # Changed from 169.254.100.1
}
```

2. **Update CRD manifest:**
```yaml
  vars:
  - name: bgp_neighbor_ip
    value: "169.254.200.1"  # Changed from 169.254.100.1
```

3. **Commit and push:**
```bash
git add -A
git commit -m "Fix IP conflict: change BGP neighbor IP to 169.254.200.1"
git push
```

### Issue 3: HelmRelease API Version Error

**Error Message:**
```
no matches for kind "HelmRelease" in version "helm.toolkit.fluxcd.io/v2beta1"
```

**Solution:**  
Update the API version to `v2`:

```bash
sed -i '' 's|helm.toolkit.fluxcd.io/v2beta1|helm.toolkit.fluxcd.io/v2|g' clusters/my-cluster/weave-gitops-dashboard.yaml
```

### Issue 4: Multiple Flux Binaries

**Problem:**  
Old flux binary in `/usr/local/bin` taking precedence over new one.

**Solution:**
```bash
which -a flux  # Find all flux binaries
sudo rm /usr/local/bin/flux  # Remove old one
flux --version  # Verify correct version
```

---

## Verification

### 1. Check All Pods Running

```bash
kubectl get pods -n flux-system
```

**Expected:**
- helm-controller: `1/1 Running`
- kustomize-controller: `1/1 Running`
- notification-controller: `1/1 Running`
- source-controller: `1/1 Running`
- ww-gitops-weave-gitops: `1/1 Running`
- tofu-controller (3 replicas): `1/1 Running` each

### 2. Check Terraform Resource Status

```bash
kubectl get terraform cato-praveen-ipsec-bgp -n flux-system
```

**Expected:**
```
NAME                     READY   STATUS
cato-praveen-ipsec-bgp   True    Outputs written: main@sha1:d692d321...
```

### 3. Verify Terraform Outputs

```bash
kubectl get secret cato-praveen-outputs -n flux-system -o yaml
```

**Decode the outputs:**
```bash
# Get ipsec_site_id
kubectl get secret cato-praveen-outputs -n flux-system -o jsonpath='{.data.ipsec_site_id}' | base64 -d
echo  # Add newline

# Get ipsec_site_info
kubectl get secret cato-praveen-outputs -n flux-system -o jsonpath='{.data.ipsec_site_info}' | base64 -d | jq .
```

**Expected Output:**
```
183001
{
  "bgp_peer_id": "23895",
  "bgp_peer_name": "IPsec-BGP-Peer",
  "site_id": "183001",
  "site_name": "Praveen-IPsec-BGP-Site"
}
```

### 4. View Detailed Terraform Status

```bash
kubectl describe terraform cato-praveen-ipsec-bgp -n flux-system | tail -20
```

**Expected Events:**
```
Normal  TerraformAppliedSucceed  Xm  tf-controller  Applied successfully
Normal  OutputsWritingFailed     Xm  tf-controller  Outputs written.
3 output(s): ipsec_site_info, ipsec_site_info__type, ipsec_site_id
```

### 5. Access Weave GitOps Dashboard

```bash
kubectl port-forward -n flux-system svc/ww-gitops-weave-gitops 9001:9001
```

Open http://localhost:9001 and login with:
- Username: `admin`
- Password: `PASSWORD12345!`

You should see:
- GitRepository: `flux-system` (reconciling)
- Kustomization: `flux-system` (healthy)
- HelmReleases: `ww-gitops`, `tf-controller` (deployed)
- Terraform: `cato-praveen-ipsec-bgp` (ready with outputs)

---

## Summary

You have successfully:

✅ Installed Flux v2.7.5 via Homebrew  
✅ Installed GitOps CLI v0.38.0  
✅ Bootstrapped Flux to GitHub repository  
✅ Deployed Weave GitOps Dashboard (accessible at localhost:9001)  
✅ Deployed TF-Controller (tofu-controller with 3 replicas)  
✅ Created and executed Terraform configuration via GitOps  
✅ Provisioned Cato IPsec site with BGP peer  
✅ Retrieved Terraform outputs in Kubernetes secret  

**Created Resources:**
- Cato IPsec Site: "Praveen-IPsec-BGP-Site" (ID: 183001)
- BGP Peer: "IPsec-BGP-Peer" (ID: 23895)
- Network: 10.201.1.0/24
- BGP Configuration: ASN 65100, IP 169.254.200.1

---

## Next Steps

1. **Monitor Ongoing Reconciliation:**
   ```bash
   flux get kustomizations
   flux get helmreleases -A
   kubectl get terraform -A
   ```

2. **View Logs:**
   ```bash
   # TF-Controller logs
   kubectl logs -n flux-system deployment/tofu-controller --tail=50
   
   # Flux logs
   flux logs --level=info
   ```

3. **Make Changes:**
   - Edit Terraform files locally
   - Commit and push to Git
   - Flux automatically reconciles within 5 minutes
   - Or force immediate reconciliation:
     ```bash
     flux reconcile source git flux-system
     flux reconcile terraform cato-praveen-ipsec-bgp
     ```

4. **Destroy Resources (if needed):**
   ```bash
   kubectl delete terraform cato-praveen-ipsec-bgp -n flux-system
   ```

---

## References

- [Flux Documentation](https://fluxcd.io/docs/)
- [Weave GitOps Documentation](https://docs.gitops.weave.works/)
- [TF-Controller Documentation](https://weaveworks.github.io/tf-controller/)
- [Cato Networks Provider](https://registry.terraform.io/providers/catonetworks/cato/latest/docs)
- [GitHub Repository](https://github.com/praveenkkishore/weaveworks-fleet-infra-test)
