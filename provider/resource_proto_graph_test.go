package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// protoGraphFixtureHCL gives a graph prototype the two things it needs: a
// discovery rule, and item prototypes to plot. Like a trigger prototype, a
// graph prototype has no ruleid attribute -- Zabbix derives the owning rule
// from the item prototypes in its "item" blocks, so at least one of them must
// be a prototype rather than a plain item.
const protoGraphFixtureHCL = `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template"
}
resource "zabbix_lld_trapper" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "test.lld.rule"
	name = "Test LLD Rule"
	delay = "0"
}
resource "zabbix_proto_item_trapper" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "trapper[{#FSNAME}]"

	name = "Proto Trapper {#FSNAME}"
	valuetype = "unsigned"
}
resource "zabbix_proto_item_trapper" "testproto2" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "trapperb[{#FSNAME}]"

	name = "Proto Trapper B {#FSNAME}"
	valuetype = "unsigned"
}
resource "zabbix_item_trapper" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "trapper.plain"

	name = "Plain Trapper Item"
	valuetype = "unsigned"
}
`

func TestAccResourceProtoGraph(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // create: one item prototype, everything else defaulted
				Config: hcl(t, protoGraphFixtureHCL+`
resource "zabbix_proto_graph" "testgraph" {
	name = "Proto Graph {#FSNAME}"
	width = "600"
	height = "400"

	item {
		color = "FFFF00"
		itemid = zabbix_proto_item_trapper.testproto.id
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "name", "Proto Graph {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "width", "600"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "height", "400"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "type", "normal"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "legend", "true"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "item.#", "1"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "item.0.color", "FFFF00"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "item.0.function", "min"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "item.0.drawtype", "line"),
					resource.TestCheckResourceAttrPair(
						"zabbix_proto_graph.testgraph", "item.0.itemid",
						"zabbix_proto_item_trapper.testproto", "id"),
				),
			},
			{ // update: axis options, a second prototype and a plain item --
				// a graph prototype may mix both, provided at least one is a
				// prototype
				Config: hcl(t, protoGraphFixtureHCL+`
resource "zabbix_proto_graph" "testgraph" {
	name = "Proto Graph Renamed {#FSNAME}"
	width = "500"
	height = "300"

	type = "stacked"
	legend = false
	work_period = false
	ymin = "10"
	ymin_type = "fixed"
	ymax = "80"
	ymax_type = "fixed"

	item {
		color = "FFFF00"
		itemid = zabbix_proto_item_trapper.testproto.id
		function = "max"
		drawtype = "bold"
		sortorder = "1"
		yaxis_side = "right"
	}
	item {
		color = "00FF00"
		itemid = zabbix_proto_item_trapper.testproto2.id
		sortorder = "2"
	}
	item {
		color = "0000FF"
		itemid = zabbix_item_trapper.testitem.id
		sortorder = "3"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "name", "Proto Graph Renamed {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "width", "500"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "height", "300"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "type", "stacked"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "legend", "false"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "work_period", "false"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "ymin_type", "fixed"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "ymax_type", "fixed"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "item.#", "3"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "item.0.function", "max"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "item.0.drawtype", "bold"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "item.0.yaxis_side", "right"),
					resource.TestCheckResourceAttrPair(
						"zabbix_proto_graph.testgraph", "item.2.itemid",
						"zabbix_item_trapper.testitem", "id"),
				),
			},
			{ // import
				ResourceName:      "zabbix_proto_graph.testgraph",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
