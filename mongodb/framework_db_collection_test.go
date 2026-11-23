package mongodb

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestAccMongoDBCollection_Basic(t *testing.T) {
	var collectionName = acctest.RandomWithPrefix("tf-acc-test")
	var databaseName = acctest.RandomWithPrefix("tf-acc-db")
	resourceName := "mongodb_db_collection.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMongoDBCollectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBCollectionBasic(databaseName, collectionName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBCollectionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "db", databaseName),
					resource.TestCheckResourceAttr(resourceName, "name", collectionName),
					resource.TestCheckResourceAttr(resourceName, "deletion_protection", "false"),
					resource.TestCheckResourceAttr(resourceName, "change_stream_pre_and_post_images", "false"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"deletion_protection"},
			},
		},
	})
}

func TestAccMongoDBCollection_WithChangeStreamImages(t *testing.T) {
	var collectionName = acctest.RandomWithPrefix("tf-acc-test")
	var databaseName = acctest.RandomWithPrefix("tf-acc-db")
	resourceName := "mongodb_db_collection.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMongoDBCollectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBCollectionWithChangeStreamImages(databaseName, collectionName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBCollectionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "db", databaseName),
					resource.TestCheckResourceAttr(resourceName, "name", collectionName),
					resource.TestCheckResourceAttr(resourceName, "deletion_protection", "false"),
					resource.TestCheckResourceAttr(resourceName, "change_stream_pre_and_post_images", "true"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"deletion_protection"},
			},
		},
	})
}

func TestAccMongoDBCollection_Update(t *testing.T) {
	var collectionName = acctest.RandomWithPrefix("tf-acc-test")
	var databaseName = acctest.RandomWithPrefix("tf-acc-db")
	resourceName := "mongodb_db_collection.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMongoDBCollectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBCollectionBasic(databaseName, collectionName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBCollectionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "change_stream_pre_and_post_images", "false"),
				),
			},
			{
				Config: testAccMongoDBCollectionWithChangeStreamImages(databaseName, collectionName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBCollectionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "change_stream_pre_and_post_images", "true"),
				),
			},
		},
	})
}

func testAccCheckMongoDBCollectionExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		config := testAccMongoConfig()
		client, err := MongoClientInit(config)
		if err != nil {
			return fmt.Errorf("error connecting to database: %s", err)
		}

		db, collectionName, err := resourceDatabaseCollectionParseId(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error parsing ID: %s", err)
		}

		dbClient := client.Database(db)
		filter := bson.M{"name": collectionName}
		cursor, err := dbClient.ListCollections(context.Background(), filter)
		if err != nil {
			return fmt.Errorf("error listing collections: %s", err)
		}

		if !cursor.Next(context.Background()) {
			return fmt.Errorf("collection %s does not exist in database %s", collectionName, db)
		}

		return nil
	}
}

func testAccCheckMongoDBCollectionDestroy(s *terraform.State) error {
	config := testAccMongoConfig()
	client, err := MongoClientInit(config)
	if err != nil {
		return fmt.Errorf("error connecting to database: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "mongodb_db_collection" {
			continue
		}

		db, collectionName, err := resourceDatabaseCollectionParseId(rs.Primary.ID)
		if err != nil {
			continue // If we can't parse the ID, assume it's destroyed
		}

		dbClient := client.Database(db)
		filter := bson.M{"name": collectionName}
		cursor, err := dbClient.ListCollections(context.Background(), filter)
		if err != nil {
			continue // If we can't list collections, assume it's destroyed
		}

		if cursor.Next(context.Background()) {
			return fmt.Errorf("collection %s still exists in database %s", collectionName, db)
		}
	}

	return nil
}

// TestAccMongoDBCollection_deletionProtection verifies that deletion_protection
// blocks destroy while enabled, and that clearing it allows cleanup.
func TestAccMongoDBCollection_deletionProtection(t *testing.T) {
	dbName := acctest.RandomWithPrefix("tf-acc-db")
	collName := acctest.RandomWithPrefix("tf-acc-coll")
	resourceName := "mongodb_db_collection.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMongoDBCollectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBCollectionProtected(dbName, collName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBCollectionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "deletion_protection", "true"),
				),
			},
			{
				// Removing the collection while protected must fail the destroy.
				Config:      "// deletion_protection blocks destroy\n",
				ExpectError: regexp.MustCompile(`deletion protection`),
			},
			{
				// Clearing protection allows the collection to be destroyed (teardown).
				Config: testAccMongoDBCollectionProtected(dbName, collName, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBCollectionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "deletion_protection", "false"),
				),
			},
		},
	})
}

func testAccMongoDBCollectionProtected(dbName, collectionName string, protected bool) string {
	return fmt.Sprintf(`
resource "mongodb_db_collection" "test" {
  db                  = %[1]q
  name                = %[2]q
  deletion_protection = %[3]t
}
`, dbName, collectionName, protected)
}

func testAccMongoDBCollectionBasic(dbName, collectionName string) string {
	return fmt.Sprintf(`
resource "mongodb_db_collection" "test" {
  db                   = "%s"
  name                 = "%s"
  deletion_protection  = false
}
`, dbName, collectionName)
}

func testAccMongoDBCollectionWithChangeStreamImages(dbName, collectionName string) string {
	return fmt.Sprintf(`
resource "mongodb_db_collection" "test" {
  db                                  = "%s"
  name                                = "%s"
  deletion_protection                 = false
  change_stream_pre_and_post_images   = true
}
`, dbName, collectionName)
}
