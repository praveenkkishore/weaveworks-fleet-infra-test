# Terraform and Provider Requirements
terraform {
  required_version = ">= 1.5"
  required_providers {
    cato = {
      source  = "catonetworks/cato"
      version = ">= 0.0.38"
    }
  }
}

# Cato Provider Configuration
provider "cato" {
  baseurl    = "https://api.catonetworks.com/api/v1/graphql2"
  token      = var.cato_token
  account_id = var.cato_account_id
}

// Data Source for site location
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

// Data Source for Cato allocated IPs
data "cato_allocatedIp" "public_ips" {
}

# IPsec Site with BGP
resource "cato_ipsec_site" "ipsec_bgp" {
  name        = "Praveen-IPsec-BGP-Site"
  site_type   = "BRANCH"
  description = "IPsec site with BGP routing"
  
  # Native network range - required field (string)
  native_network_range = "10.201.1.0/24"
  
  # Use data source to lookup valid site location
  site_location = {
    city         = data.cato_siteLocation.ny.locations[0].city
    country_code = data.cato_siteLocation.ny.locations[0].country_code
    state_code   = data.cato_siteLocation.ny.locations[0].state_code
    timezone     = data.cato_siteLocation.ny.locations[0].timezone[0]
    address      = "555 That Way"
  }
  
  # IPsec Configuration
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

# BGP Configuration for the IPsec Site
resource "cato_bgp_peer" "ipsec_bgp_peer" {
  site_id              = cato_ipsec_site.ipsec_bgp.id
  name                 = "IPsec-BGP-Peer"
  peer_ip              = var.bgp_neighbor_ip
  peer_asn             = var.bgp_peer_asn
  cato_asn             = 65000
  default_action       = "ACCEPT"
  advertise_all_routes = true
}
