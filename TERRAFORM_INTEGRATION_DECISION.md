# Terraform Integration Architecture Decision

## Question: Weave GitOps TF-Controller vs Go Terraform SDK in Dispatcher?

For the **DDIaaS SDWAN Adapter** project, where should Terraform execution happen for device configurations (Cato IPsec sites, Versa devices, etc.)?

## TL;DR: **Use Go Terraform SDK in Dispatcher** ✅

### Quick Comparison

| Aspect | Weave GitOps TF-Controller | Go Terraform SDK (Dispatcher) |
|--------|---------------------------|-------------------------------|
| **Use Case** | Infrastructure as Code (IaC) | Device Configuration Management |
| **Latency** | High (Git commit → Flux poll → reconcile) | Low (Direct execution) |
| **State Management** | Kubernetes Secrets/ConfigMaps | PostgreSQL (existing DB) |
| **Feedback Loop** | Asynchronous (watch events) | Synchronous (return outputs) |
| **Multi-tenancy** | Namespace isolation | Event-based isolation |
| **Blast Radius** | All resources in one CRD | Per-device event isolation |
| **Approval Workflow** | Manual annotation | Custom workflow in DB |
| **Best For** | Cluster infrastructure, VPCs, RDS | Device configs, IPsec sites, BGP |

---

## Detailed Analysis

### Option 1: Weave GitOps TF-Controller

**Architecture:**
```
Client Request
    ↓
apiserver (creates event)
    ↓
dispatcher (generates YAML)
    ↓
Git commit (push Terraform CRD)
    ↓
Flux polls Git (30s - 5m interval)
    ↓
TF-Controller reconciles
    ↓
Terraform applies to Cato/Versa
    ↓
(How does client get result? Poll K8s Secret?)
```

**Pros:**
- ✅ GitOps native - everything tracked in Git
- ✅ Declarative - desired state in version control
- ✅ Built-in drift detection
- ✅ Web UI for monitoring (Weave GitOps dashboard)
- ✅ Kubernetes-native RBAC
- ✅ Audit trail via Git history

**Cons:**
- ❌ **High latency** (5-10 minutes typical)
- ❌ Complex state retrieval (read K8s Secrets/ConfigMaps)
- ❌ Hard to return outputs to client synchronously
- ❌ Git repo grows with every device config
- ❌ Doesn't leverage existing event/DB architecture
- ❌ Multiple K8s resources per device = complexity
- ❌ Approval workflow via annotations (awkward)

**Example CRD per device:**
```yaml
apiVersion: infra.contrib.fluxcd.io/v1alpha2
kind: Terraform
metadata:
  name: cato-site-ny-ipsec-12345
  namespace: flux-system
spec:
  interval: 5m
  approvePlan: ""
  path: ./terraform/cato-ipsec
  sourceRef:
    kind: GitRepository
    name: device-configs
  vars:
  - name: site_name
    value: "Site-NY-IPsec"
  - name: network_range
    value: "10.200.1.0/24"
  varsFrom:
  - kind: Secret
    name: cato-credentials
```

**Timeline:**
1. API request: 0s
2. Dispatcher creates CRD: 1s
3. Git commit + push: 2s
4. Flux polls Git: 30s - 5m ⏰
5. TF-Controller reconciles: 10s
6. Terraform init/plan/apply: 30s - 2m
7. **Total: 1-8 minutes**

---

### Option 2: Go Terraform SDK in Dispatcher ✅ **Recommended**

**Architecture:**
```
Client Request
    ↓
apiserver (creates dispatcher_event in PostgreSQL)
    ↓
dispatcher (polls events table)
    ↓
Terraform Executor:
  - Generate .tf files from event data
  - terraform init
  - terraform plan
  - terraform apply
  - Store outputs in DB
    ↓
Update event status to "completed"
    ↓
apiserver polls event status
    ↓
Return response to client
```

**Pros:**
- ✅ **Low latency** (30s - 2m total)
- ✅ Synchronous execution with immediate feedback
- ✅ Leverages existing event-driven architecture
- ✅ State stored in PostgreSQL (consistent with existing design)
- ✅ Can return Terraform outputs directly to client
- ✅ No Git commit overhead
- ✅ Fits existing vendor processor pattern
- ✅ Custom approval workflow via DB state
- ✅ Easy rollback (update event, reprocess)
- ✅ Per-device event isolation

**Cons:**
- ❌ No GitOps audit trail (but DB has audit logs)
- ❌ Terraform state management responsibility (solvable)
- ❌ Need to implement drift detection (can be async job)
- ❌ No built-in web UI for Terraform (but you have apiserver API)

