resource "mongodb_db_index" "partial_index" {
  db         = "my_database"
  collection = "example"
  name       = "my_partial_index"
  keys {
    field = "field_a"
    value = "1"
  }
  keys {
    field = "field_b"
    value = "1"
  }
  keys {
    field = "field_c"
    value = "1"
  }
  partial_filter_expression = jsonencode({
    "field_a" = { "$exists" = true }
  })
  timeout = 30
}
