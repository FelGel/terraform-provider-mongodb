---
page_title: "mongodb_db_user List Resource - mongodb"
subcategory: ""
description: |-
  Lists all MongoDB database users. Use with terraform query (Terraform 1.14 and later).
---

# mongodb_db_user (List Resource)

Lists all MongoDB database users. Use with `terraform query` (Terraform 1.14 and later).

## Example Usage

```terraform
list "mongodb_db_user" "all" {
  provider = mongodb
}
```

Then run:

```sh
terraform query
```

## Schema

This list resource takes no configuration arguments. It returns every existing `mongodb_db_user`; each result uses the [`mongodb_db_user`](../resources/db_user.md) schema, and its identity is the base64-encoded `id` (`auth_database.username`).
