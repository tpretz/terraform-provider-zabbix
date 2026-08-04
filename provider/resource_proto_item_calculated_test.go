package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceProtoItemCalculated(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // create
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_calculated" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "proto.calculated[{#FSNAME}]"

	name = "Proto Calculated {#FSNAME}"
	valuetype = "float"

	formula = "last(//test.lld.rule)"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_calculated.testproto", "key", "proto.calculated[{#FSNAME}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_calculated.testproto", "name", "Proto Calculated {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_calculated.testproto", "valuetype", "float"),
					resource.TestCheckResourceAttr("zabbix_proto_item_calculated.testproto", "formula", "last(//test.lld.rule)"),
					resource.TestCheckResourceAttr("zabbix_proto_item_calculated.testproto", "delay", "1m"),
					resource.TestCheckResourceAttrPair(
						"zabbix_proto_item_calculated.testproto", "ruleid",
						"zabbix_lld_trapper.testlld", "id"),
				),
			},
			{ // update: rename, new formula, non-default delay
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_calculated" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "proto.calculated.changed[{#FSNAME}]"

	name = "Proto Calculated Changed {#FSNAME}"
	valuetype = "unsigned"

	formula = "count(//test.lld.rule,10m)"
	delay = "5m"
	history = "2h"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_calculated.testproto", "key", "proto.calculated.changed[{#FSNAME}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_calculated.testproto", "name", "Proto Calculated Changed {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_calculated.testproto", "valuetype", "unsigned"),
					resource.TestCheckResourceAttr("zabbix_proto_item_calculated.testproto", "formula", "count(//test.lld.rule,10m)"),
					resource.TestCheckResourceAttr("zabbix_proto_item_calculated.testproto", "delay", "5m"),
					resource.TestCheckResourceAttr("zabbix_proto_item_calculated.testproto", "history", "2h"),
				),
			},
			{ // import
				ResourceName:      "zabbix_proto_item_calculated.testproto",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
