resource "mongodb_db_user" "iam" {
  auth_mechanism = "MONGODB-AWS"
  name           = "arn:aws:iam::123456789012:role/my-app-role"
  role {
    role = "readWrite"
    db   = "my_database"
  }
}
