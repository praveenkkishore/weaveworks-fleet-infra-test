# Weaveworks Fleet Infra Test - Full Project Context

> Saved on: 4 March 2026
> Repository: github.com/praveenkkishore/weaveworks-fleet-infra-test
> Branch: main
> Latest Commit: 515560e

---

## 1. Project Purpose

A GitOps-powered Terraform automation project for provisioning **Cato Networks IPsec sites with BGP peering**. Built as a proof-of-concept for DDIaaS SD-WAN infrastructure provisioning.

### Two Approaches Implemented
1. **GitOps (Flux + TF-Controller)**: Git-driven, reconciliation-based Terraform
2. **Go SDK (`terraform-exec`)**: Programmatic, event-driven Terraform execution

---

## 2. Repository Structure

```
weaveworks-fleet-infra-test/
├── INSTALLATION_GUIDE.md          # ~850 lines, full GitOps stack setup
├── README.md                      # Project overview
├── TERRAFORM_INTEGRATION_DECISION.md  # Architecture decision record
├── clusters/
│   └── my-cluster/
│       ├── cato-praveen-ipsec.yaml     # Terraform CR for Cato site
│       ├── weave-gitops-dashboard.yaml # Weave GitOps UI (port 9001)
│       ├── flux-system/
│       │   ├── gotk-components.yaml    # Flux toolkit components
│       │   ├── gotk-sync.yaml          # Git sync config
│       │   └── kustomization.yaml      # Kustomize overlay
│       └── infra/
│           ├── kustomization.yaml      # Infra kustomization
│           └── tf-controller.yaml      # TF-Controller HelmRelease
├── terraform/
│   └── cato-ipsec-praveen/
│       ├── main.tf                     # Reference TF config (GitOps)
│       ├── variables.tf                # Variable definitions
│       └── outputs.tf                  # Output definitions
├── go_src/
│   ├── Makefile                        # Build + run automation
│   ├── README.md                       # Go SDK docs with real execution logs
│   ├── STATE_MANAGEMENT.md             # State backend guide + test results
│   ├── comparison.md                   # Architecture comparison at scale
│   ├── go.mod                          # Go 1.21, terraform-exec v0.20.0
│   ├── go.sum
│   ├── cmd/
│   │   └── cato-terraform/
│   │       └── main.go                 # CLI entry point
│   └── pkg/
│       └── terraform/
│           └── executor.go             # Core Terraform executor
└── bin/
    └── cato-terraform                  # Compiled binary
```

---

## 3. Git History

```
515560e Add Terraform architecture comparison doc for 10K customers at 500 req/min
23bf1c4 Add persistent state management support with real testing validation
59cb4b6 Fix Terraform syntax to match working GitOps config
ca34e8a Add Go Terraform SDK for Cato Networks automation
b172b2d Add comprehensive installation guide for Weave GitOps + TF-Controller
d692d32 Fix IP conflict: change BGP neighbor IP to 169.254.200.1
27a3236 Fix Cato provider registry URL for OpenTofu compatibility
43a7d3a Add Praveen Cato IPsec BGP Terraform configuration
329c12f Add Terraform integration architecture decision document
398872f Add comprehensive documentation
041ba24 Add TF-Controller with proper structure
35aad0b Fix tf-controller Helm repo URL
8be6abe Add Terraform Controller
5937fc5 Fix HelmRelease API version to v2
a198bbc Add Weave GitOps Dashboard
331adf1 Add Flux sync manifests
d84c576 Add Flux v2.7.5 component manifests
```

---

## 4. Infrastructure Stack

### Local Kubernetes (Rancher Desktop)
- **K3s**: v1.34.4 on Mac M4 ARM
- **Flux**: v2.7.5 (GitOps toolkit)
- **Weave GitOps Dashboard**: v4.0.36 at http://localhost:9001 (admin/password)
- **TF-Controller**: tofu-controller v0.16.1 (OpenTofu-based, 3 replicas)
- **Kubernetes context**: rancher-desktop

### Cato Networks
- **Provider**: registry.terraform.io/catonetworks/cato >= 0.0.38
- **API**: https://api.catonetworks.com/api/v1/graphql2
- **Account ID**: 17957
- **Token**: R=eu1|K=D5ACD90B5FFC9EE916E04192AD048C70

### Created Resources
| Resource | ID | Method | Status |
|---|---|---|---|
| Praveen-IPsec-BGP-Site | 183001 | GitOps (TF-Controller) | Created |
| Praveen-Go-SDK-Test-2 | 183005 | Go SDK | Created |
| State-Test-Demo | 183007 | Go SDK (local state) | Tested create/update/destroy |

