package mongodb

import (
	"context"
	"os"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// frameworkProvider is the MongoDB provider, built on terraform-plugin-framework.
type frameworkProvider struct{}

func NewFrameworkProvider() provider.Provider { return &frameworkProvider{} }

func (p *frameworkProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "mongodb"
}

func (p *frameworkProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The MongoDB provider is used to interact with the resources supported by [MongoDB](https://www.mongodb.com/). The provider needs to be configured with the proper credentials before it can be used.",
		Attributes: map[string]schema.Attribute{
			"connection_string": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "MongoDB connection string (`mongodb://...`). When set it is used instead of `host` and `port`; the `retrywrites`, `tls`, `insecure_skip_verify`, `replica_set` and `direct` options are still appended to it.",
			},
			"host": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Host of your MongoDB server. Can also be sourced from the `MONGO_HOST` environment variable. Defaults to `127.0.0.1`.",
			},
			"port": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Port of your MongoDB server. Can also be sourced from the `MONGO_PORT` environment variable. Defaults to `27017`.",
			},
			"certificate": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "PEM-encoded content of the CA certificate used to verify the MongoDB server over TLS (for example `file(\"ca.pem\")`). Can also be sourced from the `MONGODB_CERT` environment variable.",
			},
			"username": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Username with which to authenticate to the MongoDB database. Can also be sourced from the `MONGO_USR` environment variable.",
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Password with which to authenticate to the MongoDB database. Can also be sourced from the `MONGO_PWD` environment variable.",
			},
			"auth_database": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Authentication database where the specified `username` has been created. Defaults to `admin`. Use `$external` with the `MONGODB-X509`, `MONGODB-AWS` and `MONGODB-OIDC` mechanisms.",
			},
			"replica_set": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Name of the replica set to connect to. Ignored when `direct` is `true`.",
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Set to `true` to skip TLS certificate and hostname verification. Defaults to `false`.",
			},
			"tls": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Set to `true` to connect to the deployment using TLS/SSL. Defaults to `false`.",
			},
			"direct": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Set to `true` to force a direct connection to `host` instead of discovering the topology. Defaults to `false`.",
			},
			"retrywrites": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Retryable writes let the driver automatically retry certain write operations once if they hit network errors or cannot find a healthy primary in the replica set or sharded cluster. Defaults to `true`.",
			},
			"proxy": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "SOCKS5 proxy URL to connect through, e.g. `socks5://myproxy:8080` (`socks5h://` is also accepted). Can also be sourced from the `ALL_PROXY` or `all_proxy` environment variable.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^socks5h?://.*:\d+$`), "The proxy URL is not a valid socks url."),
				},
			},
			"auth_mechanism": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "SASL authentication mechanism the provider uses for its own connection, e.g. `SCRAM-SHA-256`, `MONGODB-X509`, `MONGODB-AWS` or `MONGODB-OIDC`. When empty the driver negotiates SCRAM. Can also be sourced from the `MONGO_AUTH_MECHANISM` environment variable. Mechanisms other than SCRAM authenticate against `$external`, so set `auth_database = \"$external\"` for `MONGODB-X509`, `MONGODB-AWS` and `MONGODB-OIDC`.",
			},
			"auth_mechanism_properties": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Additional properties for the selected `auth_mechanism`. For `MONGODB-OIDC` these are the OIDC properties such as `ENVIRONMENT` (e.g. `gcp`, `azure`) and `TOKEN_RESOURCE`; for `MONGODB-AWS`, `AWS_SESSION_TOKEN`.",
			},
		},
	}
}

type frameworkProviderModel struct {
	ConnectionString   types.String `tfsdk:"connection_string"`
	Host               types.String `tfsdk:"host"`
	Port               types.String `tfsdk:"port"`
	Certificate        types.String `tfsdk:"certificate"`
	Username           types.String `tfsdk:"username"`
	Password           types.String `tfsdk:"password"`
	AuthDatabase       types.String `tfsdk:"auth_database"`
	ReplicaSet         types.String `tfsdk:"replica_set"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
	Tls                types.Bool   `tfsdk:"tls"`
	Direct             types.Bool   `tfsdk:"direct"`
	RetryWrites        types.Bool   `tfsdk:"retrywrites"`
	Proxy              types.String `tfsdk:"proxy"`
	AuthMechanism      types.String `tfsdk:"auth_mechanism"`
	AuthMechanismProps types.Map    `tfsdk:"auth_mechanism_properties"`
}

func (p *frameworkProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg frameworkProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Replicate the SDKv2 DefaultFunc / Default behavior, since the framework
	// has no schema-level defaults.
	clientConfig := ClientConfig{
		ConnectionString:   cfg.ConnectionString.ValueString(),
		Host:               strDefault(cfg.Host, envDefault("MONGO_HOST", "127.0.0.1")),
		Port:               strDefault(cfg.Port, envDefault("MONGO_PORT", "27017")),
		Certificate:        strDefault(cfg.Certificate, envDefault("MONGODB_CERT", "")),
		Username:           strDefault(cfg.Username, envDefault("MONGO_USR", "")),
		Password:           strDefault(cfg.Password, envDefault("MONGO_PWD", "")),
		DB:                 strDefault(cfg.AuthDatabase, "admin"),
		ReplicaSet:         cfg.ReplicaSet.ValueString(),
		Tls:                cfg.Tls.ValueBool(),
		InsecureSkipVerify: cfg.InsecureSkipVerify.ValueBool(),
		Direct:             cfg.Direct.ValueBool(),
		RetryWrites:        boolDefault(cfg.RetryWrites, true),
		Proxy:              strDefault(cfg.Proxy, envMultiDefault("ALL_PROXY", "all_proxy")),
		AuthMechanism:      strDefault(cfg.AuthMechanism, envDefault("MONGO_AUTH_MECHANISM", "")),
	}

	if !cfg.AuthMechanismProps.IsNull() && !cfg.AuthMechanismProps.IsUnknown() {
		props := make(map[string]string, len(cfg.AuthMechanismProps.Elements()))
		resp.Diagnostics.Append(cfg.AuthMechanismProps.ElementsAs(ctx, &props, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		clientConfig.AuthMechanismProperties = props
	}

	mc := &MongoDatabaseConfiguration{Config: &clientConfig, MaxConnLifetime: 10}
	resp.ResourceData = mc
	// List resources receive provider data from a separate field.
	resp.ListResourceData = mc
}

func (p *frameworkProvider) ListResources(_ context.Context) []func() list.ListResource {
	return []func() list.ListResource{
		newDBUserListResource,
		newDBRoleListResource,
		newDBCollectionListResource,
		newDBIndexListResource,
	}
}

func (p *frameworkProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newDBUserResource,
		newDBRoleResource,
		newDBCollectionResource,
		newDBIndexResource,
	}
}

func (p *frameworkProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func strDefault(v types.String, def string) string {
	if v.IsNull() || v.IsUnknown() {
		return def
	}
	return v.ValueString()
}

func boolDefault(v types.Bool, def bool) bool {
	if v.IsNull() || v.IsUnknown() {
		return def
	}
	return v.ValueBool()
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envMultiDefault(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}
