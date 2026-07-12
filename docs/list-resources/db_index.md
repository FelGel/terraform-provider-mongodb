# mongodb_db_index (List Resource)

Lists all MongoDB indexes across databases and collections. Use with `terraform query` (Terraform 1.14 and later) to enumerate existing indexes.

## Example Usage

```hcl
list "mongodb_db_index" "all" {
  provider = mongodb
}
```

Then run:

```sh
terraform query
```

## Schema

This list resource takes no configuration arguments — it returns every `mongodb_db_index`. Each returned resource uses the [`mongodb_db_index`](../resources/database_index.md) schema; its identity is the base64-encoded `id` (`db.collection.indexName`).
