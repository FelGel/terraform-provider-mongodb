variable "username" {
  description = "the user name"
}
variable "password" {
  description = "the user password"
  sensitive   = true
}

resource "mongodb_db_user" "user_with_custom_role" {
  depends_on    = [mongodb_db_role.example_role]
  auth_database = "my_database"
  name          = var.username
  password      = var.password
  role {
    role = mongodb_db_role.example_role.name
    db   = "my_database"
  }
  role {
    role = "readAnyDatabase"
    db   = "admin"
  }
}
