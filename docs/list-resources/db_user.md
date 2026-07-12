# mongodb_db_user (List Resource)

Lists all MongoDB database users. Use with `terraform query` (Terraform 1.14 and later) to enumerate existing users across all databases.

## Example Usage

```hcl
list "mongodb_db_user" "all" {
  provider = mongodb
}
```

Then run:

```sh
terraform query
```

## Schema

This list resource takes no configuration arguments — it returns every `mongodb_db_user`. Each returned resource uses the [`mongodb_db_user`](../resources/database_user.md) schema; its identity is the base64-encoded `id` (`auth_database.username`).
