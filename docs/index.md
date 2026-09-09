---
page_title: "i-doit Provider"
description: |-
  Manage i-doit CMDB objects and their category fields through the i-doit JSON-RPC API.
---

# i-doit Provider

The i-doit provider manages CMDB objects and their category fields declaratively
through the i-doit JSON-RPC API (`<url>/src/jsonrpc.php`).

## Example Usage

```terraform
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

  request_timeout      = 60
  max_retries          = 3
  insecure_skip_verify = false
  language             = "en"
}

variable "idoit_apikey" {
  type      = string
  sensitive = true
}
```

## Authentication

An API key is always required. Session authentication (`idoit.login` /
`idoit.logout`) is used additionally when both `username` and `password` are
set. On configure the provider calls `idoit.version` to verify that the
endpoint is reachable and the credentials are accepted.

Every argument can also be supplied through an environment variable.

## Schema

### Optional

- `url` (String) Base URL of the i-doit installation, without the
  `/src/jsonrpc.php` suffix. Env: `IDOIT_URL`.
- `apikey` (String, Sensitive) JSON-RPC API key. Env: `IDOIT_APIKEY`.
- `username` (String, Sensitive) Username for session authentication. Env:
  `IDOIT_USERNAME`.
- `password` (String, Sensitive) Password for session authentication. Env:
  `IDOIT_PASSWORD`.
- `request_timeout` (Number) Per-request timeout in seconds. Defaults to `60`.
- `max_retries` (Number) Retries with exponential backoff for transient
  failures (network errors, HTTP 429/5xx). Defaults to `3`.
- `max_concurrent_requests` (Number) Cap on in-flight HTTP requests against the
  API. `0` means unlimited. Defaults to `10`.
- `insecure_skip_verify` (Boolean) Disable TLS certificate verification.
  Defaults to `false`.
- `ca_cert` (String) Custom CA certificate(s) to trust, as inline PEM data or a
  path to a PEM file. Env: `IDOIT_CA_CERT`.
- `client_cert` (String) Client certificate for mutual TLS, as inline PEM data
  or a path to a PEM file. Requires `client_key`. Env: `IDOIT_CLIENT_CERT`.
- `client_key` (String, Sensitive) Private key for `client_cert`. Env:
  `IDOIT_CLIENT_KEY`.
- `tls_server_name` (String) Override the server name used for SNI and
  certificate verification. Env: `IDOIT_TLS_SERVER_NAME`.
- `language` (String) API language. Defaults to `en`. Env: `IDOIT_LANGUAGE`.
