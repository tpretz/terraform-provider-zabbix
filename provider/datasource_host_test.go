package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// dataSourceHostFixtureHCL is one host with enough on it -- group, interface,
// macro, tag, visible name -- that the data source's computed attributes are
// actually worth asserting on.
const dataSourceHostFixtureHCL = `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group-hostdata"
}
resource "zabbix_host" "testhost" {
	host   = "test-host-data"
	name   = "Test Host Data"
	groups = [ zabbix_hostgroup.testgrp.id ]

	interface {
		type = "agent"
		ip   = "127.0.0.1"
		port = 10099
	}

	macro {
		name  = "{$TESTMACRO}"
		value = "testvalue"
	}

	tag {
		key   = "env"
		value = "test"
	}
}
`

func TestAccDataSourceHost(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // lookup by technical host name
				Config: dataSourceHostFixtureHCL + `
data "zabbix_host" "byhost" {
	host = zabbix_host.testhost.host
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.zabbix_host.byhost", "id",
						"zabbix_host.testhost", "id"),
					resource.TestCheckResourceAttr("data.zabbix_host.byhost", "host", "test-host-data"),
					resource.TestCheckResourceAttr("data.zabbix_host.byhost", "name", "Test Host Data"),
					resource.TestCheckResourceAttr("data.zabbix_host.byhost", "enabled", "true"),
					resource.TestCheckResourceAttr("data.zabbix_host.byhost", "groups.#", "1"),
					resource.TestCheckResourceAttrPair(
						"data.zabbix_host.byhost", "groups.0",
						"zabbix_hostgroup.testgrp", "id"),
					resource.TestCheckResourceAttr("data.zabbix_host.byhost", "interface.#", "1"),
					resource.TestCheckResourceAttr("data.zabbix_host.byhost", "interface.0.ip", "127.0.0.1"),
					resource.TestCheckResourceAttr("data.zabbix_host.byhost", "interface.0.port", "10099"),
					resource.TestCheckResourceAttr("data.zabbix_host.byhost", "interface.0.type", "agent"),
					resource.TestCheckResourceAttr("data.zabbix_host.byhost", "macro.#", "1"),
					resource.TestCheckResourceAttr("data.zabbix_host.byhost", "templates.#", "0"),
				),
			},
			{ // lookup by visible name, and by id
				Config: dataSourceHostFixtureHCL + `
data "zabbix_host" "byname" {
	name = zabbix_host.testhost.name
}
data "zabbix_host" "byid" {
	hostid = zabbix_host.testhost.id
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.zabbix_host.byname", "id",
						"zabbix_host.testhost", "id"),
					resource.TestCheckResourceAttr("data.zabbix_host.byname", "host", "test-host-data"),
					resource.TestCheckResourceAttrPair(
						"data.zabbix_host.byid", "id",
						"zabbix_host.testhost", "id"),
					resource.TestCheckResourceAttr("data.zabbix_host.byid", "host", "test-host-data"),
					resource.TestCheckResourceAttr("data.zabbix_host.byid", "name", "Test Host Data"),
				),
			},
			{ // the data source tracks a change to the underlying host
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group-hostdata"
}
resource "zabbix_host" "testhost" {
	host    = "test-host-data"
	name    = "Test Host Data Renamed"
	enabled = false
	groups  = [ zabbix_hostgroup.testgrp.id ]

	interface {
		type = "agent"
		ip   = "127.0.0.2"
		port = 10098
	}
}
data "zabbix_host" "byhost" {
	host = zabbix_host.testhost.host
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.zabbix_host.byhost", "name", "Test Host Data Renamed"),
					resource.TestCheckResourceAttr("data.zabbix_host.byhost", "enabled", "false"),
					resource.TestCheckResourceAttr("data.zabbix_host.byhost", "interface.0.ip", "127.0.0.2"),
					resource.TestCheckResourceAttr("data.zabbix_host.byhost", "interface.0.port", "10098"),
					resource.TestCheckResourceAttr("data.zabbix_host.byhost", "macro.#", "0"),
				),
			},
		},
	})
}
