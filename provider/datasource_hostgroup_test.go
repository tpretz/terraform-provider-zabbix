package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceHostgroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // look the group up by name and confirm it resolves to the same object
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group-data"
}
data "zabbix_hostgroup" "lookup" {
	name = zabbix_hostgroup.testgrp.name
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.zabbix_hostgroup.lookup", "name", "test-group-data"),
					resource.TestCheckResourceAttrPair(
						"data.zabbix_hostgroup.lookup", "id",
						"zabbix_hostgroup.testgrp", "id"),
				),
			},
			{ // the id follows a rename, i.e. the lookup is really being redone
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group-data-renamed"
}
data "zabbix_hostgroup" "lookup" {
	name = zabbix_hostgroup.testgrp.name
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.zabbix_hostgroup.lookup", "name", "test-group-data-renamed"),
					resource.TestCheckResourceAttrPair(
						"data.zabbix_hostgroup.lookup", "id",
						"zabbix_hostgroup.testgrp", "id"),
				),
			},
			{ // the resolved id is usable as a real group reference
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group-data-renamed"
}
data "zabbix_hostgroup" "lookup" {
	name = zabbix_hostgroup.testgrp.name
}
resource "zabbix_host" "testhost" {
	host   = "test-host-datagroup"
	groups = [ data.zabbix_hostgroup.lookup.id ]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "groups.#", "1"),
					resource.TestCheckResourceAttrPair(
						"zabbix_host.testhost", "groups.0",
						"zabbix_hostgroup.testgrp", "id"),
				),
			},
		},
	})
}
