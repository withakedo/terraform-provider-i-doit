# Terraform Provider for i-doit

[![ci](https://github.com/withakedo/terraform-provider-i-doit/actions/workflows/ci.yml/badge.svg)](https://github.com/withakedo/terraform-provider-i-doit/actions/workflows/ci.yml)
[![release](https://github.com/withakedo/terraform-provider-i-doit/actions/workflows/release.yml/badge.svg)](https://github.com/withakedo/terraform-provider-i-doit/actions/workflows/release.yml)

Manage [i-doit](https://www.i-doit.com/) CMDB objects and their category fields
declaratively through the i-doit JSON-RPC API.

The provider is built on the
[terraform-plugin-framework](https://developer.hashicorp.com/terraform/plugin/framework)
and ships a small, dependency-free JSON-RPC client (a single `POST` endpoint,
`<url>/src/jsonrpc.php`).

## Features

- `idoit_object` – create / read / update / archive (or purge) any CMDB object.
- `idoit_category_entry` – attach arbitrary global (`C__CATG__*`) or specific
  (`C__CATS__*`) category data to an object, one entry per resource, so
  multi-value categories work naturally.
- Typed network resources: `idoit_layer3_net` (`C__OBJTYPE__LAYER3_NET` +
  `C__CATS__NET`), `idoit_layer2_net` (VLAN, `C__OBJTYPE__LAYER2_NET` +
  `C__CATS__LAYER2_NET`) and `idoit_ip` (`C__CATG__IP` address assignment into a
  Layer-3 net). Each has typed fields for the common attributes plus an `extra`
  passthrough for version-specific keys.
- Data sources: `idoit_object`, `idoit_objects`, `idoit_object_type`,
  `idoit_layer3_net`.
- API-key auth plus optional session auth (`idoit.login` / `idoit.logout`).
- Configurable `request_timeout`, `max_retries` (exponential backoff),
  `max_concurrent_requests` and full TLS control (`insecure_skip_verify`,
  custom `ca_cert`, mutual-TLS `client_cert` / `client_key`, `tls_server_name`).
- Precise not-found handling: only a genuine "object does not exist" API error
  removes a resource from state; auth and server errors are surfaced.
- Every setting can come from an environment variable; all secrets are marked
  `sensitive`.
- `terraform import` for every resource.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.5
- [Go](https://go.dev/dl/) >= 1.23 (to build)
- An i-doit installation with the JSON-RPC API enabled and an API key
  (*Administration > Interfaces / external data > JSON-RPC API*).

## Build & install locally

```bash
go mod tidy          # resolves dependencies and writes go.sum
make build           # ./terraform-provider-i-doit
make install         # ~/.terraform.d/plugins/registry.terraform.io/withakedo/i-doit/<version>/<os_arch>/
```

Point Terraform at the local build with a `dev_overrides` block in
`~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/withakedo/i-doit" = "/absolute/path/to/repo"
  }
  direct {}
}
```

## Provider configuration

```hcl
terraform {
  required_providers {
    idoit = {
      source = "withakedo/i-doit"
    }
  }
}

provider "idoit" {
  url    = "https://cmdb.example.com" # without /src/jsonrpc.php
  apikey = var.idoit_apikey

  # optional session authentication
  # username = var.idoit_username
  # password = var.idoit_password

  request_timeout      = 60
  max_retries          = 3
  insecure_skip_verify = false
  language             = "en"
}
```

| Argument | Env var | Default | |
|---|---|---|---|
| `url` | `IDOIT_URL` | – | required |
| `apikey` | `IDOIT_APIKEY` | – | required, sensitive |
| `username` | `IDOIT_USERNAME` | – | sensitive |
| `password` | `IDOIT_PASSWORD` | – | sensitive |
| `request_timeout` | – | `60` | seconds, per request |
| `max_retries` | – | `3` | retries on network / 429 / 5xx |
| `max_concurrent_requests` | – | `10` | in-flight request cap; `0` = unlimited |
| `insecure_skip_verify` | – | `false` | |
| `ca_cert` | `IDOIT_CA_CERT` | – | PEM data or path to a PEM file |
| `client_cert` | `IDOIT_CLIENT_CERT` | – | mutual TLS; PEM data or path |
| `client_key` | `IDOIT_CLIENT_KEY` | – | sensitive; PEM data or path |
| `tls_server_name` | `IDOIT_TLS_SERVER_NAME` | – | SNI / cert host override |
| `language` | `IDOIT_LANGUAGE` | `en` | |

On configure the provider calls `idoit.version` to verify reachability and
authentication.

## Usage

```hcl
resource "idoit_object" "web01" {
  type  = "C__OBJTYPE__SERVER"
  title = "web01"

  # destroy => cmdb.object.archive by default; set true for cmdb.object.purge
  purge_on_destroy = false
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
    ipv4_address  = "10.0.0.11"
    hostname      = "web01"
    primary       = "1"
  }
}
```

See [`examples/`](./examples) for more, including data sources.

### Networks (L2 / L3)

```hcl
resource "idoit_layer3_net" "servers" {
  title       = "10.0.0.0/24 - servers"
  type        = "ipv4"
  address     = "10.0.0.0"
  cidr_suffix = "24"
  dns_server  = "10.0.0.1"
  range_from  = "10.0.0.10"
  range_to    = "10.0.0.250"
}

resource "idoit_layer2_net" "vlan100" {
  title          = "VLAN 100 - servers"
  vlan_id        = "100"
  layer3_net_ids = [idoit_layer3_net.servers.id]
}

resource "idoit_ip" "web01" {
  object_id    = idoit_object.web01.id
  net_id       = idoit_layer3_net.servers.id
  ipv4_address = "10.0.0.11"
  hostname     = "web01"
  primary      = true
}
```

The typed fields cover the common attributes; the exact `C__CATS__NET` /
`C__CATS__LAYER2_NET` / `C__CATG__IP` keys vary between i-doit versions, so each
net resource also takes an `extra = { ... }` map that is merged into the category
payload untouched, and everything remains reachable through the generic
`idoit_category_entry`.

### Importing

```bash
terraform import idoit_object.web01 42
terraform import idoit_category_entry.web01_model 42/C__CATG__MODEL/17
terraform import idoit_layer3_net.servers 1234
terraform import idoit_layer2_net.vlan100 1235
terraform import idoit_ip.web01 42/17
```

## Category `data`

`data` is a `map(string)` sent verbatim to `cmdb.category.save`. Values are read
back with `cmdb.category.read` and reduced to scalars for drift detection:
dialog / dialog+ attributes come back as objects and are collapsed to their
`title`, so configure those with the **value title**. Fields that cannot be
represented as a simple string are left untouched in state. Attribute keys and
their accepted values are listed in i-doit under *Administration > CMDB >
Attribute documentation*.

## Testing

Unit tests:

```bash
make test
```

Acceptance tests create and delete real objects and are skipped unless
`TF_ACC=1`:

```bash
export IDOIT_URL="https://demo.i-doit.com"
export IDOIT_APIKEY="<demo api key>"
export TF_ACC=1
make testacc
```

The bundled acceptance tests target the public demo instance at
`https://demo.i-doit.com/src/jsonrpc.php`.

### Continuous integration

`.github/workflows/ci.yml` runs `go build`, `go vet`, `go test` and
`staticcheck` on every push and pull request. On pushes to `main` it also
regenerates `go.sum` and re-applies `gofmt`, committing the result, so the
module stays reproducible without a `go.sum` maintained by hand.

Lint locally with either:

```bash
make staticcheck        # go run honnef.co/go/tools/cmd/staticcheck@latest ./...
make lint               # golangci-lint run (install golangci-lint separately)
```

## Documentation

`docs/` is generated with
[`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs):

```bash
make docs
```

## Releasing

Releases are cut by the `release` GitHub Actions workflow
(`.github/workflows/release.yml`) when a `v*` tag is pushed. It runs
[GoReleaser](https://goreleaser.com/) to build the cross-platform archives,
writes `SHA256SUMS`, GPG-signs them and the registry manifest, and publishes a
GitHub Release in the Terraform Registry layout.

Required repository secrets:

| Secret | Purpose |
|---|---|
| `GPG_PRIVATE_KEY` | ASCII-armored private signing key |
| `PASSPHRASE` | passphrase for that key |

Cut a release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

Local dry run (no signing):

```bash
goreleaser release --clean --snapshot
```

## License

[MPL-2.0](./LICENSE)
