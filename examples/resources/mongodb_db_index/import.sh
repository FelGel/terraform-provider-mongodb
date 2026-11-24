# The ID is the base64-encoded "db.collection.index",
# e.g. for index "example_index" on collection "collection_test" in database "test_db":
printf '%s' "test_db.collection_test.example_index" | base64
# dGVzdF9kYi5jb2xsZWN0aW9uX3Rlc3QuZXhhbXBsZV9pbmRleA==

terraform import mongodb_db_index.example_index dGVzdF9kYi5jb2xsZWN0aW9uX3Rlc3QuZXhhbXBsZV9pbmRleA==
