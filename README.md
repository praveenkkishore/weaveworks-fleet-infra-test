# Weave GitOps + Terraform Controller Setup

## Overview

This repository contains the GitOps configuration for managing Kubernetes infrastructure using Flux and Weave GitOps with Terraform Controller (OpenTofu).

**Installed Components:**
- ✅ Flux v2.7.5
- ✅ Weave GitOps Dashboard v4.0.36
- ✅ Terraform Controller (tofu-controller)
- ✅ Rancher Desktop (Local Kubernetes)

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│ Git Repository (Source of Truth)                        │
│ https://github.com/praveenkkishore/                     │
│         weaveworks-fleet-infra-test                     │
└────────────────┬────────────────────────────────────────┘
                 │
                 ↓ (Flux watches & syncs)
┌─────────────────────────────────────────────────────────┐
│ Kubernetes Cluster (Rancher Desktop)                    │
│                                                          │
│  flux-system namespace:                                 │
│  ├── source-controller        (Git sync)                │
│  ├── kustomize-controller     (Apply manifests)         │
│  ├── helm-controller          (Helm releases)           │
│  ├── notification-controller  (Events)                  │
│  ├── tofu-controller (x3)     (Terraform execution)     │
│  └── ww-gitops-weave-gitops   (Web UI)                  │
└─────────────────────────────────────────────────────────┘
```

## Repository Structure

```
weaveworks-fleet-infra-test/
├── clusters/
│   └── my-cluster/
│       ├── flux-system/               # Flux core components
│       │   ├── gotk-components.yaml
│       │   ├── gotk-sync.yaml
│       │   └── kustomization.yaml
│       ├── infra/                     # Infrastructure components
│       │   ├── tf-controller.yaml     # Terraform Controller
│       │   └── kustomization.yaml
│       └── weave-gitops-dashboard.yaml # Weave GitOps UI
└── terraform/                         # Terraform configurations
    └── (your terraform modules here)
```

## Access & Credentials

### Weave GitOps Dashboard

**Access:**
```bash
kubectl port-forward -n flux-system svc/ww-gitops-weave-gitops 9001:9001
```

Then open: http://localhost:9001

**Login:**
- Username: `admin`
- Password: `PASSWORD12345!`

### Kubernetes Cluster

```bash
kubectl config get-contexts
# Current context: rancher-desktop

kubectl cluster-info
# Kubernetes control plane: https://127.0.0.1:6443
```

## Deployment Workflow

### How GitOps Works

1. **Commit changes** to this Git repository
2. **Flux detects** changes (polls every 1m by default)
3. **Flux applies** changes to the cluster automatically
4. **View status** in Weave GitOps dashboard

### Manual Reconciliation

Force Flux to sync immediately:

```bash
# Sync the entire flux-system
flux reconcile kustomization flux-system --with-source

# Sync specific resource
flux reconcile helmrelease ww-gitops -n flux-system
```

## Terraform Controller Usage

### Basic Terraform Resource

Create a Terraform resource that TF-Controller will execute:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: my-terraform-repo
  namespace: flux-system
spec:
  interval: 1m
  url: https://github.com/your-org/terraform-configs
  ref:
    branch: main
---
apiVersion: infra.contrib.fluxcd.io/v1alpha2
kind: Terraform
metadata:
  name: my-infrastructure
  namespace: flux-system
spec:
  interval: 5m
  approvePlan: auto  # or "" for manual approval
  path: ./terraform/infrastructure
  sourceRef:
    kind: GitRepository
    name: my-terraform-repo
  
  # Non-sensitive variables
  vars:
  - name: region
    value: "us-east-1"
  - name: environment
    value: "dev"
  
  # Sensitive variables from Kubernetes Secret
  varsFrom:
  - kind: Secret
    name: cloud-credentials
    varsKeys:
    - aws_access_key_id
    - aws_secret_access_key
  
  # Store outputs in a Secret
  writeOutputsToSecret:
    name: infrastructure-outputs
    outputs:
    - vpc_id
    - subnet_ids
```

### Approval Modes

**Auto-approve (CI/CD mode):**
```yaml
spec:
  approvePlan: auto
```
- Terraform applies automatically after successful plan
- Use for dev/staging environments

**Manual approval (Production mode):**
```yaml
spec:
  approvePlan: ""
```
- Plan is created but not applied
- Review plan, then manually approve:
```bash
kubectl annotate terraform my-infrastructure -n flux-system \
  infra.contrib.fluxcd.io/apply="approved"
```

