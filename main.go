package main

import (
	"context"
	"log"

	"github.com/FelGel/terraform-provider-mongodb/mongodb"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
)

const providerAddr = "registry.terraform.io/FelGel/mongodb"

// The provider is mid-migration from terraform-plugin-sdk/v2 to
// terraform-plugin-framework. Both halves are served together through a
// protocol-6 mux server (see mongodb.MuxServerFactory).
func main() {
	ctx := context.Background()

	serverFactory, err := mongodb.MuxServerFactory(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if err := tf6server.Serve(providerAddr, serverFactory); err != nil {
		log.Fatal(err)
	}
}
