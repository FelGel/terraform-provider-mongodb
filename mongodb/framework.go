package mongodb

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
)

// MuxServerFactory builds the protocol-6 mux server that fronts both provider
// halves: the SDKv2 provider (upgraded 5->6) and the terraform-plugin-framework
// provider. main.go, the acceptance-test factory, and the schema-match test all
// go through this so they exercise the same wiring.
func MuxServerFactory(ctx context.Context) (func() tfprotov6.ProviderServer, error) {
	upgradedSDK, err := tf5to6server.UpgradeServer(ctx, Provider().GRPCProvider)
	if err != nil {
		return nil, err
	}
	providers := []func() tfprotov6.ProviderServer{
		func() tfprotov6.ProviderServer { return upgradedSDK },
		providerserver.NewProtocol6(NewFrameworkProvider()),
	}
	muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		return nil, err
	}
	return muxServer.ProviderServer, nil
}

// frameworkProvider is the terraform-plugin-framework half of the provider.
// Its configuration schema must stay identical to the SDKv2 provider schema
// (a mux requirement); only resources migrated to the framework live here.
type frameworkProvider struct{}

func NewFrameworkProvider() provider.Provider { return &frameworkProvider{} }

func (p *frameworkProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "mongodb"
}

func (p *frameworkProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"connection_string":    schema.StringAttribute{Optional: true, Sensitive: true, Description: "The mongodb server connection string"},
			"host":                 schema.StringAttribute{Optional: true, Description: "The mongodb server address"},
			"port":                 schema.StringAttribute{Optional: true, Description: "The mongodb server port"},
			"certificate":          schema.StringAttribute{Optional: true, Description: "PEM-encoded content of Mongodb host CA certificate"},
			"username":             schema.StringAttribute{Optional: true, Description: "The mongodb user"},
			"password":             schema.StringAttribute{Optional: true, Sensitive: true, Description: "The mongodb password"},
			"auth_database":        schema.StringAttribute{Optional: true, Description: "The mongodb auth database"},
			"replica_set":          schema.StringAttribute{Optional: true, Description: "The mongodb replica set"},
			"insecure_skip_verify": schema.BoolAttribute{Optional: true, Description: "ignore hostname verification"},
			"tls":                  schema.BoolAttribute{Optional: true, Description: "TLS activation"},
			"direct":               schema.BoolAttribute{Optional: true, Description: "enforces a direct connection instead of discovery"},
			"retrywrites":          schema.BoolAttribute{Optional: true, Description: "Retryable Writes"},
			"proxy":                schema.StringAttribute{Optional: true},
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
	}

	mc := &MongoDatabaseConfiguration{Config: &clientConfig, MaxConnLifetime: 10}
	resp.ResourceData = mc
}

func (p *frameworkProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newDBUserResource,
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
