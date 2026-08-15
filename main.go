package main

import (
	"flag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/tpretz/terraform-provider-zabbix/provider"
)

// version and commit are set by the linker at release time, from the ldflags in
// .goreleaser.yml. They were declared there long before they existed here, so
// the linker silently dropped both -- a -X flag naming a variable the binary
// does not have is not an error. Anything that reads them has to tolerate the
// zero values, because a `go build` with no ldflags is the normal case for
// anyone working on the provider.
var (
	version = "dev"
	commit  = "none"
)

func main() {
	// terraform-plugin-sdk supports attaching a debugger to a running provider:
	// `go run . -debug` prints a TF_REATTACH_PROVIDERS line to export before
	// running terraform. Without this flag the only way to exercise local
	// changes is a dev_overrides block -- see DEVELOPMENT.md.
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers like delve")
	flag.Parse()

	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: provider.Provider,
		ProviderAddr: "registry.terraform.io/tpretz/zabbix",
		Debug:        debug,
	})
}
