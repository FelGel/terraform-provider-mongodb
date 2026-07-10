package mongodb

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProvider *schema.Provider

// testAccProtoV6ProviderFactories serves the muxed provider (SDKv2 + framework)
// over protocol 6. All resources are framework-backed and use this.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"mongodb": func() (tfprotov6.ProviderServer, error) {
		factory, err := MuxServerFactory(context.Background())
		if err != nil {
			return nil, err
		}
		return factory(), nil
	},
}

// testAccMongoConfig builds a client config from the test environment. Check
// functions use this instead of testAccProvider.Meta(), which is not populated
// when a resource is exercised through the muxed protocol-6 factory.
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

func init() {
	testAccProvider = Provider()
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
