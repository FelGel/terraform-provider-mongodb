
# Mongo Database User

Provides a Database User resource.

Each user has a set of roles that provide access to the databases.

> **IMPORTANT:** The `password` argument is stored in the raw state as plain-text. [Read more about sensitive data in state.](https://developer.hashicorp.com/terraform/state/sensitive-data) To keep the password out of state entirely, use the write-only `password_wo` argument (Terraform 1.11+).

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

#### Create a user with a write-only password (Terraform 1.11+)

`password_wo` is never stored in Terraform state. Bump `password_wo_version` to rotate it.

```hcl
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
```

#### Create an IAM (MONGODB-AWS) user for Amazon DocumentDB

For `MONGODB-AWS`, `name` must be an AWS IAM ARN, no password is set, and the user lives in the `$external` database.

```hcl
resource "mongodb_db_user" "iam" {
  auth_mechanism = "MONGODB-AWS"
  name           = "arn:aws:iam::123456789012:role/my-app-role"
  role {
    role = "readWrite"
    db   = "my_database"
  }
}
```

## Argument Reference

* `auth_database` (Optional, string) – Database against which Mongo authenticates the user. Defaults to `$external` for `MONGODB-AWS` users; required for password users.
* `name` (Required, string) – Username for authenticating to MongoDB. For `MONGODB-AWS` this must be a valid AWS IAM ARN (`arn:aws:iam::<account-id>:(user|role)/<name>`).
* `password` (Optional, string, Sensitive) – User's password, stored in state as plain-text. Mutually exclusive with `password_wo`. Required for password users unless `password_wo` is set. See [Sensitive Data in State](https://developer.hashicorp.com/terraform/state/sensitive-data).
* `password_wo` (Optional, string, [Write-only](https://developer.hashicorp.com/terraform/language/resources/ephemeral#write-only-arguments)) – User's password, supplied via config and **never stored in state**. Requires Terraform 1.11+. Mutually exclusive with `password`; requires `password_wo_version`.
* `password_wo_version` (Optional, string) – Change this to rotate the write-only `password_wo` (write-only values aren't tracked in state, so this is the update trigger).
* `auth_mechanism` (Optional, string) – Authentication mechanism. Either `MONGODB-AWS` (Amazon DocumentDB IAM authentication) or empty (standard SCRAM password auth). When `MONGODB-AWS`, `password`/`password_wo` must not be set.
* `role` (Optional, block) – List of user’s roles and the databases/collections on which the roles apply. See [Role Block](#role-block) below for more details.
* `authentication_restriction` (Optional, block) – Restricts the IP addresses/CIDR ranges from which the user may connect and to which server addresses. See [Authentication Restriction Block](#authentication-restriction-block) below.

### Authentication Restriction Block

Maps to MongoDB's user [`authenticationRestrictions`](https://www.mongodb.com/docs/manual/reference/method/db.createUser/#authentication-restrictions). Available in Community MongoDB (3.6+). May be repeated; a connection is allowed if it satisfies any one restriction block.

* `client_source` (Optional, list of string) – IP addresses or CIDR ranges from which the user is allowed to connect.
* `server_address` (Optional, list of string) – IP addresses or CIDR ranges of the MongoDB instance addresses the user is allowed to connect to.

> **NOTE:** The configured value is preserved in state as written; the provider does not read the restrictions back from the server.

### Role Block

Block mapping a user's role to a database/collection. A role allows the user to perform particular actions on the specified database. A role on the admin database can include privileges that apply to the other databases as well.

* `role` (Required, string) – Name of the role to grant. See [Create a Database User](https://www.mongodb.com/docs/manual/reference/method/db.createUser/#create-administrative-user-with-roles) `roles`.
  > **NOTE:** You can also use [built-in-roles](https://www.mongodb.com/docs/manual/reference/built-in-roles/).
* `db` (Required, string) – Database on which the user has the specified role. A role on the `admin` database can include privileges that apply to the other databases.


## Attributes Reference

This resource exports the following attributes:

* `id` – The base64-encoded ID of the user in the format `auth_database.username`.
* `name` – The username.
* `auth_database` – The authentication database (`$external` for `MONGODB-AWS` users).
* `auth_mechanism` – Set to `MONGODB-AWS` for IAM users; unset for password users.


## Import

MongoDB users can be imported using the base64-encoded id, e.g. for a user named `user_test` in database `test_db`:

```sh
printf '%s' "test_db.user_test" | base64
# This encodes db.username to base64
dGVzdF9kYi51c2VyX3Rlc3Q=

terraform import mongodb_db_user.example_user dGVzdF9kYi51c2VyX3Rlc3Q=
```
```