package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceProtoItemDependent(t *testing.T) {
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

resource "zabbix_proto_item_simple" "parent" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "script[\"abc\"]"
  name   = "Proto Simple Master"
  valuetype = "text"
}

resource "zabbix_proto_item_dependent" "testitem" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id

  key       = "depitem"
  name      = "Proto Dependent Item"
  valuetype = "text"

  master_itemid = zabbix_proto_item_simple.parent.id
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("zabbix_proto_item_dependent.testitem", "master_itemid"),
					resource.TestCheckResourceAttr("zabbix_proto_item_dependent.testitem", "key", "depitem"),
					resource.TestCheckResourceAttr("zabbix_proto_item_dependent.testitem", "name", "Proto Dependent Item"),
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

resource "zabbix_proto_item_simple" "parent" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "script[\"abc\"]"
  name   = "Proto Simple Master"
  valuetype = "text"
}

resource "zabbix_proto_item_simple" "parent2" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "scriptb[\"abc\"]"
  name   = "Proto Simple Master 2"
  valuetype = "text"
}

resource "zabbix_proto_item_dependent" "testitem" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id

  key       = "depitem2"
  name      = "Proto Dependent Item A"
  valuetype = "unsigned"

  master_itemid = zabbix_proto_item_simple.parent2.id
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_dependent.testitem", "key", "depitem2"),
					resource.TestCheckResourceAttr("zabbix_proto_item_dependent.testitem", "name", "Proto Dependent Item A"),
					resource.TestCheckResourceAttr("zabbix_proto_item_dependent.testitem", "valuetype", "unsigned"),
				),
			},
		},
	})
}
