# The ID is the base64-encoded "auth_database.username",
# e.g. for user "user_test" in database "test_db":
printf '%s' "test_db.user_test" | base64
# dGVzdF9kYi51c2VyX3Rlc3Q=

terraform import mongodb_db_user.example_user dGVzdF9kYi51c2VyX3Rlc3Q=
