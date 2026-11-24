resource "mongodb_db_collection" "collection_1" {
  db                                = "my_database"
  name                              = "example"
  change_stream_pre_and_post_images = true
  deletion_protection               = true
}
