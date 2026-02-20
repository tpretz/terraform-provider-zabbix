package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceProtoItemSnmpTrap(t *testing.T) {
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

resource "zabbix_proto_item_snmptrap" "testitem" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "snmptrap[.*]"

  name      = "Proto SNMPTrap Item"
  valuetype = "text"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("zabbix_proto_item_snmptrap.testitem", "ruleid"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmptrap.testitem", "key", "snmptrap[.*]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmptrap.testitem", "name", "Proto SNMPTrap Item"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmptrap.testitem", "valuetype", "text"),
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

resource "zabbix_proto_item_snmptrap" "testitem" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "snmptrap.fallback"

  name      = "Proto SNMPTrap Item A"
  valuetype = "unsigned"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_snmptrap.testitem", "key", "snmptrap.fallback"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmptrap.testitem", "name", "Proto SNMPTrap Item A"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmptrap.testitem", "valuetype", "unsigned"),
				),
			},
		},
	})
}
