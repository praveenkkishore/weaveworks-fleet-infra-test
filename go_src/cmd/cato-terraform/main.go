package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/praveenkkishore/weaveworks-fleet-infra-test/go_src/pkg/terraform"
)

func main() {
	// Command line flags
	catoToken := flag.String("token", "", "Cato API token (or set CATO_TOKEN env)")
	accountID := flag.String("account", "", "Cato account ID (or set CATO_ACCOUNT_ID env)")
	siteName := flag.String("site-name", "Praveen-IPsec-BGP-Site", "IPsec site name")
	publicIP := flag.String("public-ip", "1.1.1.1", "Router public IP")
	bgpNeighborIP := flag.String("bgp-ip", "169.254.200.1", "BGP neighbor IP")
	bgpASN := flag.Int("bgp-asn", 65100, "BGP peer ASN")
	ipsecPSK := flag.String("psk", "praveen_infoblox", "IPsec pre-shared key")
	networkRange := flag.String("network", "10.201.1.0/24", "Native network range")
	destroy := flag.Bool("destroy", false, "Destroy resources instead of creating")

	// State backend options
	stateBackend := flag.String("state-backend", "", "State backend: pg, s3, local, or empty for ephemeral")
	stateConn := flag.String("state-conn", "", "State connection string (Postgres DSN or S3 bucket)")

	flag.Parse()

	// Get credentials from env if not provided via flags
	if *catoToken == "" {
		*catoToken = os.Getenv("CATO_TOKEN")
	}
	if *accountID == "" {
		*accountID = os.Getenv("CATO_ACCOUNT_ID")
	}

	if *catoToken == "" || *accountID == "" {
		log.Fatal("Cato token and account ID are required (via flags or environment)")
	}

	// Create configuration
	config := &terraform.CatoIPsecConfig{
		CatoToken:       *catoToken,
		AccountID:       *accountID,
		SiteName:        *siteName,
		PublicIP:        *publicIP,
		BGPNeighborIP:   *bgpNeighborIP,
		BGPASN:          *bgpASN,
		IPsecPSK:        *ipsecPSK,
		NetworkRange:    *networkRange,
		StateBackend:    *stateBackend,
		StateConnString: *stateConn,
	}

	// Create executor
	executor, err := terraform.NewCatoExecutor(config)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}
	defer executor.Cleanup()

	ctx := context.Background()

	if *destroy {
		// Destroy resources
		fmt.Println("Destroying Cato IPsec site...")
		if err := executor.Destroy(ctx); err != nil {
			log.Fatalf("Failed to destroy: %v", err)
		}
		fmt.Println("✓ Resources destroyed successfully")
	} else {
		// Create resources
		fmt.Println("Creating Cato IPsec site with BGP...")
		outputs, err := executor.Apply(ctx)
		if err != nil {
			log.Fatalf("Failed to apply: %v", err)
		}

		// Display outputs
		fmt.Println("\n✓ Resources created successfully!")
		fmt.Println("\nOutputs:")
		fmt.Printf("  Site ID: %s\n", outputs.SiteID)
		fmt.Printf("  Site Name: %s\n", outputs.SiteName)
		fmt.Printf("  BGP Peer ID: %s\n", outputs.BGPPeerID)
		fmt.Printf("  BGP Peer Name: %s\n", outputs.BGPPeerName)
	}
}