**Implementation Example:**

```go
// pkg/dispatcher/terraform/executor.go
package terraform

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    
    "github.com/hashicorp/terraform-exec/tfexec"
    "github.com/sirupsen/logrus"
)

type TerraformExecutor struct {
    logger    *logrus.Entry
    workDir   string
    tfBinary  string
    stateDB   StateManager
}

type StateManager interface {
    SaveState(ctx context.Context, resourceID string, state []byte) error
    LoadState(ctx context.Context, resourceID string) ([]byte, error)
}

// ExecuteEvent processes a dispatcher event with Terraform
func (e *TerraformExecutor) ExecuteEvent(ctx context.Context, event *DispatcherEvent) (*TerraformResult, error) {
    logger := e.logger.WithField("event_id", event.ID)
    
    // 1. Create working directory
    execDir := filepath.Join(e.workDir, fmt.Sprintf("event-%s", event.ID))
    if err := os.MkdirAll(execDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create exec dir: %w", err)
    }
    defer os.RemoveAll(execDir) // Cleanup
    
    // 2. Generate Terraform files based on vendor
    switch event.VendorType {
    case "cato":
        if err := e.generateCatoConfig(execDir, event.Payload); err != nil {
            return nil, err
        }
    case "versa":
        if err := e.generateVersaConfig(execDir, event.Payload); err != nil {
            return nil, err
        }
    default:
        return nil, fmt.Errorf("unsupported vendor: %s", event.VendorType)
    }
    
    // 3. Initialize Terraform
    tf, err := tfexec.NewTerraform(execDir, e.tfBinary)
    if err != nil {
        return nil, fmt.Errorf("failed to create terraform instance: %w", err)
    }
    
    // Set environment variables (credentials, etc.)
    tf.SetEnv(map[string]string{
        "CATO_TOKEN":      os.Getenv("CATO_TOKEN"),
        "CATO_ACCOUNT_ID": os.Getenv("CATO_ACCOUNT_ID"),
    })
    
    logger.Info("Initializing Terraform...")
    if err := tf.Init(ctx, tfexec.Upgrade(true)); err != nil {
        return nil, fmt.Errorf("terraform init failed: %w", err)
    }
    
    // 4. Load previous state if exists
    stateFile := filepath.Join(execDir, "terraform.tfstate")
    if state, err := e.stateDB.LoadState(ctx, event.ResourceID); err == nil {
        if err := os.WriteFile(stateFile, state, 0644); err != nil {
            logger.WithError(err).Warn("Failed to write previous state")
        }
    }
    
    // 5. Plan
    logger.Info("Running Terraform plan...")
    planFile := filepath.Join(execDir, "tfplan")
    hasChanges, err := tf.Plan(ctx, tfexec.Out(planFile))
    if err != nil {
        return nil, fmt.Errorf("terraform plan failed: %w", err)
    }
    
    if !hasChanges {
        logger.Info("No changes detected")
        return &TerraformResult{
            Changed: false,
            Message: "No changes required",
        }, nil
    }
    
    // 6. Apply (or wait for approval based on event)
    if event.RequiresApproval {
        logger.Info("Plan generated, awaiting approval")
        // Save plan for later approval
        return &TerraformResult{
            Changed:       false,
            Message:       "Awaiting approval",
            PlanGenerated: true,
        }, nil
    }
    
    logger.Info("Applying Terraform changes...")
    if err := tf.Apply(ctx, tfexec.DirOrPlan(planFile)); err != nil {
        return nil, fmt.Errorf("terraform apply failed: %w", err)
    }
    
    // 7. Get outputs
    outputs, err := tf.Output(ctx)
    if err != nil {
        logger.WithError(err).Warn("Failed to get outputs")
    }
    
    // 8. Save state
    newState, err := os.ReadFile(stateFile)
    if err != nil {
        logger.WithError(err).Error("Failed to read state file")
    } else {
        if err := e.stateDB.SaveState(ctx, event.ResourceID, newState); err != nil {
            logger.WithError(err).Error("Failed to save state")
        }
    }
    
    return &TerraformResult{
        Changed: true,
        Message: "Successfully applied",
        Outputs: convertOutputs(outputs),
    }, nil
}

// generateCatoConfig creates Terraform files for Cato device
func (e *TerraformExecutor) generateCatoConfig(dir string, payload json.RawMessage) error {
    var config CatoIPsecConfig
    if err := json.Unmarshal(payload, &config); err != nil {
        return err
    }
    
    // Generate main.tf
    mainTF := fmt.Sprintf(`
