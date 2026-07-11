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
	_ list.ListResource              = &dbRoleListResource{}
	_ list.ListResourceWithConfigure = &dbRoleListResource{}
)

func newDBRoleListResource() list.ListResource { return &dbRoleListResource{} }

type dbRoleListResource struct {
	config *MongoDatabaseConfiguration
}

func (r *dbRoleListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_db_role"
}

func (r *dbRoleListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "Lists all custom MongoDB roles across databases. Use with `terraform query` (Terraform 1.14 and later).",
	}
}

func (r *dbRoleListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dbRoleListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	client, err := MongoClientInit(r.config)
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Error connecting to database", err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	// rolesInfo has no forAllDBs option, so enumerate databases and query each.
	var dbs struct {
		Databases []struct{ Name string }
	}
	if err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "listDatabases", Value: 1}}).Decode(&dbs); err != nil {
		var diags diag.Diagnostics
		diags.AddError("Failed to list databases", err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, db := range dbs.Databases {
			var roles SingleResultGetRole
			if err := client.Database(db.Name).RunCommand(ctx, bson.D{{Key: "rolesInfo", Value: 1}}).Decode(&roles); err != nil {
				continue // skip databases we can't read roles for
			}
			for _, role := range roles.Roles {
				id := base64.StdEncoding.EncodeToString([]byte(role.Db + "." + role.Role))
				result := req.NewListResult(ctx)
				result.DisplayName = role.Role
				result.Diagnostics.Append(result.Identity.Set(ctx, dbUserIdentityModel{ID: types.StringValue(id)})...)
				if !push(result) {
					return
				}
			}
		}
	}
}
