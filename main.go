package main

import (
	"context"
	"log"

	"github.com/FelGel/terraform-provider-mongodb/mongodb"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Format the examples and generate the docs for the registry:
//go:generate terraform fmt -recursive ./examples/
//go:generate go tool tfplugindocs generate --provider-name mongodb

const providerAddr = "registry.terraform.io/FelGel/mongodb"

func main() {
	if err := providerserver.Serve(context.Background(), mongodb.NewFrameworkProvider, providerserver.ServeOpts{
		Address:         providerAddr,
		ProtocolVersion: 6,
	}); err != nil {
		log.Fatal(err)
	}
}
