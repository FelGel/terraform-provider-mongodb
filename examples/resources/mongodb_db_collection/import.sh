# The ID is the base64-encoded "db.collection",
# e.g. for collection "collection_test" in database "test_db":
printf '%s' "test_db.collection_test" | base64
# dGVzdF9kYi5jb2xsZWN0aW9uX3Rlc3Q=

terraform import mongodb_db_collection.example_collection dGVzdF9kYi5jb2xsZWN0aW9uX3Rlc3Q=
