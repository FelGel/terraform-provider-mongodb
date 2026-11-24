resource "mongodb_db_user" "user" {
  auth_database = "my_database"
  name          = "example"
  password      = "example"
  role {
    role = "readAnyDatabase"
    db   = "my_database"
  }
}
