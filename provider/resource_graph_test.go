package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// graphFixtureHCL is the scaffolding every graph step needs: a group, a
// template, and three plain items to plot. Three, and all of the same backend
// type, so that C3 has elements the provider can only tell apart by content.
const graphFixtureHCL = `
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
resource "zabbix_item_agent" "testitem-3" {
	hostid = zabbix_template.testtmpl.id
	key = "testitemc"

	name = "Test Itemc"
	valuetype = "unsigned"
}
`

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
			{ // C2: one item
				Config: hcl(t, graphFixtureHCL+`
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
					resource.TestCheckResourceAttr("zabbix_graph.test", "item.#", "1"),
					testAccCheckGraphItemCount("zabbix_graph.test", 1),
				),
			},
			{ // adjust and optional settings, second item
				Config: hcl(t, graphFixtureHCL+`
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
					testAccCheckGraphItemCount("zabbix_graph.test", 2),
				),
			},
			{ // C4: the same two items, written the other way round. `item` is
				// a set, so the order they appear in the config carries no
				// meaning and this must plan clean. As a TypeList it did not.
				Config: hcl(t, graphFixtureHCL+`
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
			{ // C3: three items, all of the same backend type, so nothing but
				// their content distinguishes them
				Config: hcl(t, graphFixtureHCL+`
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
	item {
		color = "0000FF"
		itemid = zabbix_item_agent.testitem-3.id
		sortorder = "7"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_graph.test", "item.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_graph.test", "item.*", map[string]string{
						"color":     "0000FF",
						"sortorder": "7",
					}),
					resource.TestCheckTypeSetElemAttrPair("zabbix_graph.test", "item.*.itemid",
						"zabbix_item_agent.testitem-3", "id"),
					testAccCheckGraphItemCount("zabbix_graph.test", 3),
				),
			},
			{ // C7: import at full size. This is the only step that proves the
				// flatten function and the set hash agree about `item`.
				ResourceName:      "zabbix_graph.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{ // C6: drop the middle item. A graph must keep at least one item,
				// so N -> 0 is not reachable for this collection -- the
				// zero case is the resource's own destroy, which CheckDestroy
				// covers. The removal is verified against the server, not just
				// against state: graph.update replaces gitems wholesale, so an
				// omitted item is a deletion and the count has to move.
				Config: hcl(t, graphFixtureHCL+`
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
		color = "0000FF"
		itemid = zabbix_item_agent.testitem-3.id
		sortorder = "7"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_graph.test", "item.#", "2"),
					// the two survivors, by content
					resource.TestCheckTypeSetElemAttrPair("zabbix_graph.test", "item.*.itemid",
						"zabbix_item_agent.testitem", "id"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_graph.test", "item.*.itemid",
						"zabbix_item_agent.testitem-3", "id"),
					// and the server agrees the third is gone
					testAccCheckGraphItemCount("zabbix_graph.test", 2),
				),
			},
			{ // C6 continued: back to a single item
				Config: hcl(t, graphFixtureHCL+`
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
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_graph.test", "item.#", "1"),
					testAccCheckGraphItemCount("zabbix_graph.test", 1),
				),
			},
		},
	})
}
