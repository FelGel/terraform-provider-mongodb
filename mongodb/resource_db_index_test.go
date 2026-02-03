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

func TestAccMongoDBIndex_Basic(t *testing.T) {
	var indexName = acctest.RandomWithPrefix("tf-acc-test")
	var collectionName = acctest.RandomWithPrefix("tf-acc-test")
	var databaseName = acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "mongodb_db_index.test"
	
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBIndexBasic(databaseName, collectionName, indexName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBIndexExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "db", databaseName),
					resource.TestCheckResourceAttr(resourceName, "collection", collectionName),
					resource.TestCheckResourceAttr(resourceName, "name", indexName),
					resource.TestCheckResourceAttr(resourceName, "keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "keys.0.field", "test_field"),
					resource.TestCheckResourceAttr(resourceName, "keys.0.value", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"timeout"},
			},
		},
	})
}

func TestAccMongoDBIndex_MultipleFields(t *testing.T) {
	var indexName = acctest.RandomWithPrefix("tf-acc-test")
	var collectionName = acctest.RandomWithPrefix("tf-acc-test")
	var databaseName = acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "mongodb_db_index.test"
	
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBIndexMultipleFields(databaseName, collectionName, indexName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBIndexExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "db", databaseName),
					resource.TestCheckResourceAttr(resourceName, "collection", collectionName),
					resource.TestCheckResourceAttr(resourceName, "name", indexName),
					resource.TestCheckResourceAttr(resourceName, "keys.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "keys.0.field", "field1"),
					resource.TestCheckResourceAttr(resourceName, "keys.0.value", "1"),
					resource.TestCheckResourceAttr(resourceName, "keys.1.field", "field2"),
					resource.TestCheckResourceAttr(resourceName, "keys.1.value", "-1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"timeout"},
			},
		},
	})
}

func TestAccMongoDBIndex_Compound(t *testing.T) {
	var indexName = acctest.RandomWithPrefix("tf-acc-compound-index")
	var collectionName = acctest.RandomWithPrefix("tf-acc-coll")
	var databaseName = acctest.RandomWithPrefix("tf-acc-db")
	resourceName := "mongodb_db_index.test"
	
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBIndexCompound(databaseName, collectionName, indexName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBIndexExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "db", databaseName),
					resource.TestCheckResourceAttr(resourceName, "collection", collectionName),
					resource.TestCheckResourceAttr(resourceName, "name", indexName),
					resource.TestCheckResourceAttr(resourceName, "keys.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "keys.0.field", "name"),
					resource.TestCheckResourceAttr(resourceName, "keys.0.value", "1"),
					resource.TestCheckResourceAttr(resourceName, "keys.1.field", "age"),
					resource.TestCheckResourceAttr(resourceName, "keys.1.value", "-1"),
					resource.TestCheckResourceAttr(resourceName, "keys.2.field", "category"),
					resource.TestCheckResourceAttr(resourceName, "keys.2.value", "1"),
				),
			},
		},
	})
}

func TestAccMongoDBIndex_TTLIndex(t *testing.T) {
	var indexName = acctest.RandomWithPrefix("tf-acc-test")
	var collectionName = acctest.RandomWithPrefix("tf-acc-test")
	var databaseName = acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "mongodb_db_index.test"
	
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBIndexTTL(databaseName, collectionName, indexName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBIndexExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "db", databaseName),
					resource.TestCheckResourceAttr(resourceName, "collection", collectionName),
					resource.TestCheckResourceAttr(resourceName, "name", indexName),
					resource.TestCheckResourceAttr(resourceName, "keys.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "keys.0.field", "created_at"),
					resource.TestCheckResourceAttr(resourceName, "keys.0.value", "1"),
					resource.TestCheckResourceAttr(resourceName, "keys.1.field", "expireAfterSeconds"),
					resource.TestCheckResourceAttr(resourceName, "keys.1.value", "3600"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"timeout"},
			},
		},
	})
}

func TestAccMongoDBIndex_GeneratedName(t *testing.T) {
	var collectionName = acctest.RandomWithPrefix("tf-acc-coll")
	var databaseName = acctest.RandomWithPrefix("tf-acc-db")
	resourceName := "mongodb_db_index.test"
	
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMongoDBIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccMongoDBIndexGeneratedName(databaseName, collectionName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBIndexExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "db", databaseName),
					resource.TestCheckResourceAttr(resourceName, "collection", collectionName),
					resource.TestCheckResourceAttr(resourceName, "keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "keys.0.field", "auto_field"),
					resource.TestCheckResourceAttr(resourceName, "keys.0.value", "1"),
				),
			},
		},
	})
}

