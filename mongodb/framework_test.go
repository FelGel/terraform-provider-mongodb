package mongodb

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
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
