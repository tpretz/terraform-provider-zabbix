package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccResourceProtoItemAggregate(t *testing.T) {
	id := resource.UniqueId()
	groupName := "test-group-" + id
	tmplHost := "test-template-" + id

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					// Zabbix >= 5.4 rejects aggregate item prototypes (type=8) (observed in 5.4.12 and 6.0.44).
					return api.Config.Version >= 50400, nil
				},
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

resource "zabbix_proto_item_aggregate" "testitem" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "grpavg[\"Zabbix servers\",\"system.uptime\",\"last\",\"0\"]"
  name   = "Proto Aggregate Item"
  valuetype = "unsigned"
  delay = "1m"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_aggregate.testitem", "delay", "1m"),
				),
			},
			{
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version >= 50400, nil
				},
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

resource "zabbix_proto_item_aggregate" "testitem" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "grpmax[\"Zabbix servers\",\"system.uptime\",\"last\",\"0\"]"
  name   = "Proto Aggregate Item A"
  valuetype = "unsigned"
  delay = "30s"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_aggregate.testitem", "key", "grpmax[\"Zabbix servers\",\"system.uptime\",\"last\",\"0\"]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_aggregate.testitem", "delay", "30s"),
				),
			},
		},
	})
}
