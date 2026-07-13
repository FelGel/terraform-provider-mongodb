package mongodb

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var dbIndexKeyObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"field": types.StringType,
	"value": types.StringType,
}}

type dbIndexResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Db                      types.String `tfsdk:"db"`
	Collection              types.String `tfsdk:"collection"`
	Keys                    types.List   `tfsdk:"keys"`
	Name                    types.String `tfsdk:"name"`
	PartialFilterExpression types.String `tfsdk:"partial_filter_expression"`
	Hidden                  types.Bool   `tfsdk:"hidden"`
	Timeout                 types.Int64  `tfsdk:"timeout"`
}

type dbIndexKeyModel struct {
	Field types.String `tfsdk:"field"`
	Value types.String `tfsdk:"value"`
}

type dbIndexResource struct {
	config *MongoDatabaseConfiguration
}

func newDBIndexResource() resource.Resource { return &dbIndexResource{} }

var (
	_ resource.Resource                = &dbIndexResource{}
	_ resource.ResourceWithConfigure   = &dbIndexResource{}
	_ resource.ResourceWithImportState = &dbIndexResource{}
	_ resource.ResourceWithIdentity    = &dbIndexResource{}
)

func (r *dbIndexResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_db_index"
}

func (r *dbIndexResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}

func (r *dbIndexResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"db": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"collection": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			// name is Optional+Computed: MongoDB auto-generates a name when omitted,
			// so keep prior state when the config leaves it empty (mirrors the SDKv2
			// DiffSuppressFunc old-nonempty/new-empty case).
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"partial_filter_expression": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString(""),
				Description:   "A JSON string representing the partialFilterExpression for a partial index. Example: {\"field\": {\"$exists\": true}}",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"hidden": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If true, the index is hidden from the query planner (MongoDB 4.4+). Can be toggled without recreating the index.",
			},
			"timeout": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(30),
			},
		},
		Blocks: map[string]schema.Block{
			"keys": schema.ListNestedBlock{
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"field": schema.StringAttribute{Required: true},
						"value": schema.StringAttribute{Required: true},
					},
				},
			},
		},
	}
}

func (r *dbIndexResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(*MongoDatabaseConfiguration)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *MongoDatabaseConfiguration, got %T", req.ProviderData))
		return
	}
	r.config = config
}

