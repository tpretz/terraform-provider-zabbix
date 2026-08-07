package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// E5 -- data source not-found (PLAN.md Phase 8).
//
// Looking up something that does not exist has to produce a clear failure. It
// is the cheapest guard against the class of bug that made the zabbix_template
// data source panic the provider on every single read until Phase 3c: the
// resource and the data source share one read function, so a change made for
// one of them lands on both, and a data source is never exercised by any
// resource test.
//
// There are two distinct not-found shapes and they fail in different places:
//
//   - no lookup attribute at all. The data source read functions check this
//     themselves and return a message naming the object. Terraform cannot
//     enforce it in the schema because the attributes are alternatives.
//   - a lookup that matches nothing. The shared read functions clear the id
//     and return no error, which is correct for a *resource* -- it is how
//     drift is reported, see acc_drift_test.go -- but for a data source it
//     leaves nothing to return, and Terraform rejects the null object. The
//     assertion here is that this is an error rather than a silently empty
//     data source whose attributes then flow on into whatever referenced it.
//
// Every step is an ExpectError, so nothing is created and the steps are
// independent.

// dataSourceNotFound matches the message dataSourceFound (provider/utils.go)
// produces. It names the object type and the lookup that failed, so the
// diagnostic points at the data source block rather than at whatever
// downstream resource was handed the result.
var dataSourceNotFound = regexp.MustCompile(`no \w+ found matching`)

func TestAccDataSourceNotFound(t *testing.T) {
	cases := []struct {
		what   string
		config string
		expect *regexp.Regexp
	}{
		{
			what: "hostgroup by a name nothing has",
			config: `
data "zabbix_hostgroup" "testgrp" {
	name = "test-nonexistent-group"
}
`,
			expect: dataSourceNotFound,
		},
		{
			what: "templategroup by a name nothing has",
			config: `
data "zabbix_templategroup" "testtmplgrp" {
	name = "test-nonexistent-template-group"
}
`,
			// below 6.2 the resource refuses outright with a version
			// message, which is also a clear error and also not a panic
			expect: regexp.MustCompile(dataSourceNotFound.String() + `|requires Zabbix 6.2 or later`),
		},
		{
			what: "host by a technical name nothing has",
			config: `
data "zabbix_host" "testhost" {
	host = "test-nonexistent-host"
}
`,
			expect: dataSourceNotFound,
		},
		{
			what: "host by an id nothing has",
			config: `
data "zabbix_host" "testhost" {
	hostid = "99999999"
}
`,
			expect: dataSourceNotFound,
		},
		{
			what: "template by a technical name nothing has",
			config: `
data "zabbix_template" "testtmpl" {
	host = "test-nonexistent-template"
}
`,
			expect: dataSourceNotFound,
		},
		{
			what: "proxy by a name nothing has",
			config: `
data "zabbix_proxy" "testproxy" {
	name = "test-nonexistent-proxy"
}
`,
			expect: dataSourceNotFound,
		},

		// the read functions' own guards, for a lookup with nothing to look
		// up by. These are the messages a user gets for an empty variable,
		// which is the common way to arrive here.
		{
			what: "host with no lookup attribute",
			config: `
data "zabbix_host" "testhost" {
}
`,
			expect: regexp.MustCompile("no host lookup attribute"),
		},
		{
			what: "template with no lookup attribute",
			config: `
data "zabbix_template" "testtmpl" {
}
`,
			expect: regexp.MustCompile("no filter parameters provided"),
		},
		{
			what: "proxy with no lookup attribute",
			config: `
data "zabbix_proxy" "testproxy" {
}
`,
			// unlike host and template, the proxy data source declares
			// ExactlyOneOf on its two lookup attributes, so Terraform
			// rejects this in the schema and the read function's own
			// "no proxy lookup attribute" guard is unreachable
			expect: regexp.MustCompile("one of `host,name` must be specified"),
		},
	}

	for _, c := range cases {
		t.Run(c.what, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:          func() { testAccPreCheck(t) },
				ProviderFactories: testAccProviderFactories,
				CheckDestroy:      testAccCheckAllDestroyed,
				Steps: []resource.TestStep{{
					Config:      c.config,
					ExpectError: c.expect,
				}},
			})
		})
	}
}

// TestAccDataSourceFoundStillWorks is the control. Every step of
// TestAccDataSourceNotFound expects a failure, so on its own it would pass
// just as happily against a data source that could never succeed at all.
func TestAccDataSourceFoundStillWorks(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-notfound-group"
}
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-notfound-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host   = "test-notfound-template"
}
resource "zabbix_host" "testhost" {
	host   = "test-notfound-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
resource "zabbix_proxy" "testproxy" {
	name = "test-notfound-proxy"
}

data "zabbix_hostgroup" "foundgrp" {
	name       = "test-notfound-group"
	depends_on = [zabbix_hostgroup.testgrp]
}
data "zabbix_templategroup" "foundtmplgrp" {
	name       = "test-notfound-template-group"
	depends_on = [zabbix_templategroup.testtmplgrp]
}
data "zabbix_template" "foundtmpl" {
	host       = "test-notfound-template"
	depends_on = [zabbix_template.testtmpl]
}
data "zabbix_host" "foundhost" {
	host       = "test-notfound-host"
	depends_on = [zabbix_host.testhost]
}
data "zabbix_proxy" "foundproxy" {
	name       = "test-notfound-proxy"
	depends_on = [zabbix_proxy.testproxy]
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.zabbix_hostgroup.foundgrp", "id", "zabbix_hostgroup.testgrp", "id"),
					resource.TestCheckResourceAttrPair("data."+tmplGroupAddr(t, "foundtmplgrp"), "id", tmplGroupAddr(t, "testtmplgrp"), "id"),
					resource.TestCheckResourceAttrPair("data.zabbix_template.foundtmpl", "id", "zabbix_template.testtmpl", "id"),
					resource.TestCheckResourceAttrPair("data.zabbix_host.foundhost", "id", "zabbix_host.testhost", "id"),
					resource.TestCheckResourceAttrPair("data.zabbix_proxy.foundproxy", "id", "zabbix_proxy.testproxy", "id"),
				),
			},
		},
	})
}
