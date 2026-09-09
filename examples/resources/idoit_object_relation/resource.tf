resource "idoit_object" "app" {
  type  = "C__OBJTYPE__APPLICATION"
  title = "billing-api"
}

resource "idoit_object" "server" {
  type  = "C__OBJTYPE__SERVER"
  title = "web01"
}

resource "idoit_object_relation" "app_runs_on_server" {
  master_object_id = idoit_object.app.id
  slave_object_id  = idoit_object.server.id
  relation_type    = "C__RELATION_TYPE__SOFTWARE"
  description      = "billing-api runs on web01"
}
