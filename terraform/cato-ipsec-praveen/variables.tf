variable "cato_token" {
  type      = string
  sensitive = true
}

variable "cato_account_id" {
  type = number
}

variable "router_public_ip" {
  type        = string
  description = "Router public IP (update when available)"
  default     = "1.1.1.1" # Placeholder
}

variable "ipsec_psk" {
  type      = string
  sensitive = true
  default   = "praveen_infoblox"
}

variable "bgp_peer_asn" {
  type    = number
  default = 65100
}

variable "bgp_neighbor_ip" {
  type    = string
  default = "169.254.100.1"
}

variable "onprem_networks" {
  type    = list(string)
  default = ["192.168.100.0/24"]
}
