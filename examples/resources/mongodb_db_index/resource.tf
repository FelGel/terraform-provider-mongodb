resource "mongodb_db_index" "example_index" {
  db         = "my_database"
  collection = "example"
  name       = "my_index"
  keys {
    field = "field_name_to_index2"
    value = "-1"
  }
  keys {
    field = "field_name_to_index"
    value = "1"
  }
  # index options go last, in the order unique, sparse, expireAfterSeconds
  keys {
    field = "unique"
    value = "true"
  }
  keys {
    field = "sparse"
    value = "true"
  }
  keys {
    field = "expireAfterSeconds"
    value = "86400"
  }
  timeout = 30
}
