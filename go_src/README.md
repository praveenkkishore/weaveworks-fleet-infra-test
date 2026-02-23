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

## Output Example

```
Creating Cato IPsec site with BGP...
→ Initializing Terraform...
→ Planning changes...
→ Applying changes...

✓ Resources created successfully!

Outputs:
  Site ID: 183001
  Site Name: Praveen-IPsec-BGP-Site
  BGP Peer ID: 23895
  BGP Peer Name: IPsec-BGP-Peer
```

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
**Solution:** Change the BGP IP address:
```bash
make custom BGP_IP="169.254.201.1"
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
