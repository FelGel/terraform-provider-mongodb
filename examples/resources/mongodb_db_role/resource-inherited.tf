resource "mongodb_db_role" "role" {
  database = "admin"
  name     = "new_role"
  privilege {
    db         = "admin"
    collection = ""
    actions    = ["collStats"]
  }
}

resource "mongodb_db_role" "role_2" {
  depends_on = [mongodb_db_role.role]
  database   = "admin"
  name       = "new_role3"

  inherited_role {
    role = mongodb_db_role.role.name
    db   = "admin"
  }
}
