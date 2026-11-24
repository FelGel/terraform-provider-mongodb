---
page_title: "mongodb_db_role List Resource - mongodb"
subcategory: ""
description: |-
  Lists all custom MongoDB roles across databases. Use with terraform query (Terraform 1.14 and later).
---

# mongodb_db_role (List Resource)

Lists all custom MongoDB roles across databases. Use with `terraform query` (Terraform 1.14 and later).

## Example Usage

```terraform
list "mongodb_db_role" "all" {
  provider = mongodb
}
```

Then run:

```sh
terraform query
```

## Schema

This list resource takes no configuration arguments. It returns every existing `mongodb_db_role`; each result uses the [`mongodb_db_role`](../resources/db_role.md) schema, and its identity is the base64-encoded `id` (`database.roleName`).
