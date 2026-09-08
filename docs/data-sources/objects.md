---
page_title: "idoit_objects Data Source - terraform-provider-i-doit"
description: |-
  Lists CMDB objects filtered by type and/or title pattern.
---

# idoit_objects (Data Source)

Lists CMDB objects filtered by object type and/or a title pattern
(`cmdb.objects.read`).

## Example Usage

```terraform
data "idoit_objects" "servers" {
  type  = "C__OBJTYPE__SERVER"
  title = "web%"
  limit = 100
}

output "server_titles" {
  value = [for o in data.idoit_objects.servers.objects : o.title]
}
```

## Schema

### Optional

- `type` (String) Object type constant to filter by.
- `title` (String) Title filter. i-doit treats `%` as a wildcard.
- `limit` (Number) Maximum number of results. `0` or unset means no limit.

### Read-Only

- `id` (String) Synthetic identifier derived from the filter.
- `objects` (List of Object) Matched objects.

### Nested Schema for `objects`

Read-Only:

- `id` (Number)
- `title` (String)
- `sysid` (String)
- `type` (String) Object type constant.
- `type_title` (String) Localised object type label.
- `status` (String) Record status.
