package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceGraph(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{ // lazy load config, needed for skipfunc that look at meta
				Config: `
resource "zabbix_hostgroup" "lazyconfigload" {
	name = "lazyload" 
}
`,
			},
			{ // simple create
				// SkipFunc: func() (bool, error) {
				// 	api := testAccProvider.Meta().(*zabbix.API)
				// 	return api.Config.Version >= 50400, nil
				// },
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_hostgroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitem"

	name = "Test Item"
	valuetype = "unsigned"
}
resource "zabbix_graph" "test" {
	name = "test" 
	width = "600"
	height = "400"

	type = "normal"

	item {
		color = "FFFF00"
		itemid = zabbix_item_agent.testitem.id
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_graph.test", "name", "test"),
				),
			},
		},
	})
}
