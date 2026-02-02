# Mongo Database Collection

Provides a Database Collection resource.

## Example Usages

##### - create collection
```hcl

resource "mongodb_db_collection" "collection_1" {
  db = "my_database"
  name = "example"
  record_pre_images = true
  change_stream_pre_and_post_images = true
  deletion_protection = true
}
```

## Argument Reference

* `db` (Required, string) – Database in which the collection will be created.
* `name` (Required, string) – Collection name.
* `record_pre_images` (Optional, bool, default: false) – Control collection's pre-image support.
* `change_stream_pre_and_post_images` (Optional, bool, default: false) – Enable capturing of full document before and after images for change streams.
* `deletion_protection` (Optional, bool, default: false) – Prevent collection from being dropped.

## Attributes Reference

This resource exports the following attributes:

* `id` – The base64-encoded ID of the collection in the format `db.collection`.
* `name` – The name of the collection.
* `db` – The database of the collection.

## Import

MongoDB collections can be imported using the base64-encoded id, e.g. for a collection named `collection_test` in database `test_db`:

```sh
$ printf '%s' "test_db.collection_test" | base64
# This encodes db.collection to base64
dGVzdF9kYi5jb2xsZWN0aW9uX3Rlc3Q=

$ terraform import mongodb_db_collection.example_collection dGVzdF9kYi5jb2xsZWN0aW9uX3Rlc3Q=
```