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

  # Optional session authentication (idoit.login / idoit.logout):
  # username = var.idoit_username
  # password = var.idoit_password

  request_timeout         = 60
  max_retries             = 3
  max_concurrent_requests = 10
  insecure_skip_verify    = false
  language                = "en"

  # Custom TLS trust / mutual TLS (inline PEM or a path to a PEM file):
  # ca_cert         = file("~/certs/idoit-ca.pem")
  # client_cert     = file("~/certs/client.pem")
  # client_key      = file("~/certs/client-key.pem")
  # tls_server_name = "cmdb.internal"
}

variable "idoit_apikey" {
  type      = string
  sensitive = true
}
