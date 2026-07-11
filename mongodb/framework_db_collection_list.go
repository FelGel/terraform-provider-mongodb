package mongodb

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	_ list.ListResource              = &dbCollectionListResource{}
	_ list.ListResourceWithConfigure = &dbCollectionListResource{}
)

func newDBCollectionListResource() list.ListResource { return &dbCollectionListResource{} }

type dbCollectionListResource struct {
	config *MongoDatabaseConfiguration
}

func (r *dbCollectionListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_db_collection"
}

func (r *dbCollectionListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "Lists all MongoDB collections across databases. Use with `terraform query` (Terraform 1.14 and later).",
	}
}

func (r *dbCollectionListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dbCollectionListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	client, err := MongoClientInit(r.config)
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Error connecting to database", err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	dbNames, err := client.ListDatabaseNames(ctx, bson.D{})
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Failed to list databases", err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, dbName := range dbNames {
			collNames, err := client.Database(dbName).ListCollectionNames(ctx, bson.D{})
			if err != nil {
				continue // skip databases we can't list collections for
			}
			for _, coll := range collNames {
				id := base64.StdEncoding.EncodeToString([]byte(dbName + "." + coll))
				result := req.NewListResult(ctx)
				result.DisplayName = coll
				result.Diagnostics.Append(result.Identity.Set(ctx, dbUserIdentityModel{ID: types.StringValue(id)})...)
				if !push(result) {
					return
				}
			}
		}
	}
}
