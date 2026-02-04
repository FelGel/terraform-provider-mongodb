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

func TestAccMongoDBUser_Basic(t *testing.T) {
	var userName = acctest.RandomWithPrefix("tf-acc-user")
	var password = acctest.RandomWithPrefix("tf-acc-pwd")
	var databaseName = acctest.RandomWithPrefix("tf-acc-db")
	resourceName := "mongodb_db_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBUserBasic(databaseName, userName, password),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "auth_database", databaseName),
					resource.TestCheckResourceAttr(resourceName, "name", userName),
					resource.TestCheckResourceAttr(resourceName, "password", password),
					resource.TestCheckResourceAttr(resourceName, "role.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func TestAccMongoDBUser_MultipleRoles(t *testing.T) {
	var userName = acctest.RandomWithPrefix("tf-acc-user")
	var password = acctest.RandomWithPrefix("tf-acc-pwd")
	var databaseName = acctest.RandomWithPrefix("tf-acc-db")
	resourceName := "mongodb_db_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBUserMultipleRoles(databaseName, userName, password),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "auth_database", databaseName),
					resource.TestCheckResourceAttr(resourceName, "name", userName),
					resource.TestCheckResourceAttr(resourceName, "role.#", "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			{
				Config: testAccMongoDBUserBasic(databaseName, userName, password),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "auth_database", databaseName),
					resource.TestCheckResourceAttr(resourceName, "name", userName),
					resource.TestCheckResourceAttr(resourceName, "password", password),
					resource.TestCheckResourceAttr(resourceName, "role.#", "1"),
				),
			},
			{
				Config: testAccMongoDBUserMultipleRoles(databaseName, userName, password),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "auth_database", databaseName),
					resource.TestCheckResourceAttr(resourceName, "name", userName),
					resource.TestCheckResourceAttr(resourceName, "role.#", "2"),
				),
			},
		},
	})
}

func TestAccMongoDBUser_Update(t *testing.T) {
	var userName = acctest.RandomWithPrefix("tf-acc-user")
	var password = acctest.RandomWithPrefix("tf-acc-pwd")
	var updatedPassword = acctest.RandomWithPrefix("tf-acc-pwd-upd")
	var databaseName = acctest.RandomWithPrefix("tf-acc-db")
	resourceName := "mongodb_db_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBUserBasic(databaseName, userName, password),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "auth_database", databaseName),
					resource.TestCheckResourceAttr(resourceName, "name", userName),
					resource.TestCheckResourceAttr(resourceName, "password", password),
					resource.TestCheckResourceAttr(resourceName, "role.#", "1"),
				),
			},
			{
				Config: testAccMongoDBUserBasic(databaseName, userName, updatedPassword),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "auth_database", databaseName),
					resource.TestCheckResourceAttr(resourceName, "name", userName),
					resource.TestCheckResourceAttr(resourceName, "password", updatedPassword),
					resource.TestCheckResourceAttr(resourceName, "role.#", "1"),
				),
			},
		},
	})
}

func TestAccMongoDBUser_AdminDatabase(t *testing.T) {
	var userName = acctest.RandomWithPrefix("tf-acc-user")
	var password = acctest.RandomWithPrefix("tf-acc-pwd")
	resourceName := "mongodb_db_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBUserAdminDatabase(userName, password),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "auth_database", "admin"),
					resource.TestCheckResourceAttr(resourceName, "name", userName),
					resource.TestCheckResourceAttr(resourceName, "password", password),
					resource.TestCheckResourceAttr(resourceName, "role.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func testAccCheckMongoDBUserExists(resourceName string) resource.TestCheckFunc {
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

		userName, database, err := resourceDatabaseUserParseId(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error parsing ID: %s", err)
		}

		result, err := getUser(client, userName, database)
		if err != nil {
			return fmt.Errorf("error getting user: %s", err)
		}

		if len(result.Users) == 0 {
			return fmt.Errorf("user not found: %s", userName)
		}

		return nil
	}
}

func testAccCheckMongoDBUserDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*MongoDatabaseConfiguration)
	client, err := MongoClientInit(config)
	if err != nil {
		return fmt.Errorf("error connecting to database: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "mongodb_db_user" {
			continue
		}

		userName, database, err := resourceDatabaseUserParseId(rs.Primary.ID)
		if err != nil {
			return err
		}

		db := client.Database(database)
		result := db.RunCommand(context.Background(), bson.D{
			{Key: "usersInfo", Value: userName},
		})

		if result.Err() != nil {
			// If there's an error getting the user, it might be deleted
			continue
		}

		var userResult SingleResultGetUser
		if err := result.Decode(&userResult); err != nil {
			return err
		}

		if len(userResult.Users) > 0 {
			return fmt.Errorf("user still exists: %s", userName)
		}
	}

	return nil
}

func testAccMongoDBUserBasic(dbName, userName, password string) string {
	return fmt.Sprintf(`
resource "mongodb_db_user" "test" {
  auth_database = "%s"
  name          = "%s"
  password      = "%s"
  
  role {
    db   = "%s"
    role = "readWrite"
  }
}
`, dbName, userName, password, dbName)
}

func testAccMongoDBUserMultipleRoles(dbName, userName, password string) string {
	return fmt.Sprintf(`
resource "mongodb_db_user" "test" {
  auth_database = "%s"
  name          = "%s"
  password      = "%s"
  
  role {
    db   = "%s"
    role = "read"
  }
  
  role {
    db   = "%s"
    role = "readWrite"
  }
}
`, dbName, userName, password, dbName, dbName)
}

func testAccMongoDBUserAdminDatabase(userName, password string) string {
	return fmt.Sprintf(`
resource "mongodb_db_user" "test" {
  auth_database = "admin"
  name          = "%s"
  password      = "%s"
  
  role {
    db   = "admin"
    role = "userAdminAnyDatabase"
  }
}
`, userName, password)
}
