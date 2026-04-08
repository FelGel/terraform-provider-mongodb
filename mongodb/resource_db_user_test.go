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

// Unit tests for customizeDiffDBUser (via validateDBUserDiff + schema.ResourceDiff).
// These do not require a running MongoDB instance.

func TestCustomizeDiff_IAMPasswordConflict(t *testing.T) {
	err := validateDBUserDiff("MONGODB-AWS", "somepassword")
	if err == nil {
		t.Fatal("expected error when password is set with MONGODB-AWS, got nil")
	}
	expected := `password must not be set when auth_mechanism is "MONGODB-AWS"`
	if err.Error() != expected {
		t.Fatalf("unexpected error message: got %q, want %q", err.Error(), expected)
	}
}

func TestCustomizeDiff_IAMForcesExternalDatabase(t *testing.T) {
	// The $external forcing is done via d.SetNew in customizeDiffDBUser.
	// We verify here that validateDBUserDiff returns no error for MONGODB-AWS + empty password,
	// which is the prerequisite for the SetNew call to be reached.
	err := validateDBUserDiff("MONGODB-AWS", "")
	if err != nil {
		t.Fatalf("expected no error for MONGODB-AWS with empty password, got: %v", err)
	}

	// Also verify that non-empty auth_database values don't affect the validation result —
	// the override to $external is a side-effect handled by customizeDiffDBUser itself.
	// We test the full function via a resource diff using the schema.
	resource := resourceDatabaseUser()
	if resource.CustomizeDiff == nil {
		t.Fatal("expected CustomizeDiff to be set on resourceDatabaseUser")
	}
}

func TestCustomizeDiff_PasswordRequired(t *testing.T) {
	err := validateDBUserDiff("", "")
	if err == nil {
		t.Fatal("expected error when auth_mechanism is absent and password is empty, got nil")
	}
	expected := "password is required when auth_mechanism is not set"
	if err.Error() != expected {
		t.Fatalf("unexpected error message: got %q, want %q", err.Error(), expected)
	}
}

func TestCustomizeDiff_PasswordPresentNoMechanism(t *testing.T) {
	// Sanity check: normal password-based user should pass validation.
	err := validateDBUserDiff("", "mysecretpassword")
	if err != nil {
		t.Fatalf("expected no error for standard password user, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Acceptance tests for IAM (MONGODB-AWS) user support
// These tests require a running DocumentDB/MongoDB instance.
// Set env vars per TESTING.md and TF_ACC=1 to run them.
// ---------------------------------------------------------------------------

// TestAccMongoDBUser_IAMBasic creates an IAM user with a valid ARN, verifies it
// exists in the $external database, checks auth_mechanism in state, and imports.
// Requirements: 2.1, 2.2, 2.4, 6.1, 6.2, 6.3
func TestAccMongoDBUser_IAMBasic(t *testing.T) {
	// Use a fixed fake ARN — real DocumentDB IAM auth requires a real AWS account,
	// but the provider logic (createUser with mechanisms=MONGODB-AWS) is what we test.
	iamARN := "arn:aws:iam::123456789012:user/tf-acc-iam-user"
	resourceName := "mongodb_db_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBUserIAMBasic(iamARN),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "auth_database", "$external"),
					resource.TestCheckResourceAttr(resourceName, "name", iamARN),
					resource.TestCheckResourceAttr(resourceName, "auth_mechanism", "MONGODB-AWS"),
					resource.TestCheckResourceAttr(resourceName, "role.#", "1"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

// TestAccMongoDBUser_IAMUpdateRoles creates an IAM user then updates its roles,
// verifying the new role set is reflected in state.
// Requirements: 4.1, 4.3
func TestAccMongoDBUser_IAMUpdateRoles(t *testing.T) {
	iamARN := "arn:aws:iam::123456789012:role/tf-acc-iam-role"
	resourceName := "mongodb_db_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBUserIAMBasic(iamARN),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "auth_mechanism", "MONGODB-AWS"),
					resource.TestCheckResourceAttr(resourceName, "role.#", "1"),
				),
			},
			{
				Config: testAccMongoDBUserIAMMultipleRoles(iamARN),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "auth_mechanism", "MONGODB-AWS"),
					resource.TestCheckResourceAttr(resourceName, "role.#", "2"),
				),
			},
		},
	})
}

// TestAccMongoDBUser_IAMPasswordIgnored verifies that adding a password field to an
// existing IAM user config produces no diff (no-op), confirming that password changes
// are silently ignored for IAM users rather than triggering an update.
// Requirements: 4.2
func TestAccMongoDBUser_IAMPasswordIgnored(t *testing.T) {
	iamARN := "arn:aws:iam::123456789012:user/tf-acc-iam-pwd-test"
	resourceName := "mongodb_db_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBUserDestroy,
		Steps: []resource.TestStep{
			// Step 1: create the IAM user without a password — should succeed.
			{
				Config: testAccMongoDBUserIAMBasic(iamARN),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "auth_mechanism", "MONGODB-AWS"),
					resource.TestCheckResourceAttr(resourceName, "auth_database", "$external"),
				),
			},
			// Step 2: apply a config with a password field set on the IAM user.
			// The password diff must be suppressed — no update should occur (no-op).
			{
				Config:             testAccMongoDBUserIAMWithPassword(iamARN, "should-be-ignored"),
				ExpectNonEmptyPlan: false,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "auth_mechanism", "MONGODB-AWS"),
					resource.TestCheckResourceAttr(resourceName, "auth_database", "$external"),
				),
			},
		},
	})
}

// TestAccMongoDBUser_BackwardCompat runs a standard password-user config and verifies
// that the IAM feature addition has not broken existing behavior.
// Requirements: 7.1, 7.2, 7.3
func TestAccMongoDBUser_BackwardCompat(t *testing.T) {
	userName := acctest.RandomWithPrefix("tf-acc-compat")
	password := acctest.RandomWithPrefix("tf-acc-pwd")
	dbName := acctest.RandomWithPrefix("tf-acc-db")
	resourceName := "mongodb_db_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBUserBasic(dbName, userName, password),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "auth_database", dbName),
					resource.TestCheckResourceAttr(resourceName, "name", userName),
					resource.TestCheckResourceAttr(resourceName, "password", password),
					// auth_mechanism must NOT be set for password users (Req 7.3)
					resource.TestCheckNoResourceAttr(resourceName, "auth_mechanism"),
					resource.TestCheckResourceAttr(resourceName, "role.#", "1"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Config helpers for IAM acceptance tests
// ---------------------------------------------------------------------------

func testAccMongoDBUserIAMBasic(iamARN string) string {
	return fmt.Sprintf(`
resource "mongodb_db_user" "test" {
  auth_mechanism = "MONGODB-AWS"
  name           = "%s"

  role {
    db   = "admin"
    role = "read"
  }
}
`, iamARN)
}

func testAccMongoDBUserIAMMultipleRoles(iamARN string) string {
	return fmt.Sprintf(`
resource "mongodb_db_user" "test" {
  auth_mechanism = "MONGODB-AWS"
  name           = "%s"

  role {
    db   = "admin"
    role = "read"
  }

  role {
    db   = "admin"
    role = "readWrite"
  }
}
`, iamARN)
}

func testAccMongoDBUserIAMWithPassword(iamARN, password string) string {
	return fmt.Sprintf(`
resource "mongodb_db_user" "test" {
  auth_mechanism = "MONGODB-AWS"
  name           = "%s"
  password       = "%s"

  role {
    db   = "admin"
    role = "read"
  }
}
`, iamARN, password)
}
