# Terraform Architecture Comparison

## Request Profile

| Parameter | Value |
|---|---|
| **Customers** | 10,000 |
| **Devices per customer** | 100 |
| **Total resources** | 1,000,000 |
| **Request rate** | 500/min (~8.3/sec) |
| **Operation type** | Create / Edit / Delete (per customer site) |

---

## Head-to-Head Comparison

| Factor | Your Go SDK (`terraform-exec`) | Tofu-Controller (Flux) | Atlantis | Terraform Operator (GalleyBytes) | Crossplane TF Provider | Direct API (no TF) |
|---|---|---|---|---|---|---|
| **Can handle 500 req/min?** | ✅ Yes (worker pool) | ❌ No | ❌ No | ⚠️ Barely | ⚠️ Barely | ✅ Yes |
| **Max throughput** | ~50-100 req/min per worker × N workers | ~10-20 req/min | ~20-30 req/min | ~30-50 req/min | ~20-40 req/min | ~500+ req/min |
| **Workers needed for 500/min** | 10-15 goroutines | 25-50 pods | 15-25 pods | 10-17 pods | 12-25 pods | 5-10 goroutines |
| **RAM per operation** | 200MB (TF process) | 200MB + 100MB (controller) | 200MB + 50MB | 200MB + 80MB | 150MB | 5-10MB |
| **RAM for 500/min** | 2-3 GB | 7-15 GB | 5-7 GB | 3-5 GB | 3-6 GB | 50-100 MB |
| **Disk per operation** | 200MB (provider cache) | 200MB + PVC | 200MB + clone | 200MB + PVC | 150MB | 0 |
| **Latency per op** | 20-60s | 30-120s (reconcile loop) | 30-90s | 20-60s | 20-60s | 1-5s |
| **State backend** | PG ✅ (existing Aurora) | K8s Secret / S3 | Any TF backend | Any TF backend | K8s Secret | Your own DB |
| **State locking** | PG advisory locks ✅ | K8s lease | Built-in | Built-in | K8s lease | Your own DB locks |
| **Kafka/Dapr trigger** | ✅ You wire it | ❌ Git only | ❌ PR only | ❌ CRD only | ❌ CRD only | ✅ You wire it |
| **Drift detection** | ❌ Manual | ✅ Auto reconcile | ❌ Manual | ✅ Auto reconcile | ✅ Auto reconcile | ❌ Manual |
| **Multi-vendor (Cato+Versa+...)** | ✅ Any TF provider | ✅ Any TF provider | ✅ Any TF provider | ✅ Any TF provider | ✅ Any TF provider | ❌ Code per vendor |
| **New vendor onboarding** | Add `.tf` template | Add `.tf` in Git | Add `.tf` in repo | Add `.tf` in CRD | Add `.tf` in CRD | Write new API client |
| **Infra you already have** | ✅ Go + PG + Kafka | ❌ Need Flux cluster | ❌ Need Atlantis server | ❌ Need operator | ❌ Need Crossplane | ✅ Go + PG + Kafka |
| **New infra needed** | None | Flux + controller + PVCs | Atlantis server + webhooks | Operator + CRDs + PVCs | Crossplane + provider | None |
| **K8s dependency** | ❌ Optional | ✅ Required | ❌ Optional | ✅ Required | ✅ Required | ❌ Optional |
| **Complexity** | Medium | High | Medium | Medium-High | High | Low (but per-vendor) |
| **Observability** | Your Prometheus + DB | Controller metrics | Webhooks + logs | Controller metrics | Crossplane metrics | Your Prometheus + DB |
| **Retry/FSM** | ✅ You build it (like your adapter) | ✅ Built-in reconcile | ⚠️ Manual re-run | ✅ Built-in reconcile | ✅ Built-in reconcile | ✅ You build it |
| **Cost at scale** | $0 (runs in your pods) | $$$ (25-50 extra pods) | $$ (15-25 pods) | $$ (10-17 pods) | $$ (12-25 pods) | $0 (runs in your pods) |

---

## Throughput Bottleneck Analysis at 500 req/min

