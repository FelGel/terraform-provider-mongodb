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
	_ list.ListResource              = &dbIndexListResource{}
	_ list.ListResourceWithConfigure = &dbIndexListResource{}
)

func newDBIndexListResource() list.ListResource { return &dbIndexListResource{} }

type dbIndexListResource struct {
	config *MongoDatabaseConfiguration
}

func (r *dbIndexListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_db_index"
}

func (r *dbIndexListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "Lists all MongoDB indexes across databases and collections. Use with `terraform query` (Terraform 1.14 and later).",
	}
}

func (r *dbIndexListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dbIndexListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
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
				continue
			}
			for _, coll := range collNames {
				cursor, err := client.Database(dbName).Collection(coll).Indexes().List(ctx)
				if err != nil {
					continue
				}
				var idxs []struct {
					Name string `bson:"name"`
				}
				if err := cursor.All(ctx, &idxs); err != nil {
					continue
				}
				for _, idx := range idxs {
					id := base64.StdEncoding.EncodeToString([]byte(dbName + "." + coll + "." + idx.Name))
					result := req.NewListResult(ctx)
					result.DisplayName = idx.Name
					result.Diagnostics.Append(result.Identity.Set(ctx, dbUserIdentityModel{ID: types.StringValue(id)})...)
					if !push(result) {
						return
					}
				}
			}
		}
	}
}
