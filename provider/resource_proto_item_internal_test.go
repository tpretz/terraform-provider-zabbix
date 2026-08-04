package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceProtoItemInternal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // create
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_internal" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "script[{#FSNAME}]"

	name = "Proto Internal {#FSNAME}"
	valuetype = "text"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_internal.testproto", "key", "script[{#FSNAME}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_internal.testproto", "name", "Proto Internal {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_internal.testproto", "valuetype", "text"),
					resource.TestCheckResourceAttr("zabbix_proto_item_internal.testproto", "delay", "1m"),
					resource.TestCheckResourceAttrPair(
						"zabbix_proto_item_internal.testproto", "ruleid",
						"zabbix_lld_trapper.testlld", "id"),
				),
			},
			{ // update
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_internal" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "scriptb[{#FSNAME}]"

	name = "Proto Internal Changed {#FSNAME}"
	valuetype = "float"

	delay = "45s"
	history = "4h"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_internal.testproto", "key", "scriptb[{#FSNAME}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_internal.testproto", "name", "Proto Internal Changed {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_internal.testproto", "valuetype", "float"),
					resource.TestCheckResourceAttr("zabbix_proto_item_internal.testproto", "delay", "45s"),
					resource.TestCheckResourceAttr("zabbix_proto_item_internal.testproto", "history", "4h"),
				),
			},
			{ // import
				ResourceName:      "zabbix_proto_item_internal.testproto",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