---

## 5. Go SDK Details

### executor.go Architecture
```
CatoIPsecConfig (config struct)
    │
    ▼
NewCatoExecutor()
    ├── Creates temp working directory
    ├── Generates main.tf (with dynamic backend block)
    ├── Generates variables.tf
    ├── Generates terraform.tfvars
    └── Initializes tfexec.Terraform
    │
    ▼
Apply(ctx)
    ├── terraform init
    ├── terraform plan → planFile
    ├── terraform apply → planFile
    └── getOutputs() → CatoOutputs{SiteID, SiteName, BGPPeerID, BGPPeerName}
    │
    ▼
Destroy(ctx)
    ├── terraform init
    └── terraform destroy
    │
    ▼
Cleanup()
    └── os.RemoveAll(workDir)
```

### State Backend Support
| Backend | Config | Tested |
|---|---|---|
| `pg` (PostgreSQL) | `conn_str` + `schema_name` | ✅ Site 183007 |
| `s3` (AWS S3) | `bucket` + `key` per site | Not yet |
| `local` | `/tmp/terraform-state/{site}/terraform.tfstate` | ✅ Site 183007 |
| Ephemeral (default) | No backend block | ✅ Sites 183001, 183005 |

### CLI Flags
```
--token         Cato API token (or CATO_TOKEN env)
--account       Cato account ID (or CATO_ACCOUNT_ID env)
--site-name     IPsec site name (default: "Praveen-IPsec-BGP-Site")
--public-ip     Router public IP (default: "1.1.1.1")
--bgp-ip        BGP neighbor IP (default: "169.254.200.1")
--bgp-asn       BGP peer ASN (default: 65100)
--psk           IPsec pre-shared key (default: "praveen_infoblox")
--network       Native network range (default: "10.201.1.0/24")
--destroy       Destroy resources instead of creating
--state-backend State backend: pg, s3, local, or empty
--state-conn    Connection string (PG DSN or S3 bucket)
```

### Makefile Targets
```
make build     # Build binary to bin/cato-terraform
make run       # Create resources (go run)
make destroy   # Destroy resources
make custom    # Run with custom params
make env       # Show environment config
make test      # Run go tests
make fmt       # Format code
make lint      # Run golangci-lint
make clean     # Clean build artifacts
```

---

## 6. Missing Features (Identified but Not Implemented)

### P0 (Critical for Production)
- **Output extraction** — read back site/tunnel IDs after apply
- **Plan → Approve → Apply separation** — don't blindly apply
- **Timeout on operations** — prevent hung workers
- **Retry with exponential backoff** — vendor APIs flake

### P1 (Important)
- **terraform validate** — pre-flight syntax check
- **Provider caching** — shared cache across workers
- **Targeted destroy** — partial resource operations
- **terraform refresh** — state sync before updates
- **terraform import** — onboard existing resources
- **Force unlock** — crash recovery for orphaned locks

### P2 (Nice to Have)
- **Drift detection** — periodic plan to detect manual changes
- **Plan file storage** — audit trail for compliance
- **Cost estimation**
- **Policy as code** (OPA/Sentinel)

---

## 7. Scale Analysis

### Target Scale
- 10,000 customers
- 100 devices per customer
- 500 requests/min peak
- Operations: Create / Edit / Delete per customer site

### Architecture Decision
| Layer | Volume | Tool |
|---|---|---|
| Site provisioning (10K) | ~500 req/min | Go SDK (terraform-exec) + PG/S3 state |
| Tunnel/BGP setup (40K) | ~500 req/min | Go SDK (terraform-exec) + PG/S3 state |
| Device programming (1M) | ~500 req/min | Direct vendor API (SDWAN adapter dispatcher) |

### State Backend Decision
- **≤100 customers**: PostgreSQL (already available, tested)
- **10K customers**: S3 + DynamoDB (no PG connection pressure, HashiCorp reference arch)

### Why Go SDK Over Controllers
1. No new infrastructure — reuses Go + PG + Kafka
2. Kafka-native trigger support
3. Goroutines (8KB) vs pods (200MB+)
4. Already built and tested
5. Multi-vendor via .tf templates

Full comparison in `go_src/comparison.md`.

---

## 8. Key Decisions Made