### Managing Secrets

Create a secret for Terraform variables:

```bash
kubectl create secret generic cloud-credentials -n flux-system \
  --from-literal=aws_access_key_id='AKIAXXXXX' \
  --from-literal=aws_secret_access_key='xxxxx'
```

### Monitoring Terraform Resources

```bash
# List all Terraform resources
kubectl get terraform -A

# Watch for changes
kubectl get terraform -n flux-system -w

# Describe resource
kubectl describe terraform my-infrastructure -n flux-system

# View logs
kubectl logs -n flux-system deployment/tofu-controller -f

# Get Terraform outputs
kubectl get secret infrastructure-outputs -n flux-system -o yaml
```

## Integration with SDWAN Adapter

### Architecture Decision: Direct Go SDK vs Weave GitOps

**For SDWAN device configurations, use Go Terraform SDK directly in dispatcher:**

```
Client Request (Create Cato IPsec Site)
    ↓
apiserver (gRPC API)
    ↓
PostgreSQL (dispatcher_events table)
    ↓
dispatcher (Go Terraform SDK)
    ├── Generate Terraform files from event data
    ├── terraform init
    ├── terraform plan
    ├── terraform apply
    └── Update event status in DB
    ↓
Response to client
```

**Reserve Weave GitOps for:**
- Deploying SDWAN adapter infrastructure
- Managing Kubernetes resources
- Database provisioning (RDS)
- Network policies

### Why Go SDK for Device Configs?

✅ **Fast synchronous execution** - No Git commit/push/pull cycle
✅ **Event-driven** - Already fits dispatcher architecture
✅ **State in PostgreSQL** - Consistent with existing design
✅ **Immediate feedback** - Can return Terraform outputs to client
✅ **Vendor-agnostic** - Easy to switch providers (Cato, Versa, etc.)

### Example: Dispatcher with Terraform SDK

```go
// pkg/dispatcher/terraform/executor.go
package terraform

import (
    "context"
    "github.com/hashicorp/terraform-exec/tfexec"
)

type TerraformExecutor struct {
    workDir string
    tfBinary string
}

func (e *TerraformExecutor) ExecuteCatoIPsec(ctx context.Context, config *CatoIPsecConfig) error {
    // 1. Generate Terraform files
    tfDir := e.generateTerraformConfig(config)
    
    // 2. Initialize
    tf, err := tfexec.NewTerraform(tfDir, e.tfBinary)
    if err != nil {
        return err
    }
    
    err = tf.Init(ctx)
    if err != nil {
        return err
    }
    
    // 3. Plan
    planFile := filepath.Join(tfDir, "plan.tfplan")
    _, err = tf.Plan(ctx, tfexec.Out(planFile))
    if err != nil {
        return err
    }
    
    // 4. Apply
    err = tf.Apply(ctx, tfexec.DirOrPlan(planFile))
    if err != nil {
        return err
    }
    
    // 5. Get outputs
    outputs, err := tf.Output(ctx)
    
    return nil
}
```

### Terraform State Management

Store Terraform state in PostgreSQL:

```hcl
terraform {
  backend "pg" {
    conn_str = "postgres://user:pass@localhost:5432/sdwan_adapter"
    schema_name = "terraform_state"
  }
}
```

Or use the existing `configuration_txns` table pattern.

## Common Operations

### Update Components

```bash
# Update Flux
flux install --export > clusters/my-cluster/flux-system/gotk-components.yaml
git add -A && git commit -m "Update Flux" && git push

# Update TF-Controller
curl -s https://raw.githubusercontent.com/weaveworks/tf-controller/main/docs/release.yaml \
  > clusters/my-cluster/infra/tf-controller.yaml
git add -A && git commit -m "Update TF-Controller" && git push

# Update Weave GitOps
PASSWORD='your-password'
gitops create dashboard ww-gitops \
  --password=$PASSWORD \
  --export > clusters/my-cluster/weave-gitops-dashboard.yaml
git add -A && git commit -m "Update Weave GitOps" && git push
```

### Troubleshooting

**Check Flux status:**
```bash
flux get all
flux check
```

**Check specific component:**
```bash
kubectl get pods -n flux-system
kubectl logs -n flux-system deployment/kustomize-controller -f
```

