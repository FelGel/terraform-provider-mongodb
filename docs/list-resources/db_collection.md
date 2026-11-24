---
page_title: "mongodb_db_collection List Resource - mongodb"
subcategory: ""
description: |-
  Lists all MongoDB collections across databases. Use with terraform query (Terraform 1.14 and later).
---

# mongodb_db_collection (List Resource)

Lists all MongoDB collections across databases. Use with `terraform query` (Terraform 1.14 and later).

## Example Usage

```terraform
list "mongodb_db_collection" "all" {
  provider = mongodb
}
```

Then run:

```sh
terraform query
```

## Schema

This list resource takes no configuration arguments. It returns every existing `mongodb_db_collection`; each result uses the [`mongodb_db_collection`](../resources/db_collection.md) schema, and its identity is the base64-encoded `id` (`db.collectionName`).