func testAccCheckMongoDBIndexExists(resourceName string) resource.TestCheckFunc {
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

		db, collectionName, indexName, err := resourceDatabaseIndexParseId(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error parsing ID: %s", err)
		}

		collectionClient := client.Database(db).Collection(collectionName)
		indexes, err := collectionClient.Indexes().List(context.Background())
		if err != nil {
			return fmt.Errorf("error listing indexes: %s", err)
		}

		var results []bson.M
		if err = indexes.All(context.Background(), &results); err != nil {
			return fmt.Errorf("error getting index results: %s", err)
		}

		indexFound := false
		for _, result := range results {
			if name, ok := result["name"]; ok && name == indexName {
				indexFound = true
				break
			}
		}

		if !indexFound {
			return fmt.Errorf("index %s does not exist in collection %s.%s", indexName, db, collectionName)
		}

		return nil
	}
}

func testAccCheckMongoDBIndexDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*MongoDatabaseConfiguration)
	client, err := MongoClientInit(config)
	if err != nil {
		return fmt.Errorf("error connecting to database: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "mongodb_db_index" {
			continue
		}

		db, collectionName, indexName, err := resourceDatabaseIndexParseId(rs.Primary.ID)
		if err != nil {
			continue // If we can't parse the ID, assume it's destroyed
		}

		collectionClient := client.Database(db).Collection(collectionName)
		indexes, err := collectionClient.Indexes().List(context.Background())
		if err != nil {
			continue // If we can't list indexes, assume the collection/index is destroyed
		}

		var results []bson.M
		if err = indexes.All(context.Background(), &results); err != nil {
			continue // If we can't get results, assume it's destroyed
		}

		for _, result := range results {
			if name, ok := result["name"]; ok && name == indexName {
				return fmt.Errorf("index %s still exists in collection %s.%s", indexName, db, collectionName)
			}
		}
	}

	return nil
}

func testAccMongoDBIndexBasic(dbName, collectionName, indexName string) string {
	return fmt.Sprintf(`
resource "mongodb_db_collection" "test" {
  db                   = "%s"
  name                 = "%s"
  deletion_protection  = false
}

resource "mongodb_db_index" "test" {
  depends_on = [mongodb_db_collection.test]
  db         = "%s"
  collection = "%s"
  name       = "%s"
  keys {
    field = "test_field"
    value = "1"
  }
  timeout = 30
}
`, dbName, collectionName, dbName, collectionName, indexName)
}

func testAccMongoDBIndexMultipleFields(dbName, collectionName, indexName string) string {
	return fmt.Sprintf(`
resource "mongodb_db_collection" "test" {
  db                   = "%s"
  name                 = "%s"
  deletion_protection  = false
}

resource "mongodb_db_index" "test" {
  depends_on = [mongodb_db_collection.test]
  db         = "%s"
  collection = "%s"
  name       = "%s"
  keys {
    field = "field1"
    value = "1"
  }
  keys {
    field = "field2"
    value = "-1"
  }
  timeout = 30
}
`, dbName, collectionName, dbName, collectionName, indexName)
}

func testAccMongoDBIndexCompound(dbName, collectionName, indexName string) string {
	return fmt.Sprintf(`
resource "mongodb_db_collection" "test" {
  db                   = "%s"
  name                 = "%s"
  deletion_protection  = false
}

resource "mongodb_db_index" "test" {
  depends_on = [mongodb_db_collection.test]
  db         = "%s"
  collection = "%s"
  name       = "%s"
  keys {
    field = "name"
    value = "1"
  }
  keys {
    field = "age"
    value = "-1"
  }
  keys {
    field = "category"
    value = "1"
  }
  timeout = 30
}
`, dbName, collectionName, dbName, collectionName, indexName)
}

func testAccMongoDBIndexTTL(dbName, collectionName, indexName string) string {
	return fmt.Sprintf(`
resource "mongodb_db_collection" "test" {
  db                   = "%s"
  name                 = "%s"
  deletion_protection  = false
}

resource "mongodb_db_index" "test" {
  depends_on = [mongodb_db_collection.test]
  db         = "%s"
  collection = "%s"
  name       = "%s"
  keys {
    field = "created_at"
    value = "1"
  }
  keys {
    field = "expireAfterSeconds"
    value = "3600"
  }
  timeout = 30
}
`, dbName, collectionName, dbName, collectionName, indexName)
}

func testAccMongoDBIndexGeneratedName(dbName, collectionName string) string {
	return fmt.Sprintf(`
resource "mongodb_db_collection" "test" {
  db                   = "%s"
  name                 = "%s"
  deletion_protection  = false
}

resource "mongodb_db_index" "test" {
  depends_on = [mongodb_db_collection.test]
  db         = "%s"
  collection = "%s"
  keys {
    field = "auto_field"
    value = "1"
  }
  timeout = 30
}
`, dbName, collectionName, dbName, collectionName)
}