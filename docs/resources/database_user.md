
# Mongo Database User

Provides a Database User resource.

Each user has a set of roles that provide access to the databases.

> **IMPORTANT:** All arguments including the password will be stored in the raw state as plain-text. [Read more about sensitive data in state.](https://developer.hashicorp.com/terraform/state/sensitive-data)

## Example Usages

### Example Usages

#### Create user with predefined role
```hcl
resource "mongodb_db_user" "user" {
  auth_database = "my_database"
  name          = "example"
  password      = "example"
  role {
    role = "readAnyDatabase"
    db   = "my_database"
  }
}
```

#### Create user with custom role `example_role`
```hcl
variable "username" {
  description = "the user name"
}
variable "password" {
  description = "the user password"
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
```

## Argument Reference

* `auth_database` (Required, string) – Database against which Mongo authenticates the user. A user must provide both a username and authentication database to log into MongoDB.
* `name` (Required, string) – Username for authenticating to MongoDB.
* `password` (Required, string, Sensitive) – User's initial password. A value is required to create the database user. Passwords may show up in Terraform related logs and will be stored in the Terraform state file as plain-text. See [Sensitive Data in State](https://developer.hashicorp.com/terraform/state/sensitive-data).
* `role` (Optional, block) – List of user’s roles and the databases/collections on which the roles apply. See [Role Block](#role-block) below for more details.

### Role Block

Block mapping a user's role to a database/collection. A role allows the user to perform particular actions on the specified database. A role on the admin database can include privileges that apply to the other databases as well.

* `role` (Required, string) – Name of the role to grant. See [Create a Database User](https://www.mongodb.com/docs/manual/reference/method/db.createUser/#create-administrative-user-with-roles) `roles`.
  > **NOTE:** You can also use [built-in-roles](https://www.mongodb.com/docs/manual/reference/built-in-roles/).
* `db` (Required, string) – Database on which the user has the specified role. A role on the `admin` database can include privileges that apply to the other databases.


## Attributes Reference

This resource exports the following attributes:

* `id` – The base64-encoded ID of the user in the format `auth_database.username`.
* `name` – The username.
* `auth_database` – The authentication database.


## Import

MongoDB users can be imported using the base64-encoded id, e.g. for a user named `user_test` in database `test_db`:

```sh
printf '%s' "test_db.user_test" | base64
# This encodes db.username to base64
dGVzdF9kYi51c2VyX3Rlc3Q=

terraform import mongodb_db_user.example_user dGVzdF9kYi51c2VyX3Rlc3Q=
```
```