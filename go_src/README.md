# Cato Terraform Go SDK

Go SDK for managing Cato Networks IPsec sites with BGP using Terraform.

## Prerequisites

- Go 1.21 or later
- Terraform installed and in PATH
- Cato Networks API credentials

## Quick Start

### 1. Set Environment Variables

```bash
export CATO_TOKEN='R=eu1|K=D5ACD90B5FFC9EE916E04192AD048C70'
export CATO_ACCOUNT_ID='17957'
```

### 2. Run with Make

```bash
cd go_src

# Show all available commands
make help

# Create resources with default configuration
make run
```

## Usage Examples

### Using Makefile (Recommended)

```bash
cd go_src

# Create resources with default settings
make run

# Create with custom parameters
make custom SITE_NAME="My-Site" BGP_IP="169.254.201.1" BGP_ASN=65101

# Build binary
make build

# Run the binary
make run-binary

# Destroy resources
make destroy

# Show current configuration
make env
```

### Using Go Run Directly

```bash
cd go_src

go run cmd/cato-terraform/main.go \
  --site-name="Praveen-IPsec-BGP-Site" \
  --public-ip="1.1.1.1" \
  --bgp-ip="169.254.200.1" \
  --bgp-asn=65100 \
  --psk="praveen_infoblox" \
  --network="10.201.1.0/24"
```

### Using Compiled Binary

```bash
# Binary location after build
./bin/cato-terraform \
  --site-name="Test-Site" \
  --bgp-ip="169.254.200.1" \
  --bgp-asn=65100
```

## Command Line Flags

| Flag | Description | Default | Required |
|------|-------------|---------|----------|
| `--token` | Cato API token | env: `CATO_TOKEN` | Yes |
| `--account` | Cato account ID | env: `CATO_ACCOUNT_ID` | Yes |
| `--site-name` | IPsec site name | `Praveen-IPsec-BGP-Site` | No |
| `--public-ip` | Router public IP address | `1.1.1.1` | No |
| `--bgp-ip` | BGP neighbor IP | `169.254.200.1` | No |
| `--bgp-asn` | BGP peer ASN | `65100` | No |
| `--psk` | IPsec pre-shared key | `praveen_infoblox` | No |
| `--network` | Native network range | `10.201.1.0/24` | No |
| `--destroy` | Destroy resources | `false` | No |

## Makefile Targets

```
help                 Show this help message
deps                 Download Go dependencies
build                Build the binary
run                  Run the application (create resources)
create               Alias for 'run' - create resources
destroy              Destroy resources
run-binary           Build and run the binary
test                 Run tests
fmt                  Format Go code
lint                 Run linter (requires golangci-lint)
clean                Clean build artifacts
env                  Show environment variables
custom               Run with custom parameters
example-site2        Create a second example site
```

## Real Execution Example

### Complete Workflow: Failure → Resolution → Success

#### Step 1: Initial Attempt - Network Range Conflict

```bash
pkishore@IB-CVH732KDFR go_src % go run cmd/cato-terraform/main.go \
                --site-name="Praveen-Go-SDK-Test-2" \
                --public-ip="1.1.1.1" \
                --bgp-ip="169.254.210.1" \
                --bgp-asn=65100 \
                --psk="praveen_infoblox" \
                --network="10.203.1.0/24"
Creating Cato IPsec site with BGP...
→ Initializing Terraform...
→ Planning changes...
→ Applying changes...
2026/02/24 00:58:16 Failed to apply: terraform apply failed: exit status 1

Error: Cato API error

  with cato_ipsec_site.ipsec_bgp,
  on main.tf line 38, in resource "cato_ipsec_site" "ipsec_bgp":
  38: resource "cato_ipsec_site" "ipsec_bgp" {

{"networkErrors":null,"graphqlErrors":[{"message":"Range
10.203.1.1-10.203.1.254 of site 'praveen-go-sdk-test-2' overlaps with range
10.203.1.1-10.203.1.254 of site
'Praveen-Go-SDK-Test-1'","path":["site","addIpsecIkeV2Site"]}]}
exit status 1
make: *** [run] Error 1
```

**Problem**: Network range `10.203.1.0/24` is already used by site `Praveen-Go-SDK-Test-1`

#### Step 2: Clean Up Conflicting Resource

```bash
pkishore@IB-CVH732KDFR go_src % SITE_NAME="Praveen-Go-SDK-Test-1" BGP_IP="169.254.210.1" make destroy NETWORK="10.203.1.0/24"
→ Destroying Cato resources...
go run cmd/cato-terraform/main.go --destroy
Destroying Cato IPsec site...
→ Initializing Terraform...
→ Destroying resources...
✓ Resources destroyed successfully
```

