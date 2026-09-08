resource "idoit_object" "web01" {
  type  = "C__OBJTYPE__SERVER"
  title = "web01"
}

# Single-value category: hardware model.
resource "idoit_category_entry" "web01_model" {
  object_id = idoit_object.web01.id
  category  = "C__CATG__MODEL"

  data = {
    manufacturer = "Dell"
    model        = "PowerEdge R660"
    serial       = "ABC123"
  }
}

# Multi-value category: one resource per IP entry.
resource "idoit_category_entry" "web01_ip_primary" {
  object_id = idoit_object.web01.id
  category  = "C__CATG__IP"

  data = {
    ipv4_address = "10.0.0.11"
    hostname     = "web01"
    primary      = "1"
  }
}

resource "idoit_category_entry" "web01_ip_backup" {
  object_id = idoit_object.web01.id
  category  = "C__CATG__IP"

  data = {
    ipv4_address = "10.0.9.11"
    hostname     = "web01-bkp"
    primary      = "0"
  }
}
