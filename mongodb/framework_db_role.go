package mongodb

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var dbRolePrivilegeObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"db":         types.StringType,
	"collection": types.StringType,
	"actions":    types.ListType{ElemType: types.StringType},
}}

var dbRoleInheritedObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"db":   types.StringType,
	"role": types.StringType,
}}

type dbRoleResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Database       types.String `tfsdk:"database"`
	Name           types.String `tfsdk:"name"`
	Privileges     types.Set    `tfsdk:"privilege"`
	InheritedRoles types.Set    `tfsdk:"inherited_role"`
}

type dbRolePrivilegeModel struct {
	Db         types.String `tfsdk:"db"`
	Collection types.String `tfsdk:"collection"`
	Actions    types.List   `tfsdk:"actions"`
}

type dbRoleInheritedModel struct {
	Db   types.String `tfsdk:"db"`
	Role types.String `tfsdk:"role"`
}

type dbRoleResource struct {
	config *MongoDatabaseConfiguration
}

func newDBRoleResource() resource.Resource { return &dbRoleResource{} }

var (
	_ resource.Resource                = &dbRoleResource{}
	_ resource.ResourceWithConfigure   = &dbRoleResource{}
	_ resource.ResourceWithImportState = &dbRoleResource{}
)

func (r *dbRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_db_role"
}

func (r *dbRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"database": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("admin"),
			},
			"name": schema.StringAttribute{
				Required: true,
			},
		},
		Blocks: map[string]schema.Block{
			"privilege": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"db":         schema.StringAttribute{Optional: true},
						"collection": schema.StringAttribute{Optional: true},
						"actions": schema.ListAttribute{
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"inherited_role": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"db":   schema.StringAttribute{Optional: true},
						"role": schema.StringAttribute{Required: true},
					},
				},
			},
		},
	}
}

