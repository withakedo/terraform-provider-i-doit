# Resolve a constant to its numeric id (or the reverse with `id`, or by `title`).
data "idoit_object_type" "server" {
  const = "C__OBJTYPE__SERVER"
}

output "server_type_id" {
  value = data.idoit_object_type.server.id
}
