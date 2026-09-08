package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/withakedo/terraform-provider-i-doit/internal/provider"
)

// version is overridden at release time via -ldflags "-X main.version=...".
var version = "dev"

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name i-doit

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		// registry.terraform.io/withakedo/i-doit
		Address: "registry.terraform.io/withakedo/i-doit",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err.Error())
	}
}