func (r *dbRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dbRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dbRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := MongoClientInit(r.config)
	if err != nil {
		resp.Diagnostics.AddError("Error connecting to database", err.Error())
		return
	}

	roleName := plan.Name.ValueString()
	database := plan.Database.ValueString()
	roleList, diags := inheritedFromSet(ctx, plan.InheritedRoles)
	resp.Diagnostics.Append(diags...)
	privileges, diags := privilegesFromSet(ctx, plan.Privileges)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := createRole(client, roleName, roleList, privileges, database); err != nil {
		resp.Diagnostics.AddError("Could not create the role", err.Error())
		return
	}

	id := base64.StdEncoding.EncodeToString([]byte(database + "." + roleName))
	var state dbRoleResourceModel
	if err := r.readRoleInto(client, id, &state); err != nil {
		resp.Diagnostics.AddError("Error reading role after create", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dbRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dbRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := MongoClientInit(r.config)
	if err != nil {
		resp.Diagnostics.AddError("Error connecting to database", err.Error())
		return
	}

	if err := r.readRoleInto(client, state.ID.ValueString(), &state); err != nil {
		resp.Diagnostics.AddError("Error reading role", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dbRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state dbRoleResourceModel
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

	// Mirror the SDKv2 update: drop the existing role, then recreate it.
	oldRole, oldDatabase, err := resourceDatabaseRoleParseId(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("ID mismatch", err.Error())
		return
	}
	if result := client.Database(oldDatabase).RunCommand(ctx, bson.D{{Key: "dropRole", Value: oldRole}}); result.Err() != nil {
		resp.Diagnostics.AddError("Could not update the role", result.Err().Error())
		return
	}

	roleName := plan.Name.ValueString()
	database := plan.Database.ValueString()
	roleList, diags := inheritedFromSet(ctx, plan.InheritedRoles)
	resp.Diagnostics.Append(diags...)
	privileges, diags := privilegesFromSet(ctx, plan.Privileges)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := createRole(client, roleName, roleList, privileges, database); err != nil {
		resp.Diagnostics.AddError("Could not update the role", err.Error())
		return
	}

	id := base64.StdEncoding.EncodeToString([]byte(database + "." + roleName))
	var newState dbRoleResourceModel
	if err := r.readRoleInto(client, id, &newState); err != nil {
		resp.Diagnostics.AddError("Error reading role after update", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *dbRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dbRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := MongoClientInit(r.config)
	if err != nil {
		resp.Diagnostics.AddError("Error connecting to database", err.Error())
		return
	}

	roleName, database, err := resourceDatabaseRoleParseId(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("ID mismatch", err.Error())
		return
	}
	if result := client.Database(database).RunCommand(ctx, bson.D{{Key: "dropRole", Value: roleName}}); result.Err() != nil {
		resp.Diagnostics.AddError("Could not delete the role", result.Err().Error())
		return
	}
}

func (r *dbRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *dbRoleResource) readRoleInto(client *mongo.Client, id string, m *dbRoleResourceModel) error {
	roleName, database, err := resourceDatabaseRoleParseId(id)
	if err != nil {
		return err
	}
	result, err := getRole(client, roleName, database)
	if err != nil {
		return err
	}
	if len(result.Roles) == 0 {
		return fmt.Errorf("role does not exist")
	}

	inheritedValues := make([]attr.Value, 0, len(result.Roles[0].InheritedRoles))
	for _, s := range result.Roles[0].InheritedRoles {
		obj, diags := types.ObjectValue(dbRoleInheritedObjectType.AttrTypes, map[string]attr.Value{
			"db":   types.StringValue(s.Db),
			"role": types.StringValue(s.Role),
		})
		if diags.HasError() {
			return fmt.Errorf("building inherited_role value")
		}
		inheritedValues = append(inheritedValues, obj)
	}
	inheritedSet, diags := types.SetValue(dbRoleInheritedObjectType, inheritedValues)
	if diags.HasError() {
		return fmt.Errorf("building inherited_role set")
	}

	privilegeValues := make([]attr.Value, 0, len(result.Roles[0].Privileges))
	for _, s := range result.Roles[0].Privileges {
		// Sort actions for a stable set representation (matches SDKv2 read).
		actions := make([]string, len(s.Actions))
		copy(actions, s.Actions)
		sort.Strings(actions)
		actionValues := make([]attr.Value, 0, len(actions))
		for _, a := range actions {
			actionValues = append(actionValues, types.StringValue(a))
		}
		actionsList, diags := types.ListValue(types.StringType, actionValues)
		if diags.HasError() {
			return fmt.Errorf("building privilege actions list")
		}
		obj, diags := types.ObjectValue(dbRolePrivilegeObjectType.AttrTypes, map[string]attr.Value{
			"db":         types.StringValue(s.Resource.Db),
			"collection": types.StringValue(s.Resource.Collection),
			"actions":    actionsList,
		})
		if diags.HasError() {
			return fmt.Errorf("building privilege value")
		}
		privilegeValues = append(privilegeValues, obj)
	}
	privilegeSet, diags := types.SetValue(dbRolePrivilegeObjectType, privilegeValues)
	if diags.HasError() {
		return fmt.Errorf("building privilege set")
	}

	m.ID = types.StringValue(id)
	m.Name = types.StringValue(roleName)
	m.Database = types.StringValue(database)
	m.InheritedRoles = inheritedSet
	m.Privileges = privilegeSet
	return nil
}

func inheritedFromSet(ctx context.Context, set types.Set) ([]Role, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}
	var models []dbRoleInheritedModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}
	roles := make([]Role, 0, len(models))
	for _, m := range models {
		roles = append(roles, Role{Role: m.Role.ValueString(), Db: m.Db.ValueString()})
	}
	return roles, diags
}

func privilegesFromSet(ctx context.Context, set types.Set) ([]PrivilegeDto, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}
	var models []dbRolePrivilegeModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}
	privileges := make([]PrivilegeDto, 0, len(models))
	for _, m := range models {
		var actions []string
		if !m.Actions.IsNull() && !m.Actions.IsUnknown() {
			diags.Append(m.Actions.ElementsAs(ctx, &actions, false)...)
		}
		privileges = append(privileges, PrivilegeDto{
			Db:         m.Db.ValueString(),
			Collection: m.Collection.ValueString(),
			Actions:    actions,
		})
	}
	return privileges, diags
}
