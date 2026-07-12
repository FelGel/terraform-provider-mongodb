# mongodb_db_role (List Resource)

Lists all custom MongoDB roles across databases. Use with `terraform query` (Terraform 1.14 and later) to enumerate existing roles.

## Example Usage

```hcl
list "mongodb_db_role" "all" {
  provider = mongodb
}
```

Then run:

```sh
terraform query
```

## Schema

This list resource takes no configuration arguments — it returns every custom `mongodb_db_role`. Each returned resource uses the [`mongodb_db_role`](../resources/database_role.md) schema; its identity is the base64-encoded `id` (`database.roleName`).