| Bottleneck | Go SDK | Tofu-Controller | Atlantis | TF Operator | Crossplane TF |
|---|---|---|---|---|---|
| `terraform init` (provider download) | ⚠️ Cache per worker dir | ⚠️ Per pod init | ⚠️ Per workspace | ⚠️ Per pod init | ⚠️ Per pod init |
| Provider API rate limit (Cato/Versa) | 🔴 **Real bottleneck** | 🔴 Same | 🔴 Same | 🔴 Same | 🔴 Same |
| State lock contention | ⚠️ PG lock per customer | ⚠️ K8s lease | ⚠️ Backend lock | ⚠️ Backend lock | ⚠️ K8s lease |
| Disk I/O | ⚠️ Provider binaries | 🔴 PVC provisioning | ⚠️ Git clone | 🔴 PVC provisioning | ⚠️ In-memory |
| Memory pressure | ⚠️ 200MB × workers | 🔴 300MB × pods | ⚠️ 250MB × pods | ⚠️ 280MB × pods | ⚠️ 150MB × pods |
| Scheduling overhead | ✅ None (goroutines) | 🔴 Pod scheduling | ✅ Thread pool | 🔴 Pod scheduling | 🔴 Pod scheduling |

> **Key insight**: At 500 req/min, the **real bottleneck is always the vendor API rate limit** (Cato, Versa, etc.), not the Terraform execution layer. Every option hits the same wall.

---

## Verdict

| If you need... | Use |
|---|---|
| **500 req/min + Kafka-driven + multi-vendor** | **Your Go SDK** — only option that checks all boxes without new infra |
| **GitOps + drift detection + low volume** | Tofu-Controller |
| **PR-based workflow + audit trail** | Atlantis |
| **K8s-native + reconciliation** | TF Operator or Crossplane |
| **Maximum throughput (no TF overhead)** | Direct API (but lose multi-vendor portability) |

---

## Recommended Architecture

```
┌────────────────────────────────────────────────────────────┐
│                    Kafka / Dapr PubSub                      │
└──────┬────────────────┬───────────────────┬────────────────┘
       │                │                   │
       ▼                ▼                   ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐
│  Site Setup   │ │ Tunnel Setup │ │  Device Programming   │
│  (Terraform)  │ │ (Terraform)  │ │  (Direct API / Go)    │
│              │ │              │ │                      │
│ 1 per cust   │ │ 2-4 per site │ │ 100 per customer     │
│ = 10K total  │ │ = 40K total  │ │ = 1,000,000 total    │
│ ✅ fits TF    │ │ ✅ fits TF    │ │ ❌ too many for TF    │
└──────────────┘ └──────────────┘ └──────────────────────┘
       │                │                   │
       └────────────────┴───────────────────┘
                        │
                  ┌─────▼──────┐
                  │ PostgreSQL │
                  │ TF State + │
                  │ App State  │
                  └────────────┘
```

### Layer Recommendations

| Layer | Volume | Tool |
|---|---|---|
| **Site provisioning** (10K sites) | ~500 req/min peak | Go SDK (`terraform-exec`) + PG state |
| **Tunnel/BGP setup** (40K tunnels) | ~500 req/min peak | Go SDK (`terraform-exec`) + PG state |
| **Device programming** (1M devices) | ~500 req/min | Direct vendor API via SDWAN adapter dispatcher |
| **Config updates** (ongoing) | ~500 req/min | Direct vendor API via SDWAN adapter dispatcher |

### Why Go SDK Wins

1. **No new infrastructure** — reuses existing Go + PostgreSQL + Kafka stack
2. **Kafka-native** — only Go SDK and Direct API support Kafka triggers; all controllers need Git/CRD
3. **Resource efficient** — goroutines (8KB each) vs pods (200MB+ each)
4. **Already built and tested** — executor.go with PG state backend proven (Site 183007)
5. **Multi-vendor** — add new `.tf` templates for Cato, Versa, or any future vendor
6. **Observable** — state queryable via SQL, metrics via existing Prometheus setup

---

## State Backend: PostgreSQL vs S3

| Factor | PostgreSQL | S3 |
|---|---|---|
| Already available | ✅ Existing Aurora | ❌ Need new S3 + DynamoDB |
| Locking | ✅ Built-in advisory locks | ⚠️ Needs DynamoDB table |
| Per-customer isolation | ✅ Schema/workspace per customer | ✅ Key prefix per customer |
| Latency | ✅ Same VPC, <1ms | ⚠️ 5-20ms API call |
| Ops overhead | ✅ Zero — reuse existing DB | ❌ S3 bucket + DynamoDB + IAM |
| Cost at 10K customers | ✅ ~0 (10K rows in existing DB) | ⚠️ Cheap but not free |
| State inspection | ✅ SQL query | ⚠️ Download from S3 |
| Backup/Recovery | ✅ Comes with RDS backups | ✅ S3 versioning |

**Recommendation: PostgreSQL** — zero new infrastructure, built-in locking, already proven.
