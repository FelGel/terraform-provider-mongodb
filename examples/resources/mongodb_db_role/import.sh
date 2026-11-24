# The ID is the base64-encoded "database.role",
# e.g. for role "role_test" in database "test_db":
printf '%s' "test_db.role_test" | base64
# dGVzdF9kYi5yb2xlX3Rlc3Q=

terraform import mongodb_db_role.example_role dGVzdF9kYi5yb2xlX3Rlc3Q=