terraform {
  required_version = ">= 1.5"
  required_providers {
    cato = {
      source  = "catonetworks/cato"
      version = ">= 0.0.38"
    }
  }
}

provider "cato" {
  baseurl    = "https://api.catonetworks.com/api/v1/graphql2"
  token      = var.cato_token
  account_id = var.cato_account_id
}

resource "cato_ipsec_site" "site" {
  name                 = %q
  site_type            = "BRANCH"
  native_network_range = %q
  
  ipsec = {
    primary = {
      tunnels = [{
        public_site_ip  = %q
        private_site_ip = %q
        private_cato_ip = %q
        psk             = var.ipsec_psk
      }]
    }
  }
}

resource "cato_bgp_peer" "peer" {
  site_id  = cato_ipsec_site.site.id
  name     = "BGP-Peer"
  peer_ip  = %q
  peer_asn = %d
  cato_asn = 65000
}

output "site_id" {
  value = cato_ipsec_site.site.id
}

output "bgp_peer_id" {
  value = cato_bgp_peer.peer.id
}
`, config.SiteName, config.NetworkRange, config.PublicIP, 
   config.BGPNeighborIP, config.CatoPrivateIP, 
   config.BGPNeighborIP, config.BGPPeerASN)
    
    if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(mainTF), 0644); err != nil {
        return err
    }
    
    // Generate variables.tf
    variablesTF := `
variable "cato_token" {
  type      = string
  sensitive = true
}

variable "cato_account_id" {
  type = number
}

variable "ipsec_psk" {
  type      = string
  sensitive = true
}
`
    return os.WriteFile(filepath.Join(dir, "variables.tf"), []byte(variablesTF), 0644)
}

type TerraformResult struct {
    Changed       bool
    Message       string
    PlanGenerated bool
    Outputs       map[string]interface{}
}

type CatoIPsecConfig struct {
    SiteName      string `json:"site_name"`
    NetworkRange  string `json:"network_range"`
    PublicIP      string `json:"public_ip"`
    BGPNeighborIP string `json:"bgp_neighbor_ip"`
    CatoPrivateIP string `json:"cato_private_ip"`
    BGPPeerASN    int    `json:"bgp_peer_asn"`
}
```

**State Management in PostgreSQL:**

```sql
-- Add to existing schema
CREATE TABLE terraform_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_id VARCHAR(255) NOT NULL UNIQUE,
    resource_type VARCHAR(100) NOT NULL,
    state_data BYTEA NOT NULL,
    state_version INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255)
);

CREATE INDEX idx_terraform_state_resource ON terraform_state(resource_id);
```

**Timeline:**
1. API request: 0s
2. Event created in DB: 0.1s
3. Dispatcher picks up event: 1s
4. Terraform init/plan/apply: 30s - 2m
5. State saved to DB: 0.5s
6. Event marked complete: 0.1s
7. **Total: 30s - 2.5 minutes**

---

## Hybrid Approach: Best of Both Worlds

**Recommendation: Use both, but for different purposes**

### Weave GitOps TF-Controller: Infrastructure Layer

Use for **long-lived infrastructure** that changes infrequently:

```yaml
# Deploy SDWAN adapter infrastructure
apiVersion: infra.contrib.fluxcd.io/v1alpha2
kind: Terraform
metadata:
  name: sdwan-infrastructure
  namespace: flux-system
spec:
  interval: 1h
  approvePlan: ""  # Manual approval for infra
  path: ./terraform/infrastructure
  sourceRef:
    kind: GitRepository
    name: flux-system
  vars:
  - name: environment
    value: "production"
```

**What to manage:**
- RDS PostgreSQL database
- VPC, subnets, security groups
- IAM roles and policies
- S3 buckets for backups
- CloudWatch alarms
- Kubernetes cluster resources (if applicable)

### Go Terraform SDK: Application Layer

Use for **device configurations** that change frequently:

```go
// In dispatcher/handlers/cato/handler.go
func (h *CatoHandler) HandleCreateIPsec(ctx context.Context, event *DispatcherEvent) error {
    executor := terraform.NewExecutor(h.logger, h.config)
    
    result, err := executor.ExecuteEvent(ctx, event)
    if err != nil {
        return h.markEventFailed(event.ID, err)
    }
    
    // Store outputs in database
    h.storage.UpdateEventOutputs(ctx, event.ID, result.Outputs)
    
    return h.markEventCompleted(event.ID)
}
```

