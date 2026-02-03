package mongodb

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviderFactories map[string]func() (*schema.Provider, error)
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider()
	testAccProviderFactories = map[string]func() (*schema.Provider, error){
		"mongodb": func() (*schema.Provider, error) {
			return testAccProvider, nil
		},
	}
}

func testAccPreCheck(t *testing.T) {
	// Check if required environment variables are set
	if v := os.Getenv("MONGO_HOST"); v == "" {
		if v := os.Getenv("TF_ACC"); v == "1" {
			t.Log("MONGO_HOST not set, using default localhost")
		}
	}

	// Configure the provider with test settings
	d := schema.TestResourceDataRaw(t, testAccProvider.Schema, map[string]interface{}{
		"host":          getEnvWithDefault("MONGO_HOST", "127.0.0.1"),
		"port":          getEnvWithDefault("MONGO_PORT", "27017"),
		"username":      getEnvWithDefault("MONGO_USR", "root"),
		"password":      getEnvWithDefault("MONGO_PWD", "root"),
		"auth_database": getEnvWithDefault("MONGO_AUTH_DB", "admin"),
		"tls":           false,
	})

	// Configure the provider
	_, err := testAccProvider.ConfigureContextFunc(context.Background(), d)
	if err != nil {
		t.Fatalf("Failed to configure provider: %v", err)
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}