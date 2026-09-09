data "idoit_layer3_net" "servers" {
  title = "10.0.0.0/24 - servers"
}

output "servers_cidr" {
  value = "${data.idoit_layer3_net.servers.address}/${data.idoit_layer3_net.servers.cidr_suffix}"
}
