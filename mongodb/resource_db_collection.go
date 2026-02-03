package mongodb

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func resourceDatabaseCollection() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDatabaseCollectionCreate,
		ReadContext:   resourceDatabaseCollectionRead,
		UpdateContext: resourceDatabaseCollectionUpdate,
		DeleteContext: resourceDatabaseCollectionDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"db": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"deletion_protection": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			// "record_pre_images": {
			// 	Type:     schema.TypeBool,
			// 	Optional: true,
			// 	Deprecated: "This field is deprecated in favor of change_stream_pre_and_post_images",
			// },
			"change_stream_pre_and_post_images": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceDatabaseCollectionCreate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to db : %s ", connectionError)
	}
	var db = data.Get("db").(string)
	var collectionName = data.Get("name").(string)
	// var recordPreImages = data.Get("record_pre_images").(bool)
	var changeStreamPreAndPostImages = data.Get("change_stream_pre_and_post_images").(bool)

	dbClient := client.Database(db)

	err := dbClient.CreateCollection(context.Background(), collectionName)
	if err != nil {
		return diag.Errorf("Could not create the collection : %s ", err)
	}

	// if recordPreImages {
	// 	var recordPreImages = data.Get("record_pre_images").(bool)
	// 	_err := setPreRecordImages(dbClient, collectionName, recordPreImages)
	// 	if _err != nil {
	// 		return _err
	// 	}
	// }

	if changeStreamPreAndPostImages {
		_err := setChangeStreamPreAndPostImages(dbClient, collectionName, changeStreamPreAndPostImages)
		if _err != nil {
			return _err
		}
	}

	SetId(data, []string{db, collectionName})
	return resourceDatabaseCollectionRead(ctx, data, i)
}

func resourceDatabaseCollectionRead(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	dbClient, db, collectionName, err := parseDbAndCollection(data, i)
	if err != nil {
		return diag.Errorf("%s", err)
	}

	// Construct the filter to check if collection exists
	filter := bson.M{"name": collectionName}

	// List the collections with the specified name
	cursor, err := dbClient.ListCollections(context.Background(), filter)
	if err != nil {
		return diag.Errorf("Failed to list collections : %s ", err)
	}

	// Check if the collection exists
	exists := cursor.Next(context.Background())
	if !exists {
		return diag.Errorf("collection does not exist")
	}

	var collectionSpec *mongo.CollectionSpecification
	err = cursor.Decode(&collectionSpec)
	if err != nil {
		return diag.Errorf("Failed decode collection specification : %s ", err)
	}

	// recordPreImages, _ := collectionSpec.Options.Lookup("recordPreImages").BooleanOK()
	changeStreamPreAndPostImages, _ := collectionSpec.Options.Lookup("changeStreamPreAndPostImages").DocumentOK()
	changeStreamEnabled := false
	if changeStreamPreAndPostImages != nil {
		enabled, ok := changeStreamPreAndPostImages.Lookup("enabled").BooleanOK()
		if ok {
			changeStreamEnabled = enabled
		}
	}

	_ = data.Set("db", db)
	_ = data.Set("name", collectionName)
	_ = data.Set("deletion_protection", data.Get("deletion_protection").(bool))
	// _ = data.Set("record_pre_images", recordPreImages)
	_ = data.Set("change_stream_pre_and_post_images", changeStreamEnabled)
	return nil
}

func resourceDatabaseCollectionUpdate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	dbClient, _, collectionName, err := parseDbAndCollection(data, i)
	if err != nil {
		return diag.Errorf("%s", err)
	}

	// var recordPreImages = data.Get("record_pre_images").(bool)
	// _err := setPreRecordImages(dbClient, collectionName, recordPreImages)
	// if _err != nil {
	// 	return _err
	// }

	var changeStreamPreAndPostImages = data.Get("change_stream_pre_and_post_images").(bool)
	_err := setChangeStreamPreAndPostImages(dbClient, collectionName, changeStreamPreAndPostImages)
	if _err != nil {
		return _err
	}

	return resourceDatabaseCollectionRead(ctx, data, i)
}

func resourceDatabaseCollectionDelete(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	dbClient, _, collectionName, err := parseDbAndCollection(data, i)
	if err != nil {
		return diag.Errorf("%s", err)
	}

	_err := dropCollection(dbClient, collectionName, data)
	if _err != nil {
		return _err
	}

	return nil
}

func dropCollection(dbClient *mongo.Database, collectionName string, data *schema.ResourceData) diag.Diagnostics {
	if data.Get("deletion_protection").(bool) {
		return diag.Errorf("Can't delete collection because deletion protection is enabled")
	}

	collectionClient := dbClient.Collection(collectionName)
	err := collectionClient.Drop(context.Background())
	if err != nil {
		return diag.Errorf("%s", err)
	}

	return nil
}

func resourceDatabaseCollectionParseId(id string) (string, string, error) {
	parts, err := ParseId(id, 2)
	if err != nil {
		return "", "", err
	}

	db := parts[0]
	collectionName := parts[1]
	return db, collectionName, nil
}

// func setPreRecordImages(dbClient *mongo.Database, collectionName string, recordPreImages bool) diag.Diagnostics {
// 	result := dbClient.RunCommand(context.Background(), bson.D{{Key: "collMod", Value: collectionName},
// 		{Key: "recordPreImages", Value: recordPreImages}})
// 	if result.Err() != nil {
// 		return diag.Errorf("Failed to set record pre-images: %s", result.Err())
// 	}
// 	return nil
// }

func setChangeStreamPreAndPostImages(dbClient *mongo.Database, collectionName string, enabled bool) diag.Diagnostics {
	result := dbClient.RunCommand(context.Background(), bson.D{
		{Key: "collMod", Value: collectionName},
		{Key: "changeStreamPreAndPostImages", Value: bson.D{
			{Key: "enabled", Value: enabled},
		}},
	})
	if result.Err() != nil {
		return diag.Errorf("Failed to set change stream pre and post images: %s", result.Err())
	}
	return nil
}

func parseDbAndCollection(data *schema.ResourceData, i interface{}) (*mongo.Database, string, string, error) {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return nil, "", "", fmt.Errorf("error connecting to database : %s ", connectionError)
	}

	db, collectionName, err := resourceDatabaseCollectionParseId(data.State().ID)
	if err != nil {
		return nil, "", "", fmt.Errorf("ID mismatch %s", err)
	}

	dbClient := client.Database(db)
	return dbClient, db, collectionName, nil
}
