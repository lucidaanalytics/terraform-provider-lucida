// terraform-provider-lucida manages Lucida project definitions (flags, experiments, cohorts,
// dashboards, alerts, keys, export schedules, ...) from Terraform. Every resource is generated
// from lucida/internal/defsync, the model behind `lucida pull|plan|apply`, so the two cannot drift.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"lucida/cmd/terraform-provider-lucida/internal/provider"
)

// version is set by goreleaser.
var version = "dev"

func main() {
	debug := flag.Bool("debug", false, "serve with debugger support")
	flag.Parse()
	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/lucidaanalytics/lucida",
		Debug:   *debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
