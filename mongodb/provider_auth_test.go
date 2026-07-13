package mongodb

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccProvider_authMechanismSCRAM exercises the provider-level auth_mechanism
// end to end: it forces the provider's own connection to authenticate with an
// explicit SCRAM-SHA-256 mechanism (rather than letting the driver negotiate)
// and confirms it can still create a user. This is the mechanism plumbing that
// unlocks MONGODB-X509/AWS/OIDC, which CI cannot run against community mongo.
func TestAccProvider_authMechanismSCRAM(t *testing.T) {
	dbName := acctest.RandomWithPrefix("tf-acc-db")
	userName := acctest.RandomWithPrefix("tf-acc-user")
	resourceName := "mongodb_db_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMongoDBUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderAuthMechanismSCRAM(dbName, userName, "acc-pass-1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMongoDBUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", userName),
				),
			},
		},
	})
}

// testAccProviderAuthMechanismSCRAM sets auth_mechanism on the provider block;
// host/port/username/password resolve from the environment defaults the other
// acceptance tests rely on.
func testAccProviderAuthMechanismSCRAM(dbName, userName, password string) string {
	return fmt.Sprintf(`
provider "mongodb" {
  auth_mechanism = "SCRAM-SHA-256"
}

resource "mongodb_db_user" "test" {
  auth_database = %[1]q
  name          = %[2]q
  password      = %[3]q

  role {
    db   = %[1]q
    role = "readWrite"
  }
}
`, dbName, userName, password)
}
