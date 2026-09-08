---
page_title: "idoit_object Data Source - terraform-provider-i-doit"
description: |-
  Looks up a single CMDB object by id or by title.
---

# idoit_object (Data Source)

Looks up a single CMDB object either by its numeric `id` or by `title`
(optionally narrowed with `type`). Provide exactly one of `id` or `title`.

## Example Usage

```terraform
data "idoit_object" "by_title" {
  title = "web01"
  type  = "C__OBJTYPE__SERVER"
}

output "web01_sysid" {
  value = data.idoit_object.by_title.sysid
}
```

## Schema

### Optional

- `id` (Number) Numeric object id.
- `title` (String) Object title.
- `type` (String) Object type constant used to disambiguate a `title` lookup.

### Read-Only

- `sysid` (String) SYSID of the object.
- `type_title` (String) Localised object type label.
- `status` (String) CMDB status label.
