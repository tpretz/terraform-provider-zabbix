package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// dataSourceTemplateFixtureHCL is written against zabbix_templategroup and
// rewritten to zabbix_hostgroup by hcl() below 6.2 -- see the comment on hcl().
const dataSourceTemplateFixtureHCL = `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group-data"
}
resource "zabbix_template" "testtmpl" {
	groups      = [ zabbix_templategroup.testtmplgrp.id ]
	host        = "test-template-data"
	name        = "Test Template Data"
	description = "template used by the data source test"

	macro {
		name  = "{$TESTMACRO}"
		value = "testvalue"
	}
}
`

func TestAccDataSourceTemplate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // lookup by technical host name
				Config: hcl(t, dataSourceTemplateFixtureHCL+`
data "zabbix_template" "byhost" {
	host = zabbix_template.testtmpl.host
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.zabbix_template.byhost", "id",
						"zabbix_template.testtmpl", "id"),
					resource.TestCheckResourceAttr("data.zabbix_template.byhost", "host", "test-template-data"),
					resource.TestCheckResourceAttr("data.zabbix_template.byhost", "name", "Test Template Data"),
					resource.TestCheckResourceAttr("data.zabbix_template.byhost", "description", "template used by the data source test"),
					resource.TestCheckResourceAttr("data.zabbix_template.byhost", "groups.#", "1"),
					resource.TestCheckResourceAttrPair(
						"data.zabbix_template.byhost", "groups.0",
						tmplGroupAddr(t, "testtmplgrp"), "id"),
					resource.TestCheckResourceAttr("data.zabbix_template.byhost", "macro.#", "1"),
				),
			},
			{ // lookup by visible name
				Config: hcl(t, dataSourceTemplateFixtureHCL+`
data "zabbix_template" "byname" {
	name = zabbix_template.testtmpl.name
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.zabbix_template.byname", "id",
						"zabbix_template.testtmpl", "id"),
					resource.TestCheckResourceAttr("data.zabbix_template.byname", "host", "test-template-data"),
					resource.TestCheckResourceAttr("data.zabbix_template.byname", "name", "Test Template Data"),
				),
			},
			{ // the data source tracks a change to the underlying template, and
				// the id it resolves is usable as a real template reference
				Config: hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group-data"
}
resource "zabbix_hostgroup" "testhostgrp" {
	name = "test-group-templatedata"
}
resource "zabbix_template" "testtmpl" {
	groups      = [ zabbix_templategroup.testtmplgrp.id ]
	host        = "test-template-data"
	name        = "Test Template Data Renamed"
	description = "changed"
}
data "zabbix_template" "byhost" {
	host = zabbix_template.testtmpl.host
}
resource "zabbix_host" "testhost" {
	host      = "test-host-templatedata"
	groups    = [ zabbix_hostgroup.testhostgrp.id ]
	templates = [ data.zabbix_template.byhost.id ]

	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.zabbix_template.byhost", "name", "Test Template Data Renamed"),
					resource.TestCheckResourceAttr("data.zabbix_template.byhost", "description", "changed"),
					resource.TestCheckResourceAttr("data.zabbix_template.byhost", "macro.#", "0"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "templates.#", "1"),
					resource.TestCheckResourceAttrPair(
						"zabbix_host.testhost", "templates.0",
						"zabbix_template.testtmpl", "id"),
				),
			},
		},
	})
}
