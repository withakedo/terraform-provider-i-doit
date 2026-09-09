---
page_title: "idoit_ip Resource - terraform-provider-i-doit"
description: |-
  Manages an IPv4 address assignment on an i-doit object.
---

# idoit_ip (Resource)

Manages one `C__CATG__IP` category entry on an existing object, optionally
placing the address into a Layer-3 net. Multi-value: use one resource per
address.

## Example Usage

```terraform
resource "idoit_ip" "web01_primary" {
  object_id    = idoit_object.web01.id
  net_id       = idoit_layer3_net.servers.id
  ipv4_address = "10.0.0.11"
  hostname     = "web01"
  primary      = true
  active       = true
}
```

## Schema

### Required

- `object_id` (String) Numeric id of the object the address belongs to. Changing
  this forces a new entry.
- `ipv4_address` (String) IPv4 address.

### Optional

- `net_id` (String) Object id of the Layer-3 net the address belongs to.
- `hostname` (String)
- `primary` (Boolean) Mark as the object's primary address. Default `false`.
- `active` (Boolean) Whether the address is active. Default `true`.

### Read-Only

- `id` (String) Synthetic identifier `<object_id>/<entry_id>`.
- `entry_id` (Number) Numeric id of the category entry.

## Import

```shell
terraform import idoit_ip.web01_primary 4242/17
```