**What to manage:**
- Cato IPsec sites
- Cato BGP peers
- Versa device configurations
- SD-WAN tunnel configurations
- Any device-specific settings

---

## Implementation Checklist

### Phase 1: Infrastructure via Weave GitOps (Already Done ✅)

- [x] Install Flux
- [x] Install Weave GitOps Dashboard
- [x] Install TF-Controller
- [ ] Create Terraform module for SDWAN adapter infrastructure
- [ ] Deploy RDS database via Terraform CRD
- [ ] Set up monitoring/alerting via Terraform

### Phase 2: Go Terraform SDK in Dispatcher

- [ ] Add `terraform-exec` dependency to `go.mod`
```bash
go get github.com/hashicorp/terraform-exec@latest
```

- [ ] Create `pkg/dispatcher/terraform/` package
- [ ] Implement `TerraformExecutor` interface
- [ ] Add `terraform_state` table migration
- [ ] Implement Cato provider config generation
- [ ] Implement Versa provider config generation
- [ ] Add Terraform execution to dispatcher event handlers
- [ ] Add unit tests for Terraform generation
- [ ] Add integration tests with mock Terraform
- [ ] Update API to return Terraform outputs

### Phase 3: Testing & Validation

- [ ] Test Cato IPsec site creation
- [ ] Test BGP peer configuration
- [ ] Test state management (create → update → delete)
- [ ] Test concurrent execution (multiple events)
- [ ] Test failure recovery
- [ ] Load testing (100+ devices)
- [ ] Drift detection job (periodic state refresh)

---

## State Management Strategy

### Option A: PostgreSQL Backend (Recommended)

Store Terraform state directly in PostgreSQL:

```hcl
terraform {
  backend "pg" {
    conn_str = "postgres://user:pass@localhost:5432/sdwan_adapter"
    schema_name = "terraform_state"
  }
}
```

**Pros:**
- Native Terraform backend
- Automatic locking
- Consistent with existing architecture

**Cons:**
- Requires PostgreSQL backend support in Terraform

### Option B: Custom State Storage

Store as BYTEA in custom table (shown above):

```go
type PostgresStateManager struct {
    db *sql.DB
}

func (s *PostgresStateManager) SaveState(ctx context.Context, resourceID string, state []byte) error {
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO terraform_state (resource_id, resource_type, state_data, state_version)
        VALUES ($1, $2, $3, 1)
        ON CONFLICT (resource_id) 
        DO UPDATE SET 
            state_data = $3,
            state_version = terraform_state.state_version + 1,
            updated_at = CURRENT_TIMESTAMP
    `, resourceID, "cato_ipsec", state)
    return err
}
```

**Pros:**
- Full control over state storage
- Can add custom metadata
- Easy to backup/restore

**Cons:**
- Need to implement locking manually
- More code to maintain

---

## Conclusion

### Final Recommendation

**Use Go Terraform SDK in the dispatcher for device configurations:**

1. **Faster** - No Git commit/push/poll cycle
2. **Better UX** - Synchronous responses with outputs
3. **Simpler** - Fits existing event-driven architecture
4. **Scalable** - PostgreSQL state management
5. **Maintainable** - Less moving parts than GitOps for device configs

**Reserve Weave GitOps TF-Controller for:**
- Infrastructure provisioning (RDS, VPC, etc.)
- Cluster-level resources
- Infrequent changes with strong audit requirements

### Migration Path

1. ✅ **Done:** Weave GitOps infrastructure setup
2. **Next:** Implement Go Terraform SDK in dispatcher
3. **Then:** Deploy SDWAN adapter infra via Weave GitOps
4. **Finally:** Handle device configs via Go SDK

---

## Questions & Answers

**Q: Can we have both?**  
A: Yes! Use Weave GitOps for infra, Go SDK for devices.

**Q: What about drift detection?**  
A: Implement periodic job that re-runs `terraform plan` and alerts on drift.

**Q: How to handle manual approval?**  
A: Add `requires_approval` flag to events, generate plan, wait for API approval call.

**Q: State locking for concurrent executions?**  
A: Use PostgreSQL advisory locks or event-level locking (already have event processor).

**Q: Terraform version management?**  
A: Pin version in Docker image or use `tfenv`.

**Q: How to test without real Cato account?**  
A: Mock Terraform provider or use localstack-style mock server.

---

**Decision Date:** February 23, 2026  
**Authors:** Architecture Team  
**Status:** Recommended ✅  
**Review Date:** Quarterly
