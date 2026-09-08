---
page_title: "idoit_category_entry Resource - terraform-provider-i-doit"
description: |-
  Manages one entry of an i-doit category attached to an object.
---

# idoit_category_entry (Resource)

Manages one entry of an i-doit category (global `C__CATG__*` or specific
`C__CATS__*`) attached to an object. Every instance maps to exactly one category
entry, so multi-value categories are modelled with multiple resources.

## Example Usage

```terraform
resource "idoit_object" "web01" {
  type  = "C__OBJTYPE__SERVER"
  title = "web01"
}

resource "idoit_category_entry" "web01_model" {
  object_id = idoit_object.web01.id
  category  = "C__CATG__MODEL"

  data = {
    manufacturer = "Dell"
    model        = "PowerEdge R660"
    serial       = "ABC123"
  }
}

resource "idoit_category_entry" "web01_ip" {
  object_id = idoit_object.web01.id
  category  = "C__CATG__IP"

  data = {
    ipv4_address = "10.0.0.11"
    hostname     = "web01"
    primary      = "1"
  }
}
```

## Schema

### Required

- `object_id` (String) Numeric id of the object the entry belongs to (accepts
  the `id` of an `idoit_object` resource). Changing this forces a new entry.
- `category` (String) Category constant, e.g. `C__CATG__IP` or `C__CATG__MODEL`.
  Changing this forces a new entry.
- `data` (Map of String) Category field values keyed by their attribute name.
  Values are sent as-is to `cmdb.category.save`. For dialog attributes provide
  the value title.

### Read-Only

- `id` (String) Synthetic identifier `<object_id>/<category>/<entry_id>`.
- `entry_id` (Number) Numeric id of the category entry as assigned by i-doit.

## Import

```shell
terraform import idoit_category_entry.web01_model 42/C__CATG__MODEL/17
```
