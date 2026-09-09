---
page_title: "idoit_dialog_value Resource - terraform-provider-i-doit"
description: |-
  Manages a selectable value of a dialog / dialog+ attribute.
---

# idoit_dialog_value (Resource)

Manages a single selectable value of a dialog / dialog+ (drop-down) attribute
via `cmdb.dialog.create` / `cmdb.dialog.read` / `cmdb.dialog.update` /
`cmdb.dialog.delete`.

~> **Note** Many built-in i-doit dialogs are system-managed and cannot be
changed through the API. `cmdb.dialog.*` will return an error for those.

## Example Usage

```terraform
resource "idoit_dialog_value" "manufacturer_acme" {
  category = "C__CATG__MODEL"
  property = "manufacturer"
  value    = "ACME Corp"
}

# hierarchical dialog+ child value
resource "idoit_dialog_value" "acme_model_x" {
  category  = "C__CATG__MODEL"
  property  = "title"
  value     = "Model X"
  parent_id = idoit_dialog_value.manufacturer_acme.entry_id
}
```

## Schema

### Required

- `category` (String) Category constant that owns the attribute, e.g.
  `C__CATG__MODEL`. Forces a new value.
- `property` (String) Attribute key within the category, e.g. `manufacturer`.
  Forces a new value.
- `value` (String) Display text of the dialog value.

### Optional

- `parent_id` (Number) Parent value id for a hierarchical dialog+ attribute.
  Forces a new value.

### Read-Only

- `id` (String) `<category>/<property>/<entry_id>`.
- `entry_id` (Number) Numeric id of the dialog value.
- `const` (String) Constant assigned by i-doit, if any.

## Import

```shell
terraform import idoit_dialog_value.manufacturer_acme C__CATG__MODEL/manufacturer/42
```
