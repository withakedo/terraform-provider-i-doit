resource "idoit_object" "web01" {
  type  = "C__OBJTYPE__SERVER"
  title = "web01"

  # Optional: set the CMDB status (constant or numeric id).
  # cmdb_status = "C__CMDB_STATUS__IN_OPERATION"

  # Optional: clone a template object on create.
  # template_id = 4321

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
