package mongodb

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var dbUserRoleObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"db":   types.StringType,
	"role": types.StringType,
}}

type dbUserResourceModel struct {
	ID            types.String `tfsdk:"id"`
	AuthDatabase  types.String `tfsdk:"auth_database"`
	Name          types.String `tfsdk:"name"`
	Password      types.String `tfsdk:"password"`
	AuthMechanism types.String `tfsdk:"auth_mechanism"`
	Roles         types.Set    `tfsdk:"role"`
}

type dbUserRoleModel struct {
	Db   types.String `tfsdk:"db"`
	Role types.String `tfsdk:"role"`
}

type dbUserResource struct {
	config *MongoDatabaseConfiguration
}

func newDBUserResource() resource.Resource { return &dbUserResource{} }

var (
	_ resource.Resource                = &dbUserResource{}
	_ resource.ResourceWithConfigure   = &dbUserResource{}
	_ resource.ResourceWithImportState = &dbUserResource{}
	_ resource.ResourceWithModifyPlan  = &dbUserResource{}
)

func (r *dbUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_db_user"
}

func (r *dbUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"auth_database": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"password": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"auth_mechanism": schema.StringAttribute{
				Optional: true,
			},
		},
		Blocks: map[string]schema.Block{
			"role": schema.SetNestedBlock{
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

func (r *dbUserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan ports the SDKv2 customizeDiffDBUser cross-field logic.
func (r *dbUserResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return // destroy
	}
	var plan dbUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	authMechanism := plan.AuthMechanism.ValueString()
	name := plan.Name.ValueString()

	if authMechanism != "" && authMechanism != "MONGODB-AWS" {
		resp.Diagnostics.AddError("Invalid db_user configuration", fmt.Sprintf(`auth_mechanism must be "MONGODB-AWS" or empty; got %q`, authMechanism))
		return
	}

	// Suppress password diffs for IAM users on update: password is irrelevant for
	// MONGODB-AWS, so pin it to prior state instead of erroring.
	if authMechanism == "MONGODB-AWS" && !req.State.Raw.IsNull() {
		var state dbUserResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Password = state.Password
	}

	password := plan.Password.ValueString()
	if err := validateDBUserDiff(authMechanism, password); err != nil {
		resp.Diagnostics.AddError("Invalid db_user configuration", err.Error())
		return
	}

	if authMechanism == "MONGODB-AWS" && !iamARNRegex.MatchString(name) {
		resp.Diagnostics.AddError("Invalid db_user configuration",
			`name must be a valid IAM ARN (arn:aws:iam::<account-id>:(user|role)/<name>) when auth_mechanism is "MONGODB-AWS"`)
		return
	}

	if authMechanism != "MONGODB-AWS" && (plan.AuthDatabase.IsNull() || plan.AuthDatabase.ValueString() == "") {
		resp.Diagnostics.AddError("Invalid db_user configuration", "auth_database is required when auth_mechanism is not set")
		return
	}

	plan.AuthDatabase = types.StringValue(resolveAuthDatabase(authMechanism, plan.AuthDatabase.ValueString()))
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *dbUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dbUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := MongoClientInit(r.config)
	if err != nil {
		resp.Diagnostics.AddError("Error connecting to database", err.Error())
		return
	}

	database := plan.AuthDatabase.ValueString()
	userName := plan.Name.ValueString()
	authMechanism := plan.AuthMechanism.ValueString()
	roleList, diags := rolesFromSet(ctx, plan.Roles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if authMechanism == "MONGODB-AWS" {
		if err := createIAMUser(client, userName, roleList); err != nil {
			resp.Diagnostics.AddError("Could not create the user", err.Error())
			return
		}
	} else {
		user := DbUser{Name: userName, Password: plan.Password.ValueString()}
		if err := createUser(client, user, roleList, database); err != nil {
			resp.Diagnostics.AddError("Could not create the user", err.Error())
			return
		}
	}

	id := base64.StdEncoding.EncodeToString([]byte(database + "." + userName))
	var state dbUserResourceModel
	state.Password = knownOrEmpty(plan.Password)
	if err := r.readUserInto(client, id, &state); err != nil {
		resp.Diagnostics.AddError("Error reading user after create", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dbUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dbUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := MongoClientInit(r.config)
	if err != nil {
		resp.Diagnostics.AddError("Error connecting to database", err.Error())
		return
	}

	prevPassword := knownOrEmpty(state.Password)
	if err := r.readUserInto(client, state.ID.ValueString(), &state); err != nil {
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}
	state.Password = prevPassword
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dbUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dbUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := MongoClientInit(r.config)
	if err != nil {
		resp.Diagnostics.AddError("Error connecting to database", err.Error())
		return
	}

	database := plan.AuthDatabase.ValueString()
	userName := plan.Name.ValueString()
	authMechanism := plan.AuthMechanism.ValueString()
	roleList, diags := rolesFromSet(ctx, plan.Roles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	rolesValue := roleList
	if rolesValue == nil {
		rolesValue = []Role{}
	}

	adminDB := client.Database(database)
	var cmd bson.D
	if authMechanism == "MONGODB-AWS" {
		// IAM users: update roles only; password is ignored.
		cmd = bson.D{{Key: "updateUser", Value: userName}, {Key: "roles", Value: rolesValue}}
	} else {
		cmd = bson.D{{Key: "updateUser", Value: userName}, {Key: "pwd", Value: plan.Password.ValueString()}, {Key: "roles", Value: rolesValue}}
	}
	if result := adminDB.RunCommand(ctx, cmd); result.Err() != nil {
		resp.Diagnostics.AddError("Could not update the user", result.Err().Error())
		return
	}

	id := base64.StdEncoding.EncodeToString([]byte(database + "." + userName))
	var state dbUserResourceModel
	state.Password = knownOrEmpty(plan.Password)
	if err := r.readUserInto(client, id, &state); err != nil {
		resp.Diagnostics.AddError("Error reading user after update", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dbUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dbUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := MongoClientInit(r.config)
	if err != nil {
		resp.Diagnostics.AddError("Error connecting to database", err.Error())
		return
	}

	userName, database, err := resourceDatabaseUserParseId(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("ID mismatch", err.Error())
		return
	}
	if result := client.Database(database).RunCommand(ctx, bson.D{{Key: "dropUser", Value: userName}}); result.Err() != nil {
		resp.Diagnostics.AddError("Could not delete the user", result.Err().Error())
		return
	}
}

func (r *dbUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// readUserInto populates roles, auth_database, name, auth_mechanism and id from
// the database. Password is left to the caller (mirrors the SDKv2 read, which
// echoes the configured password rather than reading it back).
func (r *dbUserResource) readUserInto(client *mongo.Client, id string, m *dbUserResourceModel) error {
	userName, database, err := resourceDatabaseUserParseId(id)
	if err != nil {
		return err
	}
	result, err := getUser(client, userName, database)
	if err != nil {
		return err
	}
	if len(result.Users) == 0 {
		return fmt.Errorf("user does not exist")
	}

	roleValues := make([]attr.Value, 0, len(result.Users[0].Roles))
	for _, role := range result.Users[0].Roles {
		obj, diags := types.ObjectValue(dbUserRoleObjectType.AttrTypes, map[string]attr.Value{
			"db":   types.StringValue(role.Db),
			"role": types.StringValue(role.Role),
		})
		if diags.HasError() {
			return fmt.Errorf("building role value")
		}
		roleValues = append(roleValues, obj)
	}
	roleSet, diags := types.SetValue(dbUserRoleObjectType, roleValues)
	if diags.HasError() {
		return fmt.Errorf("building role set")
	}

	m.ID = types.StringValue(id)
	m.Name = types.StringValue(userName)
	m.AuthDatabase = types.StringValue(database)
	m.Roles = roleSet

	// IAM users live in $external; mechanisms isn't returned for external users.
	isIAM := database == "$external"
	for _, mech := range result.Users[0].Mechanisms {
		if mech == "MONGODB-AWS" {
			isIAM = true
			break
		}
	}
	if isIAM {
		m.AuthMechanism = types.StringValue("MONGODB-AWS")
	} else {
		m.AuthMechanism = types.StringNull()
	}
	return nil
}

func rolesFromSet(ctx context.Context, set types.Set) ([]Role, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}
	var models []dbUserRoleModel
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

func knownOrEmpty(v types.String) types.String {
	if v.IsNull() || v.IsUnknown() {
		return types.StringValue("")
	}
	return v
}
