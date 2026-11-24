package mongodb

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// Import by identity (Terraform 1.12+); every resource's identity is its id.

func TestAccMongoDBUser_identity(t *testing.T) {
	db, name := acctest.RandomWithPrefix("tf-acc-db"), acctest.RandomWithPrefix("tf-acc-user")
	// The password cannot be read back, so the post-import plan may update it.
	testAccIdentityImport(t, "mongodb_db_user.test", testAccMongoDBUserBasic(db, name, acctest.RandomWithPrefix("tf-acc-pwd")), testAccCheckMongoDBUserDestroy, true)
}

func TestAccMongoDBRole_identity(t *testing.T) {
	db, name := acctest.RandomWithPrefix("tf-acc-db"), acctest.RandomWithPrefix("tf-acc-role")
	testAccIdentityImport(t, "mongodb_db_role.test", testAccMongoDBRoleBasic(db, name), testAccCheckMongoDBRoleDestroy, false)
}

func TestAccMongoDBIndex_identity(t *testing.T) {
	db, coll, name := acctest.RandomWithPrefix("tf-acc-test"), acctest.RandomWithPrefix("tf-acc-test"), acctest.RandomWithPrefix("tf-acc-test")
	// timeout is configuration-only and not recovered on import.
	testAccIdentityImport(t, "mongodb_db_index.test", testAccMongoDBIndexBasic(db, coll, name), testAccCheckMongoDBIndexDestroy, true)
}

func TestAccMongoDBCollection_identity(t *testing.T) {
	db, name := acctest.RandomWithPrefix("tf-acc-db"), acctest.RandomWithPrefix("tf-acc-test")
	// deletion_protection is configuration-only and not recovered on import.
	testAccIdentityImport(t, "mongodb_db_collection.test", testAccMongoDBCollectionBasic(db, name), testAccCheckMongoDBCollectionDestroy, true)
}

func testAccIdentityImport(t *testing.T, resourceName, config string, checkDestroy resource.TestCheckFunc, allowNonEmptyImportPlan bool) {
	t.Helper()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentityValueMatchesState(resourceName, tfjsonpath.New("id")),
				},
			},
			{
				ResourceName:       resourceName,
				ImportState:        true,
				ImportStateKind:    resource.ImportBlockWithResourceIdentity,
				ExpectNonEmptyPlan: allowNonEmptyImportPlan,
			},
		},
	})
}
