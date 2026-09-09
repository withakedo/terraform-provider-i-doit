---
page_title: "idoit_layer3_net Resource - terraform-provider-i-doit"
description: |-
  Manages a Layer-3 network in i-doit.
---

# idoit_layer3_net (Resource)

Manages a Layer-3 network: a `C__OBJTYPE__LAYER3_NET` object together with its
`C__CATS__NET` (Net) category, in a single resource.

## Example Usage

```terraform
resource "idoit_layer3_net" "servers" {
  title       = "10.0.0.0/24 - servers"
  type        = "ipv4"
  address     = "10.0.0.0"
  cidr_suffix = "24"

  dns_server      = "10.0.0.1"
  dns_domain      = "example.internal"
  default_gateway = "10.0.0.1"
  range_from      = "10.0.0.10"
  range_to        = "10.0.0.250"
}
```

## Schema

### Required

- `title` (String) Object title.
- `address` (String) Network address, e.g. `10.0.0.0`.
- `cidr_suffix` (String) CIDR suffix, e.g. `24`.

### Optional

- `type` (String) `ipv4` (default) or `ipv6`.
- `dns_server` (String)
- `dns_domain` (String)
- `default_gateway` (String)
- `range_from` (String) Start of the usable / DHCP range.
- `range_to` (String) End of the usable / DHCP range.
- `description` (String) Free-text description stored on the net category.
- `extra` (Map of String) Additional `C__CATS__NET` fields passed verbatim to
  `cmdb.category.save`. Not drift-tracked; use for keys that differ between
  i-doit versions.
- `purge_on_destroy` (Boolean) Purge instead of archive on destroy. Default
  `false`.

### Read-Only

- `id` (String) Numeric object id.
- `sysid` (String)

## Import

```shell
terraform import idoit_layer3_net.servers 1234
```
