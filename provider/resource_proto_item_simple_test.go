package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceProtoItemSimple(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // create
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_simple" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "net.tcp.service[{#PORT}]"

	name = "Proto Simple {#PORT}"
	valuetype = "unsigned"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_simple.testproto", "key", "net.tcp.service[{#PORT}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_simple.testproto", "name", "Proto Simple {#PORT}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_simple.testproto", "valuetype", "unsigned"),
					resource.TestCheckResourceAttr("zabbix_proto_item_simple.testproto", "delay", "1m"),
					resource.TestCheckResourceAttrPair(
						"zabbix_proto_item_simple.testproto", "ruleid",
						"zabbix_lld_trapper.testlld", "id"),
				),
			},
			{ // update
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_simple" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "net.tcp.service.perf[{#PORT}]"

	name = "Proto Simple Changed {#PORT}"
	valuetype = "float"

	delay = "90s"
	trends = "20d"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_simple.testproto", "key", "net.tcp.service.perf[{#PORT}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_simple.testproto", "name", "Proto Simple Changed {#PORT}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_simple.testproto", "valuetype", "float"),
					resource.TestCheckResourceAttr("zabbix_proto_item_simple.testproto", "delay", "90s"),
					resource.TestCheckResourceAttr("zabbix_proto_item_simple.testproto", "trends", "20d"),
				),
			},
			{ // import
				ResourceName:      "zabbix_proto_item_simple.testproto",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
