package mongodb

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestAccMongoDBRole_Basic(t *testing.T) {
	var roleName = acctest.RandomWithPrefix("tf-acc-role")
	var databaseName = acctest.RandomWithPrefix("tf-acc-db")
	resourceName := "mongodb_db_role.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBRoleBasic(databaseName, roleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "database", databaseName),
					resource.TestCheckResourceAttr(resourceName, "name", roleName),
					resource.TestCheckResourceAttr(resourceName, "privilege.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccMongoDBRole_WithInheritedRoles(t *testing.T) {
	var roleName = acctest.RandomWithPrefix("tf-acc-role")
	var databaseName = acctest.RandomWithPrefix("tf-acc-db")
	resourceName := "mongodb_db_role.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBRoleWithInheritedRoles(databaseName, roleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "database", databaseName),
					resource.TestCheckResourceAttr(resourceName, "name", roleName),
					resource.TestCheckResourceAttr(resourceName, "privilege.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "inherited_role.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccMongoDBRole_MultiplePrivileges(t *testing.T) {
	var roleName = acctest.RandomWithPrefix("tf-acc-role")
	var databaseName = acctest.RandomWithPrefix("tf-acc-db")
	resourceName := "mongodb_db_role.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBRoleMultiplePrivileges(databaseName, roleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "database", databaseName),
					resource.TestCheckResourceAttr(resourceName, "name", roleName),
					resource.TestCheckResourceAttr(resourceName, "privilege.#", "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckMongoDBRoleExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		config := testAccProvider.Meta().(*MongoDatabaseConfiguration)
		client, err := MongoClientInit(config)
		if err != nil {
			return fmt.Errorf("error connecting to database: %s", err)
		}

		roleName, database, err := resourceDatabaseRoleParseId(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error parsing ID: %s", err)
		}

		result, err := getRole(client, roleName, database)
		if err != nil {
			return fmt.Errorf("error getting role: %s", err)
		}

		if len(result.Roles) == 0 {
			return fmt.Errorf("role not found: %s", roleName)
		}

		return nil
	}
}

func testAccCheckMongoDBRoleDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*MongoDatabaseConfiguration)
	client, err := MongoClientInit(config)
	if err != nil {
		return fmt.Errorf("error connecting to database: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "mongodb_db_role" {
			continue
		}

		roleName, database, err := resourceDatabaseRoleParseId(rs.Primary.ID)
		if err != nil {
			return err
		}

		db := client.Database(database)
		result := db.RunCommand(context.Background(), bson.D{
			{Key: "rolesInfo", Value: roleName},
			{Key: "showPrivileges", Value: true},
		})

		if result.Err() != nil {
			// If there's an error getting the role, it might be deleted
			continue
		}

		var roleResult SingleResultGetRole
		if err := result.Decode(&roleResult); err != nil {
			return err
		}

		if len(roleResult.Roles) > 0 {
			return fmt.Errorf("role still exists: %s", roleName)
		}
	}

	return nil
}

func testAccMongoDBRoleBasic(dbName, roleName string) string {
	return fmt.Sprintf(`
resource "mongodb_db_role" "test" {
  database = "%s"
  name     = "%s"
  
  privilege {
    db         = "%s"
    collection = "test_collection"
    actions    = ["find", "insert", "update"]
  }
}
`, dbName, roleName, dbName)
}

func testAccMongoDBRoleWithInheritedRoles(dbName, roleName string) string {
	return fmt.Sprintf(`
resource "mongodb_db_role" "test" {
  database = "%s"
  name     = "%s"
  
  privilege {
    db         = "%s"
    collection = "test_collection"
    actions    = ["find", "insert"]
  }
  
  inherited_role {
    db   = "%s"
    role = "read"
  }
}
`, dbName, roleName, dbName, dbName)
}

func testAccMongoDBRoleMultiplePrivileges(dbName, roleName string) string {
	return fmt.Sprintf(`
resource "mongodb_db_role" "test" {
  database = "%s"
  name     = "%s"
  
  privilege {
    db         = "%s"
    collection = "collection1"
    actions    = ["find", "insert"]
  }
  
  privilege {
    db         = "%s"
    collection = "collection2"
    actions    = ["find", "remove", "update"]
  }
}
`, dbName, roleName, dbName, dbName)
}
