# Configure the MongoDB Provider
provider "mongodb" {
  host          = "127.0.0.1"
  port          = "27017"
  username      = "root"
  password      = "root"
  auth_database = "admin"
  tls           = true
  replica_set   = "replica-set"           # optional
  retrywrites   = false                   # default true
  direct        = true                    # default false
  proxy         = "socks5://myproxy:8080" # optional
}
