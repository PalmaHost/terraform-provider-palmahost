// Terraform provider for PalmaHost Cloud.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/palmahost/terraform-provider-palmahost/internal/provider"
)

// version is overwritten at release time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with debugger support")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/palmahost/palmahost",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
