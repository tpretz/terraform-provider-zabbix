package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// U1/U2 for graphs and triggers (PLAN.md § "The unit of work").
//
// Both objects come in a plain and a prototype flavour built from one schema
// -- resource_graph.go and resource_trigger.go each declare their attributes
// once and register them twice -- so, as everywhere else here, the fragment is
// covered once and the pointer-identity grouping in TestUpdateCoverageComplete
// carries it to the prototype resource.

// updateGraphFixtureHCL gives the graph tests a template and two items, so
// that a graph item can be moved from one item to another and the y-axis
// bounds can name a different item than the plot does.
const updateGraphFixtureHCL = `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-update-graph-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host   = "test-update-graph-template"
}
resource "zabbix_item_agent" "testitema" {
	hostid    = zabbix_template.testtmpl.id
	key       = "test.update.graph.a"
	name      = "Update Graph Item A"
	valuetype = "unsigned"
}
resource "zabbix_item_agent" "testitemb" {
	hostid    = zabbix_template.testtmpl.id
	key       = "test.update.graph.b"
	name      = "Update Graph Item B"
	valuetype = "unsigned"
}
`

// TestAccUpdateGraph changes every attribute a graph has, including every
// attribute of its one graph item, on a graph that already exists.
//
// ymin_itemid and ymax_itemid are only settable while the matching *_type is
// "item", so the first step establishes them with type "item" and the second
// moves them to the other item as well as switching one axis to "fixed" --
// otherwise the id would be unreachable in one of the two configurations.
func TestAccUpdateGraph(t *testing.T) {
	const addr = "zabbix_graph.testgraph"

	graph := func(body string) string {
		return hcl(t, updateGraphFixtureHCL+`
resource "zabbix_graph" "testgraph" {
`+body+`
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: graph(`
	name          = "Update Graph A"
	width         = "900"
	height        = "200"
	type          = "normal"
	do3d          = false
	legend        = true
	work_period   = true
	percent_left  = "10"
	percent_right = "20"
	ymin_type     = "item"
	ymin_itemid   = zabbix_item_agent.testitema.id
	ymax_type     = "item"
	ymax_itemid   = zabbix_item_agent.testitema.id
	item {
		itemid     = zabbix_item_agent.testitema.id
		color      = "AA0000"
		drawtype   = "line"
		function   = "min"
		sortorder  = "0"
		type       = "simple"
		yaxis_side = "left"
	}
`),
				Check: testAccCheckServerAttrs(addr, serverGraph, map[string]string{
					"name":             "Update Graph A",
					"width":            "900",
					"height":           "200",
					"graphtype":        "0",
					"show_3d":          "0",
					"show_legend":      "1",
					"show_work_period": "1",
					"percent_left":     "10",
					"percent_right":    "20",
					"ymin_type":        "2",
					"ymax_type":        "2",
				}),
			},
			{ // every one of them changed, in life
				Config: graph(`
	name          = "Update Graph B"
	width         = "1200"
	height        = "300"
	type          = "stacked"
	do3d          = true
	legend        = false
	work_period   = false
	percent_left  = "15"
	percent_right = "25"
	ymin_type     = "fixed"
	ymin          = "5"
	ymax_type     = "item"
	ymax_itemid   = zabbix_item_agent.testitemb.id
	item {
		itemid     = zabbix_item_agent.testitemb.id
		color      = "00BB00"
		drawtype   = "bold"
		function   = "average"
		sortorder  = "3"
		type       = "sum"
		yaxis_side = "right"
	}
`),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerAttrs(addr, serverGraph, map[string]string{
						"name":             "Update Graph B",
						"width":            "1200",
						"height":           "300",
						"graphtype":        "1",
						"show_3d":          "1",
						"show_legend":      "0",
						"show_work_period": "0",
						"percent_left":     "15",
						"percent_right":    "25",
						"ymin_type":        "1",
						"yaxismin":         "5",
						"ymax_type":        "2",
					}),
					// the graph item, located by the item it plots rather than
					// by position: 8.0 returns gitems in a different order
					// from 6.0 and 7.x
					testAccCheckGraphItemFor(addr, "zabbix_item_agent.testitemb", map[string]string{
						"color":     "00BB00",
						"drawtype":  "2",
						"calc_fnc":  "2",
						"sortorder": "3",
						"type":      "2",
						"yaxisside": "1",
					}),
					testAccCheckGraphAxisItem(addr, "ymax_itemid", "zabbix_item_agent.testitemb"),
				),
			},
		},
	})
}

// testAccCheckGraphItemFor asserts the graph item the server holds for the
// item at itemAddr. do3d has no server-side twin worth asserting beyond
// show_3d, which the caller does.
func testAccCheckGraphItemFor(addr, itemAddr string, want map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		itemID, err := testAccStateID(s, itemAddr)
		if err != nil {
			return err
		}
		return testAccCheckServerElem(addr, serverGraph, "gitems", "itemid", itemID, want)(s)
	}
}

// testAccCheckGraphAxisItem asserts that the server has the graph's named
// axis pointing at the item at itemAddr.
func testAccCheckGraphAxisItem(addr, prop, itemAddr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		itemID, err := testAccStateID(s, itemAddr)
		if err != nil {
			return err
		}
		return testAccCheckServerAttrs(addr, serverGraph, map[string]string{prop: itemID})(s)
	}
}