1. **GitOps vs Go SDK**: Both implemented. GitOps for demo/low-volume, Go SDK for production at scale.
2. **TF-Controller**: tofu-controller (OpenTofu fork), not original weaveworks tf-controller.
3. **IP addressing**: Changed from 169.254.100.1 → 169.254.200.1 to avoid conflicts with existing infrastructure.
4. **Cato provider**: Must use `registry.terraform.io/catonetworks/cato` (not `hashicorp/cato`).
5. **Data source syntax**: `filters = [{ field, search, operation }]` array format (not `filter {}` blocks).
6. **BGP default_action**: Required field, set to `"ACCEPT"`.
7. **connection_type**: Removed — not a valid field in cato_ipsec_site resource.
8. **State management**: Added persistent backends to enable updates (not just create/destroy).

---

## 9. Kubernetes Resources (Running on Rancher Desktop)

### Flux System
```yaml
# flux-system namespace
- GitRepository: flux-system (watches github.com/praveenkkishore/weaveworks-fleet-infra-test)
- Kustomization: flux-system (reconciles clusters/my-cluster)
- HelmRelease: tofu-controller (3 replicas)
- HelmRelease: ww-gitops (Weave GitOps dashboard)
```

### Terraform CR
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
  vars:
  - name: router_public_ip
    value: "1.1.1.1"
  - name: bgp_peer_asn
    value: 65100
  - name: bgp_neighbor_ip
    value: "169.254.200.1"
  varsFrom:
  - kind: Secret
    name: cato-credentials
    varsKeys: [cato_token, cato_account_id, ipsec_psk]
  writeOutputsToSecret:
    name: cato-praveen-outputs
    outputs: [ipsec_site_id, ipsec_site_info]
```

### TF-Controller Config
```yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: tofu-controller
  namespace: flux-system
spec:
  chart:
    spec:
      chart: tofu-controller
      sourceRef:
        kind: HelmRepository
        name: tofu-controller
      version: '>=0.16.1'
  values:
    replicaCount: 3
    concurrency: 24
    resources:
      limits:
        cpu: 1000m
        memory: 2Gi
    runner:
      image:
        tag: v0.16.1
```

---

## 10. Useful Commands

```bash
# Check Flux status
flux get all

# Check Terraform resource
kubectl get terraform -n flux-system
kubectl describe terraform cato-praveen-ipsec-bgp -n flux-system

# View TF-Controller logs
kubectl logs -n flux-system -l app=tofu-controller -f

# Weave GitOps dashboard
kubectl port-forward svc/ww-gitops-weave-gitops -n flux-system 9001:9001

# Go SDK
cd go_src && make build && make run
cd go_src && make destroy
cd go_src && make custom SITE_NAME="My-Site" BGP_IP="169.254.201.1"

# With state backend
cd go_src && go run cmd/cato-terraform/main.go \
  --state-backend=local \
  --site-name="State-Test"
```

---

## 11. Issues Resolved During Setup

| Issue | Resolution |
|---|---|
| Flux binary conflict (`/usr/local/bin/flux`) | Removed old binary, used Homebrew |
| HelmRelease API `v2beta1` deprecated | Changed to `v2` |
| TF-Controller Helm 404 | Used tofu-controller GitHub release.yaml |
| Cato data source `city = "New York"` | Changed to `filters = [{ field, search, operation }]` |
| `connection_type` invalid field | Removed from config |
| Missing `default_action` on BGP peer | Added `"ACCEPT"` |
| IP conflict 169.254.100.1 | Changed to 169.254.200.1 |
| Ephemeral state = always recreate | Added PG/S3/local state backends |
| Network range overlaps between tests | Used unique ranges per test |

---

## 12. Relationship to SDWAN Adapter

This project serves as the **Terraform execution layer** for the DDIaaS SDWAN adapter:

```
ddiaas.sdwan.adapter (production service)
├── adapter   → CDC events from Kafka
├── dispatcher → Event processing with FSM
│   ├── DDiaasAssociationHandler
│   ├── InfobloxCreateHandler
│   ├── VersaProgramDeviceHandler
│   └── [Future] CatoTerraformHandler  ← would use go_src/pkg/terraform/executor.go
└── apiserver → gRPC/HTTP API

weaveworks-fleet-infra-test (this repo)
├── GitOps approach (Flux + TF-Controller) ← for low-volume/demo
└── Go SDK approach (terraform-exec)       ← for production integration
```

The Go SDK executor would be imported into the SDWAN adapter's dispatcher as a handler, triggered by Kafka events via the existing poller/worker pool pattern.
