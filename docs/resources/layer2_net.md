---
page_title: "idoit_layer2_net Resource - terraform-provider-i-doit"
description: |-
  Manages a Layer-2 network (VLAN) in i-doit.
---

# idoit_layer2_net (Resource)

Manages a Layer-2 network (VLAN): a `C__OBJTYPE__LAYER2_NET` object together with
its `C__CATS__LAYER2_NET` category.

## Example Usage

```terraform
resource "idoit_layer2_net" "vlan100" {
  title          = "VLAN 100 - servers"
  vlan_id        = "100"
  description    = "server VLAN"
  standard       = false
  layer3_net_ids = [idoit_layer3_net.servers.id]
}
```

## Schema

### Required

- `title` (String) Object title.

### Optional

- `vlan_id` (String) VLAN id.
- `description` (String)
- `standard` (Boolean) Mark as the standard / native VLAN. Default `false`.
- `layer3_net_ids` (List of String) Object ids of the Layer-3 nets this VLAN is
  assigned to. Written to `C__CATS__LAYER2_NET` as `assigned_nets`; not
  drift-tracked. If your i-doit version uses a different key, set it via `extra`.
- `extra` (Map of String) Additional `C__CATS__LAYER2_NET` fields. Not
  drift-tracked.
- `purge_on_destroy` (Boolean) Purge instead of archive on destroy. Default
  `false`.

### Read-Only

- `id` (String) Numeric object id.
- `sysid` (String)

## Import

```shell
terraform import idoit_layer2_net.vlan100 1235
```
