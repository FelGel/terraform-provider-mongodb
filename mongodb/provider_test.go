package mongodb

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories serves the provider over protocol 6 for
// acceptance tests.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"mongodb": providerserver.NewProtocol6WithError(NewFrameworkProvider()),
}

// testAccMongoConfig builds a client config from the test environment, for
// check functions that talk to MongoDB directly.
func testAccMongoConfig() *MongoDatabaseConfiguration {
	return &MongoDatabaseConfiguration{
		Config: &ClientConfig{
			Host:     getEnvWithDefault("MONGO_HOST", "127.0.0.1"),
			Port:     getEnvWithDefault("MONGO_PORT", "27017"),
			Username: getEnvWithDefault("MONGO_USR", "root"),
			Password: getEnvWithDefault("MONGO_PWD", "root"),
			DB:       getEnvWithDefault("MONGO_AUTH_DB", "admin"),
		},
		MaxConnLifetime: 10,
	}
}

func testAccPreCheck(t *testing.T) {
	if os.Getenv("MONGO_HOST") == "" && os.Getenv("TF_ACC") == "1" {
		t.Log("MONGO_HOST not set, using default localhost")
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
