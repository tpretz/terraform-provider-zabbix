package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceGraph(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // lazy load config, needed for skipfunc that look at meta
				Config: hcl(t, `
resource "zabbix_templategroup" "lazyconfigload" {
	name = "lazyload" 
}
`),
			},
			{ // simple create
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
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
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_graph.test", "name", "test"),
				),
			},
			{ // adjust and optional settings, second item
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
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
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_graph.test", "name", "testb"),
					resource.TestCheckResourceAttr("zabbix_graph.test", "item.#", "2"),
					// `item` is a set - check by content, not by index. The
					// order Zabbix returns gitems in is its own business and
					// differs between 7.x and 8.0.
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_graph.test", "item.*", map[string]string{
						"color":      "FFFF00",
						"function":   "max",
						"drawtype":   "dot",
						"sortorder":  "5",
						"type":       "sum",
						"yaxis_side": "right",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_graph.test", "item.*", map[string]string{
						"color":      "00FF00",
						"function":   "min",
						"drawtype":   "line",
						"sortorder":  "0",
						"type":       "simple",
						"yaxis_side": "left",
					}),
					resource.TestCheckTypeSetElemAttrPair("zabbix_graph.test", "item.*.itemid",
						"zabbix_item_agent.testitem", "id"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_graph.test", "item.*.itemid",
						"zabbix_item_agent.testitem-2", "id"),
				),
			},
			{ // the same two items, written the other way round: `item` is a
				// set, so the order they appear in the config carries no
				// meaning and this must plan clean. As a TypeList it did not.
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
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
		color = "00FF00"
		itemid = zabbix_item_agent.testitem-2.id
	}
	item {
		color = "FFFF00"
		itemid = zabbix_item_agent.testitem.id
		function = "max"
		drawtype = "dot"
		sortorder = "5"
		type = "sum"
		yaxis_side = "right"
	}
}
`),
				PlanOnly: true,
			},
		},
	})
}
