provider "mongodb" {
  host          = "127.0.0.1"
  port          = "27017"
  username      = "root"
  password      = "root"
  auth_database = "admin"
  tls           = true

  # PEM-encoded CA certificate used to verify the server
  certificate = file(pathexpand("~/.mongodb/ca.pem"))

  # set to true to skip certificate and hostname verification
  insecure_skip_verify = false
}
