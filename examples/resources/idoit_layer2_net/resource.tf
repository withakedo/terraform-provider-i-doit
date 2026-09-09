resource "idoit_layer3_net" "servers" {
  title       = "10.0.0.0/24 - servers"
  address     = "10.0.0.0"
  cidr_suffix = "24"
}

resource "idoit_layer2_net" "vlan100" {
  title          = "VLAN 100 - servers"
  vlan_id        = "100"
  description    = "server VLAN"
  standard       = false
  layer3_net_ids = [idoit_layer3_net.servers.id]

  purge_on_destroy = true
}
