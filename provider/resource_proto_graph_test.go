package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceProtoGraph(t *testing.T) {
	id := resource.UniqueId()
	groupName := "test-group-" + id
	tmplHost := "test-template-" + id

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}

resource "zabbix_template" "testtmpl" {
  groups = [zabbix_hostgroup.testgrp.id]
  host   = %q
}

resource "zabbix_lld_simple" "rule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.simple.discovery"
  name   = "LLD Simple Rule"
}

resource "zabbix_proto_item_simple" "item1" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "script[\"abc\"]"

  name      = "Proto Simple Item 1"
  valuetype = "unsigned"
}

resource "zabbix_proto_item_simple" "item2" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "scriptb[\"abc\"]"

  name      = "Proto Simple Item 2"
  valuetype = "unsigned"
}

resource "zabbix_proto_graph" "testgraph" {
  name   = "proto-graph"
  width  = "900"
  height = "200"

  item {
    color     = "00AA00"
    itemid    = zabbix_proto_item_simple.item1.id
    drawtype  = "line"
    function  = "min"
    sortorder = "0"
    type      = "simple"
    yaxis_side = "left"
  }

  item {
    color     = "AA0000"
    itemid    = zabbix_proto_item_simple.item2.id
    drawtype  = "line"
    function  = "min"
    sortorder = "1"
    type      = "simple"
    yaxis_side = "right"
  }
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "name", "proto-graph"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "width", "900"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "height", "200"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "item.#", "2"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}

resource "zabbix_template" "testtmpl" {
  groups = [zabbix_hostgroup.testgrp.id]
  host   = %q
}

resource "zabbix_lld_simple" "rule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.simple.discovery"
  name   = "LLD Simple Rule"
}

resource "zabbix_proto_item_simple" "item1" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "script[\"abc\"]"

  name      = "Proto Simple Item 1"
  valuetype = "unsigned"
}

resource "zabbix_proto_item_simple" "item2" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "scriptb[\"abc\"]"

  name      = "Proto Simple Item 2"
  valuetype = "unsigned"
}

resource "zabbix_proto_graph" "testgraph" {
  name   = "proto-graph-a"
  width  = "1000"
  height = "250"

  item {
    color     = "0000AA"
    itemid    = zabbix_proto_item_simple.item1.id
    drawtype  = "bold"
    function  = "max"
    sortorder = "0"
    type      = "simple"
    yaxis_side = "left"
  }

  item {
    color     = "AAAA00"
    itemid    = zabbix_proto_item_simple.item2.id
    drawtype  = "dot"
    function  = "average"
    sortorder = "1"
    type      = "simple"
    yaxis_side = "right"
  }
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "name", "proto-graph-a"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "width", "1000"),
					resource.TestCheckResourceAttr("zabbix_proto_graph.testgraph", "height", "250"),
				),
			},
		},
	})
}
