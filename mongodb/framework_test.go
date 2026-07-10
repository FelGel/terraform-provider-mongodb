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
	if _, ok := resp.ResourceSchemas["mongodb_db_user"]; !ok {
		t.Error("mongodb_db_user not present in muxed provider schema")
	}
}

// TestDBUserStateShapeUnchanged is a state-compatibility guard for the SDKv2 ->
// framework migration. A true registry-based upgrade test isn't possible (this
// fork is unpublished), so instead we assert the framework db_user schema has
// the same attribute names and types as the retained SDKv2 schema. If they
// diverge, existing state written by the SDKv2 resource would not round-trip
// through the framework resource (spurious diffs / plan errors on upgrade).
func TestDBUserStateShapeUnchanged(t *testing.T) {
	ctx := context.Background()

	sdkType := resourceDatabaseUser().CoreConfigSchema().ImpliedType()
	sdkKinds := map[string]string{}
	for name, at := range sdkType.AttributeTypes() {
		sdkKinds[name] = ctyKind(at)
	}

	var resp resource.SchemaResponse
	newDBUserResource().Schema(ctx, resource.SchemaRequest{}, &resp)
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
		if _, present := sdkKinds[name]; !present {
			t.Errorf("framework has attribute %q absent from SDKv2 schema — new state field", name)
		}
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
