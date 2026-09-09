resource "idoit_layer3_net" "servers" {
  title       = "10.0.0.0/24 - servers"
  address     = "10.0.0.0"
  cidr_suffix = "24"
}

resource "idoit_object" "web01" {
  type  = "C__OBJTYPE__SERVER"
  title = "web01"
}

resource "idoit_ip" "web01_primary" {
  object_id    = idoit_object.web01.id
  net_id       = idoit_layer3_net.servers.id
  ipv4_address = "10.0.0.11"
  hostname     = "web01"
  primary      = true
  active       = true
}
