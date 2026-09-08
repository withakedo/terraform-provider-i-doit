# Terraform Provider for i-doit

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
- Data sources: `idoit_object`, `idoit_objects`, `idoit_object_type`.
- API-key auth plus optional session auth (`idoit.login` / `idoit.logout`).
- Configurable `request_timeout`, `max_retries` (exponential backoff) and
  `insecure_skip_verify`.
- Every setting can come from an environment variable; all secrets are marked
  `sensitive`.
- `terraform import` for both resources.

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
| `insecure_skip_verify` | – | `false` | |
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

### Importing

```bash
terraform import idoit_object.web01 42
terraform import idoit_category_entry.web01_model 42/C__CATG__MODEL/17
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

## Documentation

`docs/` is generated with
[`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs):

```bash
make docs
```

## Releasing

There is no CI. Produce local artifacts by hand with
[GoReleaser](https://goreleaser.com/):

```bash
export GPG_FINGERPRINT=<your key>
goreleaser release --clean
```

## License

[MPL-2.0](./LICENSE)
