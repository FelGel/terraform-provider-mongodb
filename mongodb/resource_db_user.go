package mongodb

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func customizeDiffDBUser(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	authMechanism := d.Get("auth_mechanism").(string)
	password := d.Get("password").(string)

	// On update, suppress password diffs for IAM users instead of erroring —
	// the password field is irrelevant for MONGODB-AWS (Requirement 4.2).
	if authMechanism == "MONGODB-AWS" && password != "" && d.Id() != "" {
		if err := d.Clear("password"); err != nil {
			return err
		}
		password = ""
	}

	if err := validateDBUserDiff(authMechanism, password); err != nil {
		return err
	}

	currentAuthDB := d.Get("auth_database").(string)
	if resolved := resolveAuthDatabase(authMechanism, currentAuthDB); resolved != currentAuthDB {
		return d.SetNew("auth_database", resolved)
	}
	return nil
}

// resolveAuthDatabase returns the effective auth_database for a user.
// For IAM users (MONGODB-AWS), it always returns "$external" regardless of the input.
// For password users, it returns the provided value unchanged.
func resolveAuthDatabase(authMechanism, currentAuthDB string) string {
	if authMechanism == "MONGODB-AWS" {
		return "$external"
	}
	return currentAuthDB
}

// validateDBUserDiff contains the pure cross-field validation logic, extracted for unit testing.
func validateDBUserDiff(authMechanism, password string) error {
	if authMechanism == "MONGODB-AWS" {
		if password != "" {
			return fmt.Errorf(`password must not be set when auth_mechanism is "MONGODB-AWS"`)
		}
		return nil
	}
	if password == "" {
		return fmt.Errorf("password is required when auth_mechanism is not set")
	}
	return nil
}

func resourceDatabaseUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDatabaseUserCreate,
		ReadContext:   resourceDatabaseUserRead,
		UpdateContext: resourceDatabaseUserUpdate,
		DeleteContext: resourceDatabaseUserDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: customdiff.All(
			customizeDiffDBUser,
		),
		Schema: map[string]*schema.Schema{
			"auth_database": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"password": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},
			"auth_mechanism": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateAuthMechanism,
			},
			"role": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 25,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"db": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"role": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceDatabaseUserDelete(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	var stateId = data.State().ID
	var database = data.Get("auth_database").(string)

	id, errEncoding := base64.StdEncoding.DecodeString(stateId)
	if errEncoding != nil {
		return diag.Errorf("ID mismatch %s", errEncoding)
	}

	// StateID is a concatenation of database and username. We only use the username here.
	splitId := strings.Split(string(id), ".")
	userName := splitId[1]

	adminDB := client.Database(database)

	result := adminDB.RunCommand(context.Background(), bson.D{{Key: "dropUser", Value: userName}})
	if result.Err() != nil {
		return diag.Errorf("%s", result.Err())
	}

	return nil
}

func resourceDatabaseUserUpdate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	var stateId = data.State().ID
	_, errEncoding := base64.StdEncoding.DecodeString(stateId)
	if errEncoding != nil {
		return diag.Errorf("ID mismatch %s", errEncoding)
	}

	var userName = data.Get("name").(string)
	var database = data.Get("auth_database").(string)
	var authMechanism = data.Get("auth_mechanism").(string)

	adminDB := client.Database(database)

	// Only update if password or roles have changed
	if data.HasChange("password") || data.HasChange("role") {
		if authMechanism == "MONGODB-AWS" {
			// For IAM users: only update roles when they changed; ignore password changes entirely
			if data.HasChange("role") {
				var roleList []Role
				roles := data.Get("role").(*schema.Set).List()
				roleMapErr := mapstructure.Decode(roles, &roleList)
				if roleMapErr != nil {
					return diag.Errorf("Error decoding map : %s ", roleMapErr)
				}
				rolesValue := roleList
				if rolesValue == nil {
					rolesValue = []Role{}
				}
				result := adminDB.RunCommand(context.Background(), bson.D{
					{Key: "updateUser", Value: userName},
					{Key: "roles", Value: rolesValue},
				})
				if result.Err() != nil {
					return diag.Errorf("Could not update the user : %s ", result.Err())
				}
			}
			// password-only change for IAM user: no-op
		} else {
			var userPassword = data.Get("password").(string)
			var roleList []Role
			roles := data.Get("role").(*schema.Set).List()
			roleMapErr := mapstructure.Decode(roles, &roleList)
			if roleMapErr != nil {
				return diag.Errorf("Error decoding map : %s ", roleMapErr)
			}

			var result *mongo.SingleResult
			if len(roleList) != 0 {
				result = adminDB.RunCommand(context.Background(), bson.D{
					{Key: "updateUser", Value: userName},
					{Key: "pwd", Value: userPassword},
					{Key: "roles", Value: roleList},
				})
			} else {
				result = adminDB.RunCommand(context.Background(), bson.D{
					{Key: "updateUser", Value: userName},
					{Key: "pwd", Value: userPassword},
					{Key: "roles", Value: []bson.M{}},
				})
			}

			if result.Err() != nil {
				return diag.Errorf("Could not update the user : %s ", result.Err())
			}
		}
	}

	newId := database + "." + userName
	encoded := base64.StdEncoding.EncodeToString([]byte(newId))
	data.SetId(encoded)
	return resourceDatabaseUserRead(ctx, data, i)
}

