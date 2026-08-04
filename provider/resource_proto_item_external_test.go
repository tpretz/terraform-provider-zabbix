package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceProtoItemExternal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // create
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_external" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "script[{#FSNAME}]"

	name = "Proto External {#FSNAME}"
	valuetype = "text"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_external.testproto", "key", "script[{#FSNAME}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_external.testproto", "name", "Proto External {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_external.testproto", "valuetype", "text"),
					resource.TestCheckResourceAttr("zabbix_proto_item_external.testproto", "delay", "1m"),
					resource.TestCheckResourceAttrPair(
						"zabbix_proto_item_external.testproto", "ruleid",
						"zabbix_lld_trapper.testlld", "id"),
				),
			},
			{ // update
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_external" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "scriptb[{#FSNAME}]"

	name = "Proto External Changed {#FSNAME}"
	valuetype = "unsigned"

	delay = "30s"
	history = "3h"
	trends = "10d"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_external.testproto", "key", "scriptb[{#FSNAME}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_external.testproto", "name", "Proto External Changed {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_external.testproto", "valuetype", "unsigned"),
					resource.TestCheckResourceAttr("zabbix_proto_item_external.testproto", "delay", "30s"),
					resource.TestCheckResourceAttr("zabbix_proto_item_external.testproto", "history", "3h"),
					resource.TestCheckResourceAttr("zabbix_proto_item_external.testproto", "trends", "10d"),
				),
			},
			{ // import
				ResourceName:      "zabbix_proto_item_external.testproto",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
