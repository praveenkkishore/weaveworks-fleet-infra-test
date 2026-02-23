package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-exec/tfexec"
)

type CatoIPsecConfig struct {
	CatoToken       string
	AccountID       string
	SiteName        string
	PublicIP        string
	BGPNeighborIP   string
	BGPASN          int
	IPsecPSK        string
	NetworkRange    string
	StateBackend    string // "pg", "s3", "local", or empty for ephemeral
	StateConnString string // PostgreSQL connection string or S3 bucket
}

type CatoOutputs struct {
	SiteID      string `json:"ipsec_site_id"`
	SiteName    string
	BGPPeerID   string
	BGPPeerName string
}

type CatoExecutor struct {
	config  *CatoIPsecConfig
	workDir string
	tf      *tfexec.Terraform
}

func NewCatoExecutor(config *CatoIPsecConfig) (*CatoExecutor, error) {
	// Create temporary working directory
	workDir, err := os.MkdirTemp("", "cato-terraform-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create work dir: %w", err)
	}

	executor := &CatoExecutor{
		config:  config,
		workDir: workDir,
	}

	// Generate Terraform files
	if err := executor.generateTerraformFiles(); err != nil {
		os.RemoveAll(workDir)
		return nil, err
	}

	// Initialize Terraform executor
	tf, err := tfexec.NewTerraform(workDir, "terraform")
	if err != nil {
		os.RemoveAll(workDir)
		return nil, fmt.Errorf("failed to create terraform executor: %w", err)
	}
	executor.tf = tf

	return executor, nil
}

func (e *CatoExecutor) generateTerraformFiles() error {
	// Generate backend configuration based on config
	backendConfig := ""
	switch e.config.StateBackend {
	case "pg":
		connStr := e.config.StateConnString
		if connStr == "" {
			connStr = os.Getenv("TF_STATE_POSTGRES_CONN")
		}
		if connStr == "" {
			connStr = "postgres://postgres:postgres@localhost:5433/terraform_state?sslmode=disable"
		}
		backendConfig = fmt.Sprintf(`
  backend "pg" {
    conn_str    = "%s"
    schema_name = "cato_sites"
  }`, connStr)
	case "s3":
		bucket := e.config.StateConnString
		if bucket == "" {
			bucket = os.Getenv("TF_STATE_S3_BUCKET")
		}
		backendConfig = fmt.Sprintf(`
  backend "s3" {
    bucket = "%s"
    key    = "cato-sites/%s/terraform.tfstate"
    region = "us-east-1"
  }`, bucket, e.config.SiteName)
	case "local":
		backendConfig = fmt.Sprintf(`
  backend "local" {
    path = "/tmp/terraform-state/%s/terraform.tfstate"
  }`, e.config.SiteName)
	default:
		// No backend = ephemeral state in temp directory
		backendConfig = ""
	}

	// Generate main.tf
	mainTF := fmt.Sprintf(`
terraform {
  required_version = ">= 1.5"
  required_providers {
    cato = {
      source  = "registry.terraform.io/catonetworks/cato"
      version = ">= 0.0.38"
    }
  }%s
}

provider "cato" {
  baseurl    = "https://api.catonetworks.com/api/v1/graphql2"
  token      = var.cato_token
  account_id = var.cato_account_id
}

data "cato_siteLocation" "ny" {
  filters = [{
    field     = "city"
    search    = "New York City"
    operation = "startsWith"
    },
    {
      field     = "state_name"
      search    = "New York"
      operation = "exact"
    },
    {
      field     = "country_name"
      search    = "United States"
      operation = "contains"
  }]
}

data "cato_allocatedIp" "public_ips" {}

resource "cato_ipsec_site" "ipsec_bgp" {
  name                  = var.site_name
  site_type             = "BRANCH"
  description           = "IPsec site with BGP peering"
  native_network_range  = var.network_range
  
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
          public_site_ip  = var.public_ip
          private_site_ip = var.bgp_neighbor_ip
          private_cato_ip = "169.254.210.1"
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

resource "cato_bgp_peer" "ipsec_bgp_peer" {
  site_id              = cato_ipsec_site.ipsec_bgp.id
  name                 = "IPsec-BGP-Peer"
  peer_ip              = var.bgp_neighbor_ip
  peer_asn             = var.bgp_asn
  cato_asn             = 65000
  default_action       = "ACCEPT"
  advertise_all_routes = true
}

output "ipsec_site_id" {
  value = cato_ipsec_site.ipsec_bgp.id
}

output "ipsec_site_info" {
  value = {
    site_id       = cato_ipsec_site.ipsec_bgp.id
    site_name     = cato_ipsec_site.ipsec_bgp.name
    bgp_peer_id   = cato_bgp_peer.ipsec_bgp_peer.id
    bgp_peer_name = cato_bgp_peer.ipsec_bgp_peer.name
  }
}
`, backendConfig)
	variablesTF := `
variable "cato_token" {
  type      = string
  sensitive = true
}

variable "cato_account_id" {
  type = string
}

variable "site_name" {
  type = string
}

variable "public_ip" {
  type = string
}

variable "bgp_neighbor_ip" {
  type = string
}

variable "bgp_asn" {
  type = number
}

variable "ipsec_psk" {
  type      = string
  sensitive = true
}

variable "network_range" {
  type = string
}
`

	// Write files
	if err := os.WriteFile(filepath.Join(e.workDir, "main.tf"), []byte(mainTF), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(e.workDir, "variables.tf"), []byte(variablesTF), 0644); err != nil {
		return err
	}

	// Generate terraform.tfvars
	tfvars := fmt.Sprintf(`cato_token      = "%s"
cato_account_id = "%s"
site_name       = "%s"
public_ip       = "%s"
bgp_neighbor_ip = "%s"
bgp_asn         = %d
ipsec_psk       = "%s"
network_range   = "%s"
`,
		e.config.CatoToken,
		e.config.AccountID,
		e.config.SiteName,
		e.config.PublicIP,
		e.config.BGPNeighborIP,
		e.config.BGPASN,
		e.config.IPsecPSK,
		e.config.NetworkRange,
	)

	return os.WriteFile(filepath.Join(e.workDir, "terraform.tfvars"), []byte(tfvars), 0644)
}

