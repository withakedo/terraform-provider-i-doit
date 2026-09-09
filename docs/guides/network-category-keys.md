---
page_title: "Network category attribute keys"
subcategory: "Networks"
description: |-
  Reference for the C__CATS__NET, C__CATS__LAYER2_NET and C__CATG__IP attribute
  keys used by the typed network resources, and how to verify them against a
  specific i-doit installation.
---

# Network category attribute keys

The typed network resources (`idoit_layer3_net`, `idoit_layer2_net`, `idoit_ip`)
write their fields to i-doit categories through `cmdb.category.save`. The
attribute keys below are what the provider sends. They are derived from the
public [i-doit API client](https://github.com/i-doit/api-client-php) examples and
the i-doit attribute model, **not** from a running instance — i-doit installs
differ by version, add-on set and custom attributes, so treat this page as a
starting point and confirm against your own CMDB.

Anything not covered by a typed field is still reachable: every network resource
accepts an `extra = { ... }` map that is merged verbatim into the category
payload, and the generic `idoit_category_entry` resource can read and write any
category attribute by key.

## `C__CATS__NET` — Layer-3 net (`idoit_layer3_net`)

| Resource field | Category key      | Type in i-doit         | Notes |
|----------------|------------------|------------------------|-------|
| `type`         | `type`           | dialog                 | `C__CATS_NET_TYPE__IPV4` / `C__CATS_NET_TYPE__IPV6`; the provider maps `ipv4` / `ipv6`. |
| `address`      | `address`        | text                   | Network address, e.g. `10.0.0.0`. |
| `cidr_suffix`  | `cidr_suffix`    | text                   | Prefix length as a string, e.g. `24`. |
| `dns_server`   | `dns_server`     | text / object ref      | Some versions expect a DNS server object id. |
| `dns_domain`   | `dns_domain`     | text / object ref      | |
| `default_gateway` | `default_gateway` | text / object ref  | |
| `range_from`   | `range_from`     | text                   | First address of the managed/DHCP range. |
| `range_to`     | `range_to`       | text                   | Last address of the managed/DHCP range. |
| `description`  | `description`    | text                   | |

## `C__CATS__LAYER2_NET` — Layer-2 net / VLAN (`idoit_layer2_net`)

| Resource field   | Category key     | Type in i-doit | Notes |
|------------------|-----------------|----------------|-------|
| `vlan_id`        | `vlan_id`       | text           | VLAN number as a string. |
| `description`    | `description`   | text           | |
| `standard`       | `standard`      | yes/no         | Sent as `"1"` / `"0"`. |
| `layer3_net_ids` | `assigned_nets` | object refs    | List of Layer-3 net object ids. The key varies between versions; override through `extra` if your instance rejects `assigned_nets`. |

## `C__CATG__IP` — IP address assignment (`idoit_ip`)

| Resource field | Category key     | Type in i-doit | Notes |
|----------------|-----------------|----------------|-------|
| `ipv4_address` | `ipv4_address`  | text           | |
| `net_id`       | `net`           | object ref     | Object id of the Layer-3 net. |
| `hostname`     | `hostname`      | text           | |
| `primary`      | `primary`       | yes/no         | Sent as `"1"` / `"0"`. |
| `active`       | `active`        | yes/no         | Sent as `"1"` / `"0"`. |

## Verifying keys against your instance

1. **Attribute documentation UI.** In i-doit, open
   *Administration > CMDB > Attribute documentation* (or *Attribute host*), pick
   the category and read the property key column.

2. **Round-trip with `idoit_category_entry`.** Create one entry by hand in the
   i-doit web UI, import it, and inspect the state:

   ```bash
   terraform import idoit_category_entry.probe 42/C__CATS__NET/1
   terraform state show idoit_category_entry.probe
   ```

   The `data` map shows every key i-doit returned for that entry.

3. **Raw API call.** `cmdb.category.read` with `objID` and `category` returns the
   full attribute set, including the exact keys and their value shapes (scalar
   vs. `{ id, title, const }` object for dialog and reference fields).

If a key differs, set it through `extra`:

```hcl
resource "idoit_layer3_net" "servers" {
  title       = "10.0.0.0/24"
  address     = "10.0.0.0"
  cidr_suffix = "24"

  extra = {
    # your instance calls the DNS server field differently
    dns_server_address = "10.0.0.1"
  }
}
```
