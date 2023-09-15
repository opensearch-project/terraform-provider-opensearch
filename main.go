package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/opensearch-project/terraform-provider-opensearch/provider"
)

// Generate docs for website
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debuggable", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	if debugMode {
		//nolint:staticcheck // SA1019 ignore this!
		err := plugin.Debug(context.Background(), "registry.terraform.io/opensearch-project/opensearch",
			&plugin.ServeOpts{
				ProviderFunc: provider.Provider,
			},
		)
		if err != nil {
			log.Println(err.Error())
		}
	} else {
		plugin.Serve(&plugin.ServeOpts{
			ProviderFunc: provider.Provider,
		})
	}
}
