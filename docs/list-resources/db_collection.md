# mongodb_db_collection (List Resource)

Lists all MongoDB collections across databases. Use with `terraform query` (Terraform 1.14 and later) to enumerate existing collections.

## Example Usage

```hcl
list "mongodb_db_collection" "all" {
  provider = mongodb
}
```

Then run:

```sh
terraform query
```

## Schema

This list resource takes no configuration arguments — it returns every `mongodb_db_collection`. Each returned resource uses the [`mongodb_db_collection`](../resources/database_collection.md) schema; its identity is the base64-encoded `id` (`db.collectionName`).
