# host, port, username and password come from
# MONGO_HOST, MONGO_PORT, MONGO_USR and MONGO_PWD
provider "mongodb" {
  auth_database = "admin"
}
