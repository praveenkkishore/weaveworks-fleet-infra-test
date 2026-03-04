# Quick Resume Guide

> Use this file to quickly resume work on this project in a new AI session.

## One-Line Summary
Go + Terraform SDK for Cato Networks IPsec/BGP site provisioning, with GitOps (Flux/TF-Controller) as secondary approach. Production target: 10K customers, 500 req/min.

## Current State (4 March 2026)
- **Go SDK**: Working — creates/updates/destroys Cato sites with PG/local state backends
- **GitOps**: Working — Flux + tofu-controller on Rancher Desktop
- **Missing for production**: Output extraction, plan/apply split, retry, timeout, S3 state backend

## Key Files
```
go_src/pkg/terraform/executor.go  ← Core logic (342 lines)
go_src/cmd/cato-terraform/main.go ← CLI entry point
go_src/Makefile                   ← Build automation
go_src/comparison.md              ← Scale analysis at 10K customers
go_src/STATE_MANAGEMENT.md        ← State backend testing results
```

## Quick Commands
```bash
cd go_src && make build && make run          # Create site
cd go_src && make destroy                    # Destroy site
cd go_src && make custom SITE_NAME="X"       # Custom params
flux get all                                 # Check GitOps status
kubectl get terraform -n flux-system         # Check TF resources
```

## Resume Points
- To add S3 backend: Update `executor.go` `generateTerraformFiles()` S3 case + implement `-backend-config` in Init()
- To add retry: Wrap `Apply()` and `Destroy()` with exponential backoff
- To add timeout: `context.WithTimeout(ctx, 5*time.Minute)` before each TF operation
- To integrate with SDWAN adapter: Import `pkg/terraform` in dispatcher handler, trigger via Kafka event
