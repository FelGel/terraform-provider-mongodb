# Mongo Database Index

Provides a Database Index resource.

## Example Usages

##### - create index

```hcl
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
  keys {
    field = "unique"
    value = "true"
  }  
  keys {
    field = "expireAfterSeconds"
    value = "86400"
  }
  timeout = 30
}
```

##### - create partial index

```hcl
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
```

##### - create hidden index

```hcl
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
```

## Argument Reference
* `db` - (Required) Database in which the target collection resides
* `collection` - (Required) Collection name
* `keys` - (Required) Field and value pairs where the field is the index key and the value describes the type of index for that field
                      For an ascending index on a field, specify a value of 1. For descending index, specify a value of -1
                      See https://www.mongodb.com/docs/manual/reference/method/db.collection.createIndex/ for details
* `name` - (Optional) Index name
* `partial_filter_expression` - (Optional) A JSON string representing the partialFilterExpression for a partial index. Use `jsonencode()` for readability. See https://www.mongodb.com/docs/manual/core/index-partial/ for details
* `hidden` - (Optional, default: false) If true, the index is hidden from the query planner (MongoDB 4.4+). Can be toggled in-place without recreating the index. Useful for evaluating index removal safety. See https://www.mongodb.com/docs/manual/core/index-hidden/
* `timeout` - (Optional) Timeout for index creation operation


## Import

Mongodb indexes can be imported using the hex encoded id, e.g. for a collection named `collection_test`, his database id `test_db` and collection name `example_index`:

```sh
$ printf '%s' "test_db.collection_test.example_index" | base64
## this is the output of the command above it will encode db.collection.index to HEX 
dGVzdF9kYi5jb2xsZWN0aW9uX3Rlc3QuZXhhbXBsZV9pbmRleA==

$ terraform import mongodb_db_index.example_index  dGVzdF9kYi5jb2xsZWN0aW9uX3Rlc3QuZXhhbXBsZV9pbmRleA==
```
