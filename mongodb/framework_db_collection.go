package mongodb

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type dbCollectionResourceModel struct {
	ID                           types.String `tfsdk:"id"`
	Db                           types.String `tfsdk:"db"`
	Name                         types.String `tfsdk:"name"`
	DeletionProtection           types.Bool   `tfsdk:"deletion_protection"`
	ChangeStreamPreAndPostImages types.Bool   `tfsdk:"change_stream_pre_and_post_images"`
}

type dbCollectionResource struct {
	config *MongoDatabaseConfiguration
}

func newDBCollectionResource() resource.Resource { return &dbCollectionResource{} }

var (
	_ resource.Resource                = &dbCollectionResource{}
	_ resource.ResourceWithConfigure   = &dbCollectionResource{}
	_ resource.ResourceWithImportState = &dbCollectionResource{}
)

func (r *dbCollectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_db_collection"
}

func (r *dbCollectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"deletion_protection": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"change_stream_pre_and_post_images": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
		},
	}
}

func (r *dbCollectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dbCollectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dbCollectionResourceModel
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
	collectionName := plan.Name.ValueString()
	dbClient := client.Database(db)

	if err := dbClient.CreateCollection(context.Background(), collectionName); err != nil {
		resp.Diagnostics.AddError("Could not create the collection", err.Error())
		return
	}

	if plan.ChangeStreamPreAndPostImages.ValueBool() {
		if diags := setChangeStreamPreAndPostImages(dbClient, collectionName, true); diags.HasError() {
			resp.Diagnostics.AddError("Could not set change stream pre and post images", diags[0].Summary)
			return
		}
	}

	id := base64.StdEncoding.EncodeToString([]byte(db + "." + collectionName))
	state := plan
	if err := r.readCollectionInto(client, id, &state); err != nil {
		resp.Diagnostics.AddError("Error reading collection after create", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dbCollectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dbCollectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := MongoClientInit(r.config)
	if err != nil {
		resp.Diagnostics.AddError("Error connecting to db", err.Error())
		return
	}

	// deletion_protection is a client-side flag, not stored in mongo; preserve it.
	prevDeletionProtection := state.DeletionProtection
	if err := r.readCollectionInto(client, state.ID.ValueString(), &state); err != nil {
		resp.Diagnostics.AddError("Error reading collection", err.Error())
		return
	}
	if prevDeletionProtection.IsNull() || prevDeletionProtection.IsUnknown() {
		state.DeletionProtection = types.BoolValue(true)
	} else {
		state.DeletionProtection = prevDeletionProtection
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dbCollectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dbCollectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := MongoClientInit(r.config)
	if err != nil {
		resp.Diagnostics.AddError("Error connecting to db", err.Error())
		return
	}

	db, collectionName, err := resourceDatabaseCollectionParseId(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("ID mismatch", err.Error())
		return
	}
	dbClient := client.Database(db)

	if diags := setChangeStreamPreAndPostImages(dbClient, collectionName, plan.ChangeStreamPreAndPostImages.ValueBool()); diags.HasError() {
		resp.Diagnostics.AddError("Could not set change stream pre and post images", diags[0].Summary)
		return
	}

	state := plan
	if err := r.readCollectionInto(client, plan.ID.ValueString(), &state); err != nil {
		resp.Diagnostics.AddError("Error reading collection after update", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dbCollectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dbCollectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError("Deletion protection enabled", "Can't delete collection because deletion protection is enabled")
		return
	}

	client, err := MongoClientInit(r.config)
	if err != nil {
		resp.Diagnostics.AddError("Error connecting to db", err.Error())
		return
	}

	db, collectionName, err := resourceDatabaseCollectionParseId(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("ID mismatch", err.Error())
		return
	}
	if err := client.Database(db).Collection(collectionName).Drop(context.Background()); err != nil {
		resp.Diagnostics.AddError("Could not delete the collection", err.Error())
		return
	}
}

func (r *dbCollectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// readCollectionInto populates id, db, name and change_stream_pre_and_post_images
// from the database. deletion_protection is a client-side flag preserved by the
// caller (mirrors the SDKv2 read).
func (r *dbCollectionResource) readCollectionInto(client *mongo.Client, id string, m *dbCollectionResourceModel) error {
	db, collectionName, err := resourceDatabaseCollectionParseId(id)
	if err != nil {
		return err
	}

	dbClient := client.Database(db)
	cursor, err := dbClient.ListCollections(context.Background(), bson.M{"name": collectionName})
	if err != nil {
		return fmt.Errorf("failed to list collections : %s", err)
	}
	if !cursor.Next(context.Background()) {
		return fmt.Errorf("collection does not exist")
	}

	var collectionSpec *mongo.CollectionSpecification
	if err := cursor.Decode(&collectionSpec); err != nil {
		return fmt.Errorf("failed to decode collection specification : %s", err)
	}

	changeStreamEnabled := false
	if doc, ok := collectionSpec.Options.Lookup("changeStreamPreAndPostImages").DocumentOK(); ok {
		if enabled, ok := doc.Lookup("enabled").BooleanOK(); ok {
			changeStreamEnabled = enabled
		}
	}

	m.ID = types.StringValue(id)
	m.Db = types.StringValue(db)
	m.Name = types.StringValue(collectionName)
	m.ChangeStreamPreAndPostImages = types.BoolValue(changeStreamEnabled)
	return nil
}
