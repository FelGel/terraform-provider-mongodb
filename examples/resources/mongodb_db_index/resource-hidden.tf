resource "mongodb_db_index" "hidden_index" {
  db         = "my_database"
  collection = "example"
  name       = "my_hidden_index"
  keys {
    field = "field_x"
    value = "1"
  }
  keys {
    field = "field_y"
    value = "1"
  }
  hidden  = true
  timeout = 30
}