func (r *dbIndexResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dbIndexResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := MongoClientInit(r.config)
	if err != nil {
		resp.Diagnostics.AddError("Error connecting to db", err.Error())
		return
	}

	db := plan.Db.ValueString()
	collectionName := plan.Collection.ValueString()
	indexName, err := r.createIndex(ctx, client, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Could not create the index", err.Error())
		return
	}

	plan.ID = types.StringValue(base64.StdEncoding.EncodeToString([]byte(strings.Join([]string{db, collectionName, indexName}, "."))))
	if err := r.readIndexInto(client, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading index after create", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, dbUserIdentityModel{ID: plan.ID})...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dbIndexResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dbIndexResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := MongoClientInit(r.config)
	if err != nil {
		resp.Diagnostics.AddError("Error connecting to database", err.Error())
		return
	}

	if err := r.readIndexInto(client, &state); err != nil {
		resp.Diagnostics.AddError("Error reading index", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, dbUserIdentityModel{ID: state.ID})...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dbIndexResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state dbIndexResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := MongoClientInit(r.config)
	if err != nil {
		resp.Diagnostics.AddError("Error connecting to database", err.Error())
		return
	}

	// Only the hidden flag is mutable in place; everything else forces replacement.
	if !plan.Hidden.Equal(state.Hidden) {
		db, collectionName, indexName, parseErr := resourceDatabaseIndexParseId(state.ID.ValueString())
		if parseErr != nil {
			resp.Diagnostics.AddError("ID mismatch", parseErr.Error())
			return
		}
		result := client.Database(db).RunCommand(context.Background(), bson.D{
			{Key: "collMod", Value: collectionName},
			{Key: "index", Value: bson.D{
				{Key: "name", Value: indexName},
				{Key: "hidden", Value: plan.Hidden.ValueBool()},
			}},
		})
		if result.Err() != nil {
			resp.Diagnostics.AddError("Failed to update index hidden state", result.Err().Error())
			return
		}
	}

	if err := r.readIndexInto(client, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading index after update", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, dbUserIdentityModel{ID: plan.ID})...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dbIndexResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dbIndexResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := MongoClientInit(r.config)
	if err != nil {
		resp.Diagnostics.AddError("Error connecting to database", err.Error())
		return
	}

	db, collectionName, indexName, parseErr := resourceDatabaseIndexParseId(state.ID.ValueString())
	if parseErr != nil {
		resp.Diagnostics.AddError("Failed to parse index ID", parseErr.Error())
		return
	}
	if dropErr := client.Database(db).Collection(collectionName).Indexes().DropOne(context.TODO(), indexName); dropErr != nil {
		resp.Diagnostics.AddError("Could not delete the index", dropErr.Error())
	}
}

func (r *dbIndexResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// createIndex reimplements the SDKv2 createIndex against the framework model
// (the SDKv2 version is bound to *schema.ResourceData and can't be reused).
func (r *dbIndexResource) createIndex(ctx context.Context, client *mongo.Client, plan *dbIndexResourceModel) (string, error) {
	collectionClient := client.Database(plan.Db.ValueString()).Collection(plan.Collection.ValueString())

	var keys []dbIndexKeyModel
	if diags := plan.Keys.ElementsAs(ctx, &keys, false); diags.HasError() {
		return "", fmt.Errorf("invalid keys")
	}

	indexOptions := options.Index()
	indexKeys := bson.D{}
	for _, k := range keys {
		keyField := k.Field.ValueString()
		value := k.Value.ValueString()

		if keyField == "expireAfterSeconds" {
			valueInt, err := strconv.Atoi(value)
			if err != nil {
				return "", fmt.Errorf("expireAfterSeconds value must be integer : %s ", err)
			}
			if valueInt >= 0 {
				indexOptions.SetExpireAfterSeconds(int32(valueInt))
				continue
			}
		}

		if strings.ToLower(keyField) == "unique" && (strings.ToLower(value) == "true" || strings.ToLower(value) == "false") {
			indexOptions.SetUnique(strings.ToLower(value) == "true")
			continue
		} else if strings.ToLower(keyField) == "sparse" && (strings.ToLower(value) == "true" || strings.ToLower(value) == "false") {
			indexOptions.SetSparse(strings.ToLower(value) == "true")
			continue
		} else if value == "1" {
			indexKeys = append(indexKeys, bson.E{Key: keyField, Value: 1})
		} else if value == "-1" {
			indexKeys = append(indexKeys, bson.E{Key: keyField, Value: -1})
		} else if value == "true" {
			indexKeys = append(indexKeys, bson.E{Key: keyField, Value: true})
		} else if value == "false" {
			indexKeys = append(indexKeys, bson.E{Key: keyField, Value: false})
		} else {
			indexKeys = append(indexKeys, bson.E{Key: keyField, Value: value})
		}
	}

	if name := plan.Name.ValueString(); len(name) > 0 {
		indexOptions.SetName(name)
	}

	if partialFilter := plan.PartialFilterExpression.ValueString(); len(partialFilter) > 0 {
		var filterDoc bson.D
		if err := bson.UnmarshalExtJSON([]byte(partialFilter), false, &filterDoc); err != nil {
			return "", fmt.Errorf("Invalid partial_filter_expression JSON: %s", err)
		}
		indexOptions.SetPartialFilterExpression(filterDoc)
	}

	if plan.Hidden.ValueBool() {
		indexOptions.SetHidden(true)
	}

	indexModel := mongo.IndexModel{Keys: indexKeys, Options: indexOptions}

	timeout := int(plan.Timeout.ValueInt64())
	cctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	return collectionClient.Indexes().CreateOne(cctx, indexModel)
}

// readIndexInto mirrors the SDKv2 read: it rebuilds keys (including the unique
// and expireAfterSeconds pseudo-entries) in the same order, and marshals
// partialFilterExpression back with the identical bson.MarshalExtJSON call so
// the value round-trips without a diff. Leaves timeout and id as-is.
func (r *dbIndexResource) readIndexInto(client *mongo.Client, m *dbIndexResourceModel) error {
	db, collectionName, indexName, err := resourceDatabaseIndexParseId(m.ID.ValueString())
	if err != nil {
		return err
	}

	collectionClient := client.Database(db).Collection(collectionName)
	cursor, err := collectionClient.Indexes().List(context.Background())
	if err != nil {
		return fmt.Errorf("Failed to list indexes: %s", err)
	}
	var results []bson.M
	if err = cursor.All(context.Background(), &results); err != nil {
		return fmt.Errorf("Failed to list indexes: %s", err)
	}

	found := false
	var keyValues []attr.Value
	for _, result := range results {
		if name, ok := result["name"]; !ok || name != indexName {
			continue
		}

		if keyD, ok := result["key"].(bson.D); ok {
			for _, elem := range keyD {
				keyValues = append(keyValues, mustIndexKeyObject(elem.Key, fmt.Sprintf("%v", elem.Value)))
			}
		}
		if unique, ok := result["unique"]; ok {
			keyValues = append(keyValues, mustIndexKeyObject("unique", fmt.Sprintf("%v", unique)))
		}
		if sparse, ok := result["sparse"]; ok {
			keyValues = append(keyValues, mustIndexKeyObject("sparse", fmt.Sprintf("%v", sparse)))
		}
		if expireAfter, ok := result["expireAfterSeconds"]; ok {
			keyValues = append(keyValues, mustIndexKeyObject("expireAfterSeconds", fmt.Sprintf("%v", expireAfter)))
		}
		if pfe, ok := result["partialFilterExpression"]; ok {
			if pfeBytes, marshalErr := bson.MarshalExtJSON(pfe, false, false); marshalErr == nil {
				m.PartialFilterExpression = types.StringValue(string(pfeBytes))
			}
		} else {
			// No partial filter on the index: pin to "" (the default) so state
			// matches create-time and import round-trips without a diff.
			m.PartialFilterExpression = types.StringValue("")
		}
		if hidden, ok := result["hidden"]; ok {
			if hiddenBool, isBool := hidden.(bool); isBool {
				m.Hidden = types.BoolValue(hiddenBool)
			}
		} else {
			m.Hidden = types.BoolValue(false)
		}

		found = true
		break
	}

	if !found {
		return fmt.Errorf("index does not exist")
	}

	keysList, diags := types.ListValue(dbIndexKeyObjectType, keyValues)
	if diags.HasError() {
		return fmt.Errorf("building keys list")
	}

	m.Db = types.StringValue(db)
	m.Collection = types.StringValue(collectionName)
	m.Name = types.StringValue(indexName)
	m.Keys = keysList
	return nil
}

func mustIndexKeyObject(field, value string) attr.Value {
	obj, _ := types.ObjectValue(dbIndexKeyObjectType.AttrTypes, map[string]attr.Value{
		"field": types.StringValue(field),
		"value": types.StringValue(value),
	})
	return obj
}
