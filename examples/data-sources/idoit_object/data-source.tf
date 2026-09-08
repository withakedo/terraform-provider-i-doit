# Look up by numeric id.
data "idoit_object" "by_id" {
  id = 42
}

# Look up by title, narrowed by object type.
data "idoit_object" "by_title" {
  title = "web01"
  type  = "C__OBJTYPE__SERVER"
}

output "web01_sysid" {
  value = data.idoit_object.by_title.sysid
}
