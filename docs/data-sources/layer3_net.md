---
page_title: "idoit_layer3_net Data Source - terraform-provider-i-doit"
description: |-
  Looks up a Layer-3 network in i-doit.
---

# idoit_layer3_net (Data Source)

Looks up a Layer-3 net by numeric `id` or by `title` and returns its
`C__CATS__NET` fields. Provide exactly one of `id` or `title`.

## Example Usage

```terraform
data "idoit_layer3_net" "servers" {
  title = "10.0.0.0/24 - servers"
}
```

## Schema

### Optional

- `id` (Number) Numeric object id.
- `title` (String) Object title.

### Read-Only

- `type` (String) `ipv4` or `ipv6`.
- `address` (String)
- `cidr_suffix` (String)
- `dns_server` (String)
- `dns_domain` (String)
- `range_from` (String)
- `range_to` (String)
- `sysid` (String)
```
