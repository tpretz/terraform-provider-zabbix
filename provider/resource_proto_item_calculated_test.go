package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceProtoItemCalculated(t *testing.T) {
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

resource "zabbix_proto_item_calculated" "testitem" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "calc.key"
  name   = "Proto Calculated Item"
  valuetype = "unsigned"
  delay = "1m"
  formula = "1+1"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_calculated.testitem", "formula", "1+1"),
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

resource "zabbix_proto_item_calculated" "testitem" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "calc.key2"
  name   = "Proto Calculated Item A"
  valuetype = "unsigned"
  delay = "30s"
  formula = "2+2"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_calculated.testitem", "key", "calc.key2"),
					resource.TestCheckResourceAttr("zabbix_proto_item_calculated.testitem", "formula", "2+2"),
				),
			},
		},
	})
}
