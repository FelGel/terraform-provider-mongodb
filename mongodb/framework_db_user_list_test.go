package mongodb

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// TestAccMongoDBUser_list exercises the mongodb_db_user list resource: it creates
// a user, then runs a query and asserts the user appears in the results.
// List/query resources require Terraform 1.14+, so the test skips below that.
func TestAccMongoDBUser_list(t *testing.T) {
	userName := acctest.RandomWithPrefix("tf-acc-list")
	password := acctest.RandomWithPrefix("tf-acc-pwd")
	// Users created in "admin" are enumerated by the list resource (forAllDBs).
	wantID := base64.StdEncoding.EncodeToString([]byte("admin." + userName))

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBUserListConfig(userName, password),
			},
			{
				Query: true,
				Config: `
provider "mongodb" {}

list "mongodb_db_user" "test" {
  provider = mongodb
}
`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectIdentity("mongodb_db_user.test", map[string]knownvalue.Check{
						"id": knownvalue.StringExact(wantID),
					}),
				},
			},
		},
	})
}

// TestAccMongoDBRole_list creates a custom role, runs a query against the
// mongodb_db_role list resource, and asserts the role appears in the results.
func TestAccMongoDBRole_list(t *testing.T) {
	roleName := acctest.RandomWithPrefix("tf-acc-role-list")
	wantID := base64.StdEncoding.EncodeToString([]byte("admin." + roleName))

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBRoleListConfig(roleName),
			},
			{
				Query: true,
				Config: `
provider "mongodb" {}

list "mongodb_db_role" "test" {
  provider = mongodb
}
`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectIdentity("mongodb_db_role.test", map[string]knownvalue.Check{
						"id": knownvalue.StringExact(wantID),
					}),
				},
			},
		},
	})
}

func testAccMongoDBRoleListConfig(roleName string) string {
	return fmt.Sprintf(`
resource "mongodb_db_role" "test" {
  database = "admin"
  name     = %[1]q

  privilege {
    db         = "admin"
    collection = ""
    actions    = ["find"]
  }
}
`, roleName)
}

// TestAccMongoDBCollection_list creates a collection, queries the
// mongodb_db_collection list resource, and asserts it appears.
func TestAccMongoDBCollection_list(t *testing.T) {
	dbName := acctest.RandomWithPrefix("tfacclistdb")
	collName := acctest.RandomWithPrefix("tfacclistcoll")
	wantID := base64.StdEncoding.EncodeToString([]byte(dbName + "." + collName))

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "mongodb_db_collection" "test" {
  db                  = %[1]q
  name                = %[2]q
  deletion_protection = false
}
`, dbName, collName),
			},
			{
				Query: true,
				Config: `
provider "mongodb" {}

list "mongodb_db_collection" "test" {
  provider = mongodb
}
`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectIdentity("mongodb_db_collection.test", map[string]knownvalue.Check{
						"id": knownvalue.StringExact(wantID),
					}),
				},
			},
		},
	})
}

// TestAccMongoDBIndex_list creates an index, queries the mongodb_db_index list
// resource, and asserts it appears.
func TestAccMongoDBIndex_list(t *testing.T) {
	dbName := acctest.RandomWithPrefix("tfacclistdb")
	collName := acctest.RandomWithPrefix("tfacclistcoll")
	idxName := "tfacc_list_idx"
	wantID := base64.StdEncoding.EncodeToString([]byte(dbName + "." + collName + "." + idxName))

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "mongodb_db_index" "test" {
  db         = %[1]q
  collection = %[2]q
  name       = %[3]q

  keys {
    field = "myfield"
    value = "1"
  }
}
`, dbName, collName, idxName),
			},
			{
				Query: true,
				Config: `
provider "mongodb" {}

list "mongodb_db_index" "test" {
  provider = mongodb
}
`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectIdentity("mongodb_db_index.test", map[string]knownvalue.Check{
						"id": knownvalue.StringExact(wantID),
					}),
				},
			},
		},
	})
}

func testAccMongoDBUserListConfig(userName, password string) string {
	return fmt.Sprintf(`
resource "mongodb_db_user" "test" {
  auth_database = "admin"
  name          = %[1]q
  password      = %[2]q

  role {
    db   = "admin"
    role = "read"
  }
}
`, userName, password)
}
