---
page_title: "idoit_object_relation Resource - terraform-provider-i-doit"
description: |-
  Manages a relation between two CMDB objects.
---

# idoit_object_relation (Resource)

Manages a relation between two CMDB objects through the `C__CATG__RELATION`
category on the master object. Created, read and deleted with
`cmdb.category.save` / `cmdb.category.read` / `cmdb.category.delete`.

## Example Usage

```terraform
resource "idoit_object" "app" {
  type  = "C__OBJTYPE__APPLICATION"
  title = "billing-api"
}

resource "idoit_object" "server" {
  type  = "C__OBJTYPE__SERVER"
  title = "web01"
}

resource "idoit_object_relation" "app_runs_on_server" {
  master_object_id = idoit_object.app.id
  slave_object_id  = idoit_object.server.id
  relation_type    = "C__RELATION_TYPE__SOFTWARE"
  description      = "billing-api runs on web01"
}
```

## Schema

### Required

- `master_object_id` (String) Object id on the master side. Forces a new relation.
- `slave_object_id` (String) Object id on the slave side. Forces a new relation.
- `relation_type` (String) Relation type constant (e.g.
  `C__RELATION_TYPE__SOFTWARE`) or numeric id. Forces a new relation.

### Optional

- `description` (String) Free-text description of the relation.
- `weighting` (String) Optional weighting value.
- `extra` (Map of String) Additional `C__CATG__RELATION` fields passed verbatim
  to `cmdb.category.save`. Not drift-tracked.

### Read-Only

- `id` (String) `<master_object_id>/<entry_id>`.
- `entry_id` (Number) Numeric id of the relation category entry.

## Import

```shell
terraform import idoit_object_relation.app_runs_on_server 1234/17
```
