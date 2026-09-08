data "idoit_objects" "servers" {
  type  = "C__OBJTYPE__SERVER"
  title = "web%" # i-doit wildcard
  limit = 100
}

output "server_titles" {
  value = [for o in data.idoit_objects.servers.objects : o.title]
}