**Action**: Destroyed the conflicting site to free up resources

#### Step 3: Successful Creation with New Parameters

```bash
pkishore@IB-CVH732KDFR go_src % SITE_NAME="Praveen-Go-SDK-Test-2" BGP_IP="169.254.221.1" make run NETWORK="10.212.1.0/24"
→ Running Cato Terraform executor...
  CATO_TOKEN: ...
  CATO_ACCOUNT_ID: 17957

go run cmd/cato-terraform/main.go \
                --site-name="Praveen-Go-SDK-Test-2" \
                --public-ip="1.1.1.1" \
                --bgp-ip="169.254.221.1" \
                --bgp-asn=65100 \
                --psk="praveen_infoblox" \
                --network="10.212.1.0/24"
Creating Cato IPsec site with BGP...
→ Initializing Terraform...
→ Planning changes...
→ Applying changes...

✓ Resources created successfully!

Outputs:
  Site ID: 183005
  Site Name: Praveen-Go-SDK-Test-2
  BGP Peer ID: 23896
  BGP Peer Name: IPsec-BGP-Peer
```

**Result**: Successfully created site with:
- Network range: `10.212.1.0/24` (changed from `10.203.1.0/24`)
- BGP IP: `169.254.221.1` (changed from `169.254.210.1`)
- Site ID: `183005`
- BGP Peer ID: `23896`

### Key Takeaways

1. **Network Range Conflicts**: Each site must have a unique network range
2. **BGP IP Conflicts**: BGP neighbor IPs must be unique across sites
3. **Destroy Before Recreate**: Clean up conflicting resources before creating new ones
4. **Cato IP Address**: SDK uses `169.254.201.1` as private_cato_ip (hardcoded, ensure no conflicts)

## Project Structure

```
go_src/
├── cmd/
│   └── cato-terraform/
│       └── main.go           # CLI entry point
├── pkg/
│   └── terraform/
│       └── executor.go       # Terraform execution logic
├── go.mod                    # Go module dependencies
├── go.sum                    # Dependency checksums
├── Makefile                  # Build and run tasks
└── README.md                 # This file
```

## Integration with SDWAN Adapter

This SDK can be integrated into the SDWAN adapter service:

```go
import "github.com/praveenkkishore/weaveworks-fleet-infra-test/go_src/pkg/terraform"

func provisionCatoSite(ctx context.Context, req *ProvisionRequest) error {
    config := &terraform.CatoIPsecConfig{
        CatoToken:     getCatoToken(),
        AccountID:     req.AccountID,
        SiteName:      req.SiteName,
        PublicIP:      req.PublicIP,
        BGPNeighborIP: req.BGPNeighborIP,
        BGPASN:        req.BGPASN,
        IPsecPSK:      generatePSK(),
        NetworkRange:  req.NetworkRange,
    }

    executor, err := terraform.NewCatoExecutor(config)
    if err != nil {
        return err
    }
    defer executor.Cleanup()

    outputs, err := executor.Apply(ctx)
    if err != nil {
        return err
    }

    // Store outputs in database
    return storeProvisionedSite(outputs)
}
```

## Troubleshooting

### Error: "terraform: executable file not found in $PATH"
**Solution:** Install Terraform: `brew install terraform`

### Error: "Cato token and account ID are required"
**Solution:** Set environment variables or use flags:
```bash
export CATO_TOKEN='your-token'
export CATO_ACCOUNT_ID='your-account-id'
```

### Error: "IP address already in use"
**Solution:** Change the BGP IP address or destroy conflicting resource:
```bash
# Option 1: Use different BGP IP
make custom BGP_IP="169.254.221.1" NETWORK="10.212.1.0/24"

# Option 2: Destroy existing resource first
SITE_NAME="Conflicting-Site" make destroy
```

### Error: "Range X overlaps with range Y"
**Solution:** Use a different network range:
```bash
make run NETWORK="10.212.1.0/24"
```

## Development

```bash
# Format code
make fmt

# Run tests
make test

# Run linter (requires golangci-lint)
brew install golangci-lint
make lint
```

## Comparison: Go SDK vs GitOps

| Aspect | Go SDK (This) | GitOps (Weave) |
|--------|---------------|----------------|
| **Execution** | Direct, synchronous | Eventual consistency |
| **Speed** | Instant | 1-5 minutes delay |
| **Control** | Full programmatic | Git-based workflow |
| **State** | Flexible location | Kubernetes-stored |
| **Integration** | Easy with Go services | Requires Git ops |
| **Debugging** | Direct logs | Check TF-Controller |
| **Best For** | Device automation | Infrastructure as Code |

## License

MIT
