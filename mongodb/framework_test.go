package mongodb

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	sdkschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestMuxProviderSchema verifies the SDKv2 and framework provider halves expose
// an identical provider configuration schema (a mux requirement) and that the
// framework resource schemas are valid. Runs without a database.
func TestMuxProviderSchema(t *testing.T) {
	ctx := context.Background()
	factory, err := MuxServerFactory(ctx)
	if err != nil {
		t.Fatalf("MuxServerFactory: %s", err)
	}
	resp, err := factory().GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %s", err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("provider schema diagnostic: %s — %s", d.Summary, d.Detail)
		}
	}
	for _, typ := range []string{"mongodb_db_user", "mongodb_db_role", "mongodb_db_collection", "mongodb_db_index"} {
		if _, ok := resp.ResourceSchemas[typ]; !ok {
			t.Errorf("%s not present in muxed provider schema", typ)
		}
	}
}

// TestMuxListResources verifies the mux server actually serves the framework
// list resource(s) — i.e. that terraform-plugin-mux propagates list resources.
func TestMuxListResources(t *testing.T) {
	ctx := context.Background()
	factory, err := MuxServerFactory(ctx)
	if err != nil {
		t.Fatalf("MuxServerFactory: %s", err)
	}
	resp, err := factory().GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %s", err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("provider schema diagnostic: %s — %s", d.Summary, d.Detail)
		}
	}
	for _, typ := range []string{"mongodb_db_user", "mongodb_db_role", "mongodb_db_collection", "mongodb_db_index"} {
		if _, ok := resp.ListResourceSchemas[typ]; !ok {
			got := make([]string, 0, len(resp.ListResourceSchemas))
			for k := range resp.ListResourceSchemas {
				got = append(got, k)
			}
			t.Errorf("%s not served as a list resource through the mux; list schemas present: %v", typ, got)
		}
	}
}

// TestResourceStateShapeUnchanged is the state-compatibility guard for the
// SDKv2 -> framework migration. For each migrated resource it asserts the
// framework schema has the same attribute names and types as the retained
// SDKv2 schema. Divergence would mean existing SDKv2-written state fails to
// round-trip through the framework resource (spurious diffs / plan errors on
// upgrade). No database needed.
func TestResourceStateShapeUnchanged(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		sdk  *sdkschema.Resource
		fw   resource.Resource
		// newAttrs are framework attributes added since the SDKv2 schema
		// (additive, not a state-compat concern) and excluded from the check.
		newAttrs map[string]bool
	}{
		{"mongodb_db_user", resourceDatabaseUser(), newDBUserResource(), map[string]bool{"password_wo": true, "password_wo_version": true, "authentication_restriction": true}},
		{"mongodb_db_role", resourceDatabaseRole(), newDBRoleResource(), map[string]bool{"authentication_restriction": true}},
		{"mongodb_db_collection", resourceDatabaseCollection(), newDBCollectionResource(), nil},
		{"mongodb_db_index", resourceDatabaseIndex(), newDBIndexResource(), nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sdkKinds := map[string]string{}
			for name, at := range tc.sdk.CoreConfigSchema().ImpliedType().AttributeTypes() {
				sdkKinds[name] = ctyKind(at)
			}

			var resp resource.SchemaResponse
			tc.fw.Schema(ctx, resource.SchemaRequest{}, &resp)
			fwObj, ok := resp.Schema.Type().TerraformType(ctx).(tftypes.Object)
			if !ok {
				t.Fatal("framework schema type is not an object")
			}
			fwKinds := map[string]string{}
			for name, at := range fwObj.AttributeTypes {
				fwKinds[name] = tfKind(at)
			}

			for name, kind := range sdkKinds {
				if fwKinds[name] != kind {
					t.Errorf("attribute %q: SDKv2 shape %q, framework shape %q — state upgrade would diff", name, kind, fwKinds[name])
				}
			}
			for name := range fwKinds {
				if _, present := sdkKinds[name]; !present && !tc.newAttrs[name] {
					t.Errorf("framework has attribute %q absent from SDKv2 schema — new state field", name)
				}
			}
		})
	}
}

func ctyKind(t cty.Type) string {
	switch {
	case t.Equals(cty.String):
		return "string"
	case t.Equals(cty.Bool):
		return "bool"
	case t.Equals(cty.Number):
		return "number"
	case t.IsSetType():
		return "set(" + ctyKind(t.ElementType()) + ")"
	case t.IsListType():
		return "list(" + ctyKind(t.ElementType()) + ")"
	case t.IsObjectType():
		parts := []string{}
		for n, at := range t.AttributeTypes() {
			parts = append(parts, n+":"+ctyKind(at))
		}
		sort.Strings(parts)
		return "object{" + strings.Join(parts, ",") + "}"
	}
	return t.FriendlyName()
}

func tfKind(t tftypes.Type) string {
	switch {
	case t.Is(tftypes.String):
		return "string"
	case t.Is(tftypes.Bool):
		return "bool"
	case t.Is(tftypes.Number):
		return "number"
	case t.Is(tftypes.Set{}):
		return "set(" + tfKind(t.(tftypes.Set).ElementType) + ")"
	case t.Is(tftypes.List{}):
		return "list(" + tfKind(t.(tftypes.List).ElementType) + ")"
	case t.Is(tftypes.Object{}):
		obj := t.(tftypes.Object)
		parts := []string{}
		for n, at := range obj.AttributeTypes {
			parts = append(parts, n+":"+tfKind(at))
		}
		sort.Strings(parts)
		return "object{" + strings.Join(parts, ",") + "}"
	}
	return t.String()
}
