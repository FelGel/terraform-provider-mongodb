---
page_title: "mongodb_db_index List Resource - mongodb"
subcategory: ""
description: |-
  Lists all MongoDB indexes across databases and collections. Use with terraform query (Terraform 1.14 and later).
---

# mongodb_db_index (List Resource)

Lists all MongoDB indexes across databases and collections. Use with `terraform query` (Terraform 1.14 and later).

## Example Usage

```terraform
list "mongodb_db_index" "all" {
  provider = mongodb
}
```

Then run:

```sh
terraform query
```

## Schema

This list resource takes no configuration arguments. It returns every existing `mongodb_db_index`; each result uses the [`mongodb_db_index`](../resources/db_index.md) schema, and its identity is the base64-encoded `id` (`db.collection.indexName`).
