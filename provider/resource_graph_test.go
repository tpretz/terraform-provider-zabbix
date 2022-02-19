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
			{ // adjust and optional settings, second item
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
resource "zabbix_item_agent" "testitem-2" {
	hostid = zabbix_template.testtmpl.id
	key = "testitemb"

	name = "Test Itemb"
	valuetype = "unsigned"
}
resource "zabbix_graph" "test" {
	name = "testb" 
	width = "500"
	height = "300"
	percent_left = "20"
	percent_right = "20"
	do3d = true
	legend = false
	work_period = false
	ymax = "80"
	ymax_type = "fixed"
	ymin = "10"
	ymin_type = "fixed"

	type = "stacked"

	item {
		color = "FFFF00"
		itemid = zabbix_item_agent.testitem.id
		function = "max"
		drawtype = "dot"
		sortorder = "5"
		type = "sum"
		yaxis_side = "right"
	}
	item {
		color = "00FF00"
		itemid = zabbix_item_agent.testitem-2.id
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_graph.test", "name", "testb"),
				),
			},
		},
	})
}
