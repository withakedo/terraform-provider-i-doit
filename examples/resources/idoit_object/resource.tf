resource "idoit_object" "web01" {
  type  = "C__OBJTYPE__SERVER"
  title = "web01"

  # By default `terraform destroy` archives the object (reversible).
  # Set this to permanently purge it instead.
  purge_on_destroy = false
}

output "web01_id" {
  value = idoit_object.web01.id
}

output "web01_sysid" {
  value = idoit_object.web01.sysid
}
