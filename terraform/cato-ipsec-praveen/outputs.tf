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
