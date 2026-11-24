resource "mongodb_db_role" "example_role" {
  name     = "role_name"
  database = "my_database"
  privilege {
    db         = "admin"
    collection = "*"
    actions    = ["collStats"]
  }
  privilege {
    db         = "my_database"
    collection = ""
    actions    = ["listCollections", "createCollection", "createIndex", "dropIndex", "insert", "remove", "renameCollectionSameDB", "update"]
  }
}
