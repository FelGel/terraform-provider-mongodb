package mongodb

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestProviderSchema verifies the provider and resource schemas are valid and
// every resource is served. Runs without a database.
func TestProviderSchema(t *testing.T) {
	ctx := context.Background()
	resp, err := providerserver.NewProtocol6(NewFrameworkProvider())().GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
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
			t.Errorf("%s not present in provider schema", typ)
		}
	}
}

// TestListResources verifies every resource is also served as a list resource.
func TestListResources(t *testing.T) {
	ctx := context.Background()
	resp, err := providerserver.NewProtocol6(NewFrameworkProvider())().GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
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
			t.Errorf("%s not served as a list resource; list schemas present: %v", typ, got)
		}
	}
}

// sdkv2StateShapes are the attribute names and types of the SDKv2 resource
// schemas, frozen when the SDKv2 implementation was removed (state written by
// releases up to 3.x uses them).
var sdkv2StateShapes = map[string]map[string]string{
	"mongodb_db_user": {
		"auth_database":  "string",
		"auth_mechanism": "string",
		"id":             "string",
		"name":           "string",
		"password":       "string",
		"role":           "set(object{db:string,role:string})",
	},
	"mongodb_db_role": {
		"database":       "string",
		"id":             "string",
		"inherited_role": "set(object{db:string,role:string})",
		"name":           "string",
		"privilege":      "set(object{actions:list(string),collection:string,db:string})",
	},
	"mongodb_db_collection": {
		"change_stream_pre_and_post_images": "bool",
		"db":                                "string",
		"deletion_protection":               "bool",
		"id":                                "string",
		"name":                              "string",
	},
	"mongodb_db_index": {
		"collection":                "string",
		"db":                        "string",
		"hidden":                    "bool",
		"id":                        "string",
		"keys":                      "list(object{field:string,value:string})",
		"name":                      "string",
		"partial_filter_expression": "string",
		"timeout":                   "number",
	},
}

// TestResourceStateShapeUnchanged is the state-compatibility guard for the
// SDKv2 -> framework migration: each resource keeps the attribute names and
// types of its SDKv2 schema, so state written by SDKv2 releases round-trips
// without spurious diffs. New framework attributes are listed explicitly.
// No database needed.
func TestResourceStateShapeUnchanged(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		fw   resource.Resource
		// newAttrs are framework attributes added since the SDKv2 schema
		// (additive, not a state-compat concern) and excluded from the check.
		newAttrs map[string]bool
	}{
		{"mongodb_db_user", newDBUserResource(), map[string]bool{"password_wo": true, "password_wo_version": true, "authentication_restriction": true}},
		{"mongodb_db_role", newDBRoleResource(), map[string]bool{"authentication_restriction": true}},
		{"mongodb_db_collection", newDBCollectionResource(), nil},
		{"mongodb_db_index", newDBIndexResource(), nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sdkKinds := sdkv2StateShapes[tc.name]

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
