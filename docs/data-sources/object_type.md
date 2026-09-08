---
page_title: "idoit_object_type Data Source - terraform-provider-i-doit"
description: |-
  Resolves an object type between its constant, numeric id and title.
---

# idoit_object_type (Data Source)

Resolves an object type between its constant, numeric id and title
(`cmdb.object_types`). Provide exactly one of `const`, `id` or `title`.

## Example Usage

```terraform
data "idoit_object_type" "server" {
  const = "C__OBJTYPE__SERVER"
}

output "server_type_id" {
  value = data.idoit_object_type.server.id
}
```

## Schema

### Optional

- `const` (String) Object type constant, e.g. `C__OBJTYPE__SERVER`.
- `id` (Number) Numeric object type id.
- `title` (String) Object type title.

### Read-Only

- `title_lang` (String) Localised object type title.
- `object_count` (Number) Number of objects of this type.
