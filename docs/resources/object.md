---
page_title: "idoit_object Resource - terraform-provider-i-doit"
description: |-
  Manages a CMDB object in i-doit.
---

# idoit_object (Resource)

Manages a CMDB object in i-doit. An object is identified by its object type
constant and a title; category fields are managed separately with
[`idoit_category_entry`](./category_entry.md).

## Example Usage

```terraform
resource "idoit_object" "web01" {
  type  = "C__OBJTYPE__SERVER"
  title = "web01"

  purge_on_destroy = false
}
```

## Schema

### Required

- `type` (String) Object type constant, e.g. `C__OBJTYPE__SERVER`. Changing
  this forces a new object.
- `title` (String) Object title.

### Optional

- `purge_on_destroy` (Boolean) When `true`, `terraform destroy` calls
  `cmdb.object.purge` (irreversible). When `false` (default) it calls
  `cmdb.object.archive`, which can be restored in i-doit.

### Read-Only

- `id` (String) Numeric i-doit object identifier.
- `sysid` (String) SYSID assigned by i-doit.
- `status` (String) Current CMDB status label (e.g. `in operation`).

## Import

```shell
terraform import idoit_object.web01 42
```

After import, set `type` in the configuration to the object's type constant.
