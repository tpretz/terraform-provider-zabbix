package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceProtoItemTrapper(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // create
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_trapper" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "trapper[{#FSNAME}]"

	name = "Proto Trapper {#FSNAME}"
	valuetype = "text"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_trapper.testproto", "key", "trapper[{#FSNAME}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_trapper.testproto", "name", "Proto Trapper {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_trapper.testproto", "valuetype", "text"),
					resource.TestCheckResourceAttr("zabbix_proto_item_trapper.testproto", "history", "90d"),
					resource.TestCheckResourceAttrPair(
						"zabbix_proto_item_trapper.testproto", "ruleid",
						"zabbix_lld_trapper.testlld", "id"),
				),
			},
			{ // update
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_trapper" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "trapper.fallback[{#FSNAME}]"

	name = "Proto Trapper Changed {#FSNAME}"
	valuetype = "unsigned"

	history = "6h"
	trends = "15d"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_trapper.testproto", "key", "trapper.fallback[{#FSNAME}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_trapper.testproto", "name", "Proto Trapper Changed {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_trapper.testproto", "valuetype", "unsigned"),
					resource.TestCheckResourceAttr("zabbix_proto_item_trapper.testproto", "history", "6h"),
					resource.TestCheckResourceAttr("zabbix_proto_item_trapper.testproto", "trends", "15d"),
				),
			},
			{ // import
				ResourceName:      "zabbix_proto_item_trapper.testproto",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
