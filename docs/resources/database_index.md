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
    field = "sparse"
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
* `keys` - (Required) Field and value pairs where the field is the index key and the value describes the type of index for that field.
                      For an ascending index on a field, specify a value of 1. For descending index, specify a value of -1.
                      A handful of index *options* are also expressed as `keys` entries (rather than dedicated attributes):
                      `unique = "true"`, `sparse = "true"`, and `expireAfterSeconds = "<n>"` (TTL). These are read back
                      as `keys` entries too, appended after the real key fields in the order unique, sparse, expireAfterSeconds —
                      list them last, in that order, to avoid a plan diff.
                      See https://www.mongodb.com/docs/manual/reference/method/db.collection.createIndex/ for details
* `name` - (Optional) Index name
* `partial_filter_expression` - (Optional) A JSON string representing the partialFilterExpression for a partial index. Use `jsonencode()` for readability. See https://www.mongodb.com/docs/manual/core/index-partial/ for details
* `hidden` - (Optional, default: false) If true, the index is hidden from the query planner (MongoDB 4.4+). Can be toggled in-place without recreating the index. Useful for evaluating index removal safety. See https://www.mongodb.com/docs/manual/core/index-hidden/
* `timeout` - (Optional) Timeout for index creation operation


## Changing an existing index

MongoDB indexes are largely immutable: the server has no "alter index" for the
key spec or most options. This resource reflects that — only `hidden` can be
changed in place. **Every other change** forces replacement: the `keys` block
(including the `unique`, `sparse`, and `expireAfterSeconds` pseudo-entries),
`name`, `partial_filter_expression`, `db`, and `collection` are all marked
`RequiresReplace`. Terraform migrates such a change by **dropping the old index
and creating the new one** in the same apply.

Migrating between index variants (e.g. non-unique → unique, adding `sparse`,
regular → TTL, full → partial) therefore means editing the config and applying;
`terraform plan` will show the index being destroyed and recreated:

```hcl
# before                         # after (forces replacement)
keys {                           keys {
  field = "email"                  field = "email"
  value = "1"                      value = "1"
}                                }
                                 keys {
                                   field = "unique"
                                   value = "true"
                                 }
```

Practical notes:

* The drop-and-recreate is not atomic — there is a brief window with no index.
  For large collections the rebuild can be expensive; plan the apply
  accordingly and raise `timeout` if index creation is slow.
* To evaluate the impact of *removing* an index before actually dropping it,
  set `hidden = true` first (an in-place change), observe query performance,
  then remove the resource.
* Changing only `hidden` never recreates the index.

## Import

Mongodb indexes can be imported using the hex encoded id, e.g. for a collection named `collection_test`, his database id `test_db` and collection name `example_index`:

```sh
$ printf '%s' "test_db.collection_test.example_index" | base64
## this is the output of the command above it will encode db.collection.index to HEX 
dGVzdF9kYi5jb2xsZWN0aW9uX3Rlc3QuZXhhbXBsZV9pbmRleA==

$ terraform import mongodb_db_index.example_index  dGVzdF9kYi5jb2xsZWN0aW9uX3Rlc3QuZXhhbXBsZV9pbmRleA==
```
