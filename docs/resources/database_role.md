# mongodb_db_role

`mongodb_db_role` provides a Custom DB Role resource. The customDBRoles resource lets you retrieve, create and modify the custom MongoDB roles in your mongo database server. Use custom MongoDB roles to specify custom sets of privileges.


## Example Usages

```hcl
resource "mongodb_db_role" "example_role" {
  name = "role_name"
  database = "my_database"
  privilege {
    db = "admin"
    collection = "*"
    actions = ["collStats"]
  }
  privilege {
    db = "my_database"
    collection = ""
    actions = ["listCollections", "createCollection","createIndex", "dropIndex", "insert", "remove", "renameCollectionSameDB", "update"]
  }


}
```
## Example Usage with inherited roles

```hcl
resource "mongodb_db_role" "role" {
  database = "admin"
  name = "new_role"
  privilege {
    db = "admin"
    collection = ""
    actions = ["collStats"]
  }
}

resource "mongodb_db_role" "role_2" {
  depends_on = [mongodb_db_role.role]
  database = "admin"
  name = "new_role3"

  inherited_role {
    role = mongodb_db_role.role.name
    db =   "admin"
  }
}
```

## Argument Reference

* `database` (Optional, string, default: "admin") – The database of the role.
  
  ~> **IMPORTANT:** If a role is created in a specific database you can only use it as inherited in another role in the same database.

* `name` (Required, string) – Name of the custom role.
  
  -> **NOTE:** The specified role name can only contain letters, digits, underscores, and dashes. Additionally, you cannot specify a role name which meets any of the following criteria:
    * Is a name already used by an existing custom role
    * Is a name of any of the built-in roles, see [built-in-roles](https://www.mongodb.com/docs/manual/reference/built-in-roles/)

### Nested Block: `privilege`
Each `privilege` block supports the following:

* `actions` (Required, list of string) – Array of the privilege actions. For a complete list, see [Custom Role Actions](https://www.mongodb.com/docs/manual/reference/privilege-actions/).
  -> **Note:** The privilege actions available to the Custom Roles API resource represent a subset of the privilege actions available in the Atlas Custom Roles UI.
* `db` (Required, string) – Database on which the action is granted.
* `collection` (Optional, string) – Collection on which the action is granted. If empty, actions are granted on all collections within the specified database.

### Nested Block: `inherited_role`
Each `inherited_role` block supports the following:

* `db` (Required, string) – Database on which the inherited role is granted.  
  -> **NOTE:** This value should be `admin` for all roles except `read` and `readWrite`.
* `role` (Required, string) – Name of the inherited role. This can be another custom role or a [built-in role](https://www.mongodb.com/docs/manual/reference/built-in-roles/).

## Attributes Reference

This resource exports the following attributes:

* `id` – The base64-encoded ID of the role in the format `database.role`.
* `name` – The name of the custom role.
* `database` – The database of the custom role.

## Import


## Import

## Import

Mongodb users can be imported using the hex encoded id, e.g. for a user named `user_test` and his database id `test_db` :

```sh
$ printf '%s' "test_db.role_test"  | base64
## this is the output of the command above it will encode db.rolename to HEX 
dGVzdF9kYi5yb2xlX3Rlc3Q=

$ terraform import mongodb_db_role.example_role  dGVzdF9kYi5yb2xlX3Rlc3Q=
```