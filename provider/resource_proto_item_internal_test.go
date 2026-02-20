package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceProtoItemInternal(t *testing.T) {
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
resource "zabbix_lld_internal" "rule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.internal.discovery"
  name   = "LLD Internal Rule"
}
resource "zabbix_proto_item_internal" "testitem" {
  ruleid = zabbix_lld_internal.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "internal.key"
  name   = "Proto Internal Item"
  valuetype = "unsigned"
  delay = "1m"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("zabbix_proto_item_internal.testitem", "ruleid"),
					resource.TestCheckResourceAttr("zabbix_proto_item_internal.testitem", "key", "internal.key"),
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
resource "zabbix_lld_internal" "rule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.internal.discovery"
  name   = "LLD Internal Rule"
}
resource "zabbix_proto_item_internal" "testitem" {
  ruleid = zabbix_lld_internal.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "internal.key2"
  name   = "Proto Internal Item A"
  valuetype = "unsigned"
  delay = "30s"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_internal.testitem", "key", "internal.key2"),
					resource.TestCheckResourceAttr("zabbix_proto_item_internal.testitem", "delay", "30s"),
				),
			},
		},
	})
}
