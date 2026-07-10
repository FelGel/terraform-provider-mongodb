package main

import (
	"context"
	"log"

	"github.com/FelGel/terraform-provider-mongodb/mongodb"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
)

const providerAddr = "registry.terraform.io/FelGel/mongodb"

// The provider is mid-migration from terraform-plugin-sdk/v2 to
// terraform-plugin-framework. Both are served together through a protocol-6 mux
// server: the SDKv2 provider is upgraded from protocol 5 to 6, and framework
// providers (once added) are appended to the mux. Until then this is behavior-
// identical to serving the SDKv2 provider directly.
func main() {
	ctx := context.Background()

	upgradedSDK, err := tf5to6server.UpgradeServer(ctx, mongodb.Provider().GRPCProvider)
	if err != nil {
		log.Fatal(err)
	}

	providers := []func() tfprotov6.ProviderServer{
		func() tfprotov6.ProviderServer { return upgradedSDK },
	}

	muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		log.Fatal(err)
	}

	if err := tf6server.Serve(providerAddr, muxServer.ProviderServer); err != nil {
		log.Fatal(err)
	}
}
