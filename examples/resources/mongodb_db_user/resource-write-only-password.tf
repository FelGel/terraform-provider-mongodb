resource "mongodb_db_user" "user_wo" {
  auth_database       = "my_database"
  name                = "example"
  password_wo         = var.password
  password_wo_version = "1"
  role {
    role = "readWrite"
    db   = "my_database"
  }
}
