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
	_ list.ListResource              = &dbUserListResource{}
	_ list.ListResourceWithConfigure = &dbUserListResource{}
)

func newDBUserListResource() list.ListResource { return &dbUserListResource{} }

type dbUserListResource struct {
	config *MongoDatabaseConfiguration
}

type dbUserIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

func (r *dbUserListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_db_user"
}

func (r *dbUserListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "Lists all MongoDB database users. Use with `terraform query` (Terraform 1.14 and later).",
	}
}

func (r *dbUserListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dbUserListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	client, err := MongoClientInit(r.config)
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Error connecting to database", err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	cmd := bson.D{{Key: "usersInfo", Value: bson.D{{Key: "forAllDBs", Value: true}}}}
	var decoded SingleResultGetUser
	if err := client.Database("admin").RunCommand(ctx, cmd).Decode(&decoded); err != nil {
		var diags diag.Diagnostics
		diags.AddError("Failed to list users", err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, u := range decoded.Users {
			id := base64.StdEncoding.EncodeToString([]byte(u.Db + "." + u.User))
			result := req.NewListResult(ctx)
			result.DisplayName = u.User
			result.Diagnostics.Append(result.Identity.Set(ctx, dbUserIdentityModel{ID: types.StringValue(id)})...)
			if !push(result) {
				return
			}
		}
	}
}
