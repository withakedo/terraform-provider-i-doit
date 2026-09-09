resource "idoit_layer3_net" "servers" {
  title       = "10.0.0.0/24 - servers"
  type        = "ipv4"
  address     = "10.0.0.0"
  cidr_suffix = "24"

  dns_server      = "10.0.0.1"
  dns_domain      = "example.internal"
  default_gateway = "10.0.0.1"
  range_from      = "10.0.0.10"
  range_to        = "10.0.0.250"

  purge_on_destroy = true

  # version-specific C__CATS__NET keys, passed through untouched
  extra = {
    dhcp = "0"
  }
}

output "servers_net_id" {
  value = idoit_layer3_net.servers.id
}