func resourceDatabaseUserRead(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	stateID := data.State().ID
	username, database, err := resourceDatabaseUserParseId(stateID)
	if err != nil {
		return diag.Errorf("%s", err)
	}
	result, decodeError := getUser(client, username, database)
	if decodeError != nil {
		return diag.Errorf("Error decoding user : %s ", err)
	}
	if len(result.Users) == 0 {
		return diag.Errorf("user does not exist")
	}
	roles := make([]interface{}, len(result.Users[0].Roles))

	for i, s := range result.Users[0].Roles {
		roles[i] = map[string]interface{}{
			"db":   s.Db,
			"role": s.Role,
		}
	}
	dataSetError := data.Set("role", roles)
	if dataSetError != nil {
		return diag.Errorf("error setting role : %s ", dataSetError)
	}
	dataSetError = data.Set("auth_database", database)
	if dataSetError != nil {
		return diag.Errorf("error setting auth_db : %s ", dataSetError)
	}
	dataSetError = data.Set("name", username)
	if dataSetError != nil {
		return diag.Errorf("error setting name : %s ", dataSetError)
	}
	dataSetError = data.Set("password", data.Get("password"))
	if dataSetError != nil {
		return diag.Errorf("error setting password : %s ", dataSetError)
	}
	// Detect IAM users from the mechanisms field returned by MongoDB
	for _, m := range result.Users[0].Mechanisms {
		if m == "MONGODB-AWS" {
			dataSetError = data.Set("auth_mechanism", "MONGODB-AWS")
			if dataSetError != nil {
				return diag.Errorf("error setting auth_mechanism : %s ", dataSetError)
			}
			break
		}
	}
	data.SetId(stateID)
	return nil
}

func resourceDatabaseUserCreate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	var database = data.Get("auth_database").(string)
	var userName = data.Get("name").(string)
	var authMechanism = data.Get("auth_mechanism").(string)
	var roleList []Role
	roles := data.Get("role").(*schema.Set).List()
	roleMapErr := mapstructure.Decode(roles, &roleList)
	if roleMapErr != nil {
		return diag.Errorf("Error decoding map : %s ", roleMapErr)
	}

	if authMechanism == "MONGODB-AWS" {
		err := createIAMUser(client, userName, roleList)
		if err != nil {
			return diag.Errorf("Could not create the user : %s ", err)
		}
	} else {
		var userPassword = data.Get("password").(string)
		var user = DbUser{
			Name:     userName,
			Password: userPassword,
		}
		err := createUser(client, user, roleList, database)
		if err != nil {
			return diag.Errorf("Could not create the user : %s ", err)
		}
	}

	str := database + "." + userName
	encoded := base64.StdEncoding.EncodeToString([]byte(str))
	data.SetId(encoded)
	return resourceDatabaseUserRead(ctx, data, i)
}

func resourceDatabaseUserParseId(id string) (string, string, error) {
	result, errEncoding := base64.StdEncoding.DecodeString(id)

	if errEncoding != nil {
		return "", "", fmt.Errorf("unexpected format of ID Error : %s", errEncoding)
	}
	parts := strings.SplitN(string(result), ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected format of ID (%s), expected attribute1.attribute2", id)
	}

	database := parts[0]
	userName := parts[1]

	return userName, database, nil
}
