package provider

import (
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"zabbix": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ *schema.Provider = Provider()
}

func testAccPreCheck(t *testing.T) {

	required := []string{"ZABBIX_URL", "ZABBIX_USER", "ZABBIX_PASS"}

	for _, envName := range required {
		if err := os.Getenv(envName); err == "" {
			t.Fatalf("environment variable %s must be set", envName)
		}
	}
}

var (
	testAccVersionOnce sync.Once
	testAccVersionVal  int
	testAccVersionErr  error
)

// testAccVersion returns the encoded version (see parseVersionString) of the
// server under test, probed once per process. apiinfo.version needs no
// credentials, so this does not depend on the provider being configured.
// Returns 0 if ZABBIX_URL is unset, which is the case for the unit tests.
func testAccVersion(t *testing.T) int {
	testAccVersionOnce.Do(func() {
		url := os.Getenv("ZABBIX_URL")
		if url == "" {
			return
		}
		api, err := zabbix.NewAPI(zabbix.Config{Url: url})
		if err != nil {
			testAccVersionErr = err
			return
		}
		testAccVersionVal = api.Config.Version
	})
	if testAccVersionErr != nil {
		t.Fatalf("could not determine Zabbix version: %s", testAccVersionErr)
	}
	return testAccVersionVal
}

// testAccTemplateGroups reports whether the server under test has the separate
// templategroup API introduced in Zabbix 6.2.
func testAccTemplateGroups(t *testing.T) bool {
	return testAccVersion(t) >= zabbix.V62
}

// hcl adapts a test configuration to the server under test.
//
// Templates belong to *template* groups from Zabbix 6.2 onwards and to *host*
// groups below that, so a config that works on 7.x does not work on 6.0 and
// vice versa. Rather than carry two copies of every fixture, tests are written
// against zabbix_templategroup — the current model — and this rewrites those
// references to zabbix_hostgroup when the server predates 6.2, where the two
// are the same object.
//
// The rewrite is purely textual, so a config that needs both must give them
// distinct resource labels and distinct group names; the shared fixtures use
// "testtmplgrp"/"test-template-group" for the template group and
// "testgrp"/"test-group" for the host group.
func hcl(t *testing.T, config string) string {
	if testAccTemplateGroups(t) {
		return config
	}
	return strings.ReplaceAll(config, "zabbix_templategroup", "zabbix_hostgroup")
}

// tmplGroupAddr is the Terraform address a template-group fixture ends up with
// once hcl() has rewritten it, so that a TestCheckResourceAttr* can name it.
// hcl() only rewrites the configuration string; check functions take addresses
// as literals and would otherwise look for a zabbix_templategroup that does not
// exist in state below 6.2.
func tmplGroupAddr(t *testing.T, label string) string {
	if testAccTemplateGroups(t) {
		return "zabbix_templategroup." + label
	}
	return "zabbix_hostgroup." + label
}
