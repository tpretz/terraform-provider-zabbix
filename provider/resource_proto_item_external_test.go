package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceProtoItemExternal(t *testing.T) {
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
resource "zabbix_lld_external" "rule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.external.discovery"
  name   = "LLD External Rule"
}
resource "zabbix_proto_item_external" "testitem" {
  ruleid = zabbix_lld_external.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "external.key"
  name   = "Proto External Item"
  valuetype = "unsigned"
  delay = "1m"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("zabbix_proto_item_external.testitem", "ruleid"),
					resource.TestCheckResourceAttr("zabbix_proto_item_external.testitem", "key", "external.key"),
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
resource "zabbix_lld_external" "rule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.external.discovery"
  name   = "LLD External Rule"
}
resource "zabbix_proto_item_external" "testitem" {
  ruleid = zabbix_lld_external.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "external.key2"
  name   = "Proto External Item A"
  valuetype = "unsigned"
  delay = "30s"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_external.testitem", "key", "external.key2"),
					resource.TestCheckResourceAttr("zabbix_proto_item_external.testitem", "delay", "30s"),
				),
			},
		},
	})
}