func (e *CatoExecutor) Apply(ctx context.Context) (*CatoOutputs, error) {
	// Initialize
	fmt.Println("→ Initializing Terraform...")
	if err := e.tf.Init(ctx); err != nil {
		return nil, fmt.Errorf("terraform init failed: %w", err)
	}

	// Plan
	fmt.Println("→ Planning changes...")
	planFile := filepath.Join(e.workDir, "tfplan")
	hasChanges, err := e.tf.Plan(ctx, tfexec.Out(planFile))
	if err != nil {
		return nil, fmt.Errorf("terraform plan failed: %w", err)
	}

	if !hasChanges {
		fmt.Println("  No changes detected")
		return e.getOutputs(ctx)
	}

	// Apply
	fmt.Println("→ Applying changes...")
	if err := e.tf.Apply(ctx, tfexec.DirOrPlan(planFile)); err != nil {
		return nil, fmt.Errorf("terraform apply failed: %w", err)
	}

	return e.getOutputs(ctx)
}

func (e *CatoExecutor) Destroy(ctx context.Context) error {
	// Initialize
	fmt.Println("→ Initializing Terraform...")
	if err := e.tf.Init(ctx); err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}

	// Destroy
	fmt.Println("→ Destroying resources...")
	return e.tf.Destroy(ctx)
}

func (e *CatoExecutor) getOutputs(ctx context.Context) (*CatoOutputs, error) {
	outputMap, err := e.tf.Output(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get outputs: %w", err)
	}

	outputs := &CatoOutputs{}

	if val, ok := outputMap["ipsec_site_id"]; ok {
		json.Unmarshal(val.Value, &outputs.SiteID)
	}

	if val, ok := outputMap["ipsec_site_info"]; ok {
		var info map[string]interface{}
		json.Unmarshal(val.Value, &info)
		if v, ok := info["site_name"].(string); ok {
			outputs.SiteName = v
		}
		if v, ok := info["bgp_peer_id"].(string); ok {
			outputs.BGPPeerID = v
		}
		if v, ok := info["bgp_peer_name"].(string); ok {
			outputs.BGPPeerName = v
		}
	}

	return outputs, nil
}

func (e *CatoExecutor) Cleanup() {
	if e.workDir != "" {
		os.RemoveAll(e.workDir)
	}
}