**Terraform not applying:**
```bash
# Check HelmRelease status
kubectl get helmrelease -n flux-system

# Check Terraform resource
kubectl describe terraform -n flux-system <name>

# View controller logs
kubectl logs -n flux-system deployment/tofu-controller -f
```

**Dashboard not accessible:**
```bash
# Check pod status
kubectl get pods -n flux-system | grep gitops

# Check service
kubectl get svc -n flux-system ww-gitops-weave-gitops

# Restart port-forward
kubectl port-forward -n flux-system svc/ww-gitops-weave-gitops 9001:9001
```

## Best Practices

### Git Workflow

1. **Create feature branch:**
```bash
git checkout -b add-terraform-resource
```

2. **Make changes** to manifests

3. **Commit and push:**
```bash
git add -A
git commit -m "Add new Terraform resource"
git push origin add-terraform-resource
```

4. **Create PR** and review

5. **Merge to main** - Flux auto-deploys

### Security

- ✅ Store secrets in Kubernetes Secrets, not Git
- ✅ Use RBAC to limit access to sensitive resources
- ✅ Enable manual approval for production Terraform
- ✅ Use separate namespaces for different environments
- ✅ Encrypt secrets at rest (KMS, Sealed Secrets, SOPS)

### Terraform Best Practices

- ✅ Use remote state (PostgreSQL backend)
- ✅ Version lock providers
- ✅ Use `terragrunt` for complex multi-environment setups
- ✅ Store sensitive vars in Kubernetes Secrets
- ✅ Set appropriate `interval` (don't check too frequently)
- ✅ Use drift detection alerts

## Resources

### Documentation
- [Flux Documentation](https://fluxcd.io/docs/)
- [Weave GitOps Docs](https://docs.gitops.weave.works/)
- [TF-Controller Docs](https://weaveworks.github.io/tf-controller/)
- [Terraform Exec Go SDK](https://github.com/hashicorp/terraform-exec)

### GitHub Repositories
- [This Repository](https://github.com/praveenkkishore/weaveworks-fleet-infra-test)
- [SDWAN Adapter](https://github.com/Infoblox-CTO/ddiaas.sdwan.adapter)

### Tools
- [Flux CLI](https://fluxcd.io/docs/cmd/)
- [GitOps CLI](https://docs.gitops.weave.works/docs/installation/weave-gitops/)
- [Rancher Desktop](https://rancherdesktop.io/)

## Installation Summary

### What Was Installed

1. **Rancher Desktop** - Local Kubernetes cluster (K3s 1.34.4)
2. **Flux v2.7.5** - GitOps toolkit
3. **Weave GitOps v4.0.36** - Web UI dashboard
4. **TF-Controller** - Terraform execution in Kubernetes

### Installation Commands Executed

```bash
# Install Flux CLI
brew install fluxcd/tap/flux

# Install GitOps CLI
curl -L "https://github.com/weaveworks/weave-gitops/releases/download/v0.38.0/gitops-$(uname)-$(uname -m).tar.gz" | tar xz -C /tmp
sudo mv /tmp/gitops /usr/local/bin

# Bootstrap Flux
export GITHUB_USER=praveenkkishore
export GITHUB_TOKEN=<your-token>
flux bootstrap github \
  --owner=$GITHUB_USER \
  --repository=weaveworks-fleet-infra-test \
  --branch=main \
  --path=./clusters/my-cluster \
  --personal

# Deploy Weave GitOps
PASSWORD='PASSWORD12345!'
gitops create dashboard ww-gitops \
  --password=$PASSWORD \
  --export > ./clusters/my-cluster/weave-gitops-dashboard.yaml

# Fix API version
sed -i '' 's|helm.toolkit.fluxcd.io/v2beta1|helm.toolkit.fluxcd.io/v2|g' \
  clusters/my-cluster/weave-gitops-dashboard.yaml

# Deploy TF-Controller
mkdir -p ./clusters/my-cluster/infra
curl -s https://raw.githubusercontent.com/weaveworks/tf-controller/main/docs/release.yaml \
  > ./clusters/my-cluster/infra/tf-controller.yaml

cat > ./clusters/my-cluster/infra/kustomization.yaml << 'EOF'
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - tf-controller.yaml
EOF

# Commit and push
git add -A
git commit -m "Setup Weave GitOps with TF-Controller"
git push
```

---

**Created:** February 23, 2026  
**Author:** Praveen Kishore  
**Cluster:** Rancher Desktop (local)  
**Environment:** Development
