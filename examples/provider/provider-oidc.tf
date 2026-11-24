provider "mongodb" {
  host           = "mongo.example.internal"
  auth_database  = "$external"
  auth_mechanism = "MONGODB-OIDC"
  auth_mechanism_properties = {
    ENVIRONMENT    = "gcp"
    TOKEN_RESOURCE = "https://mongo.example.internal"
  }
}
