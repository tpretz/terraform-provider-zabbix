package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceProtoItemSnmpTrap(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // create
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_snmptrap" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "snmptrap[{#SNMPINDEX}]"

	name = "Proto SNMP Trap {#SNMPINDEX}"
	valuetype = "text"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_snmptrap.testproto", "key", "snmptrap[{#SNMPINDEX}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmptrap.testproto", "name", "Proto SNMP Trap {#SNMPINDEX}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmptrap.testproto", "valuetype", "text"),
					resource.TestCheckResourceAttrPair(
						"zabbix_proto_item_snmptrap.testproto", "ruleid",
						"zabbix_lld_trapper.testlld", "id"),
				),
			},
			{ // update
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_snmptrap" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "snmptrap[changed-{#SNMPINDEX}]"

	name = "Proto SNMP Trap Changed {#SNMPINDEX}"
	valuetype = "log"

	history = "5h"

	tag {
		key = "source"
		value = "trap"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_snmptrap.testproto", "key", "snmptrap[changed-{#SNMPINDEX}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmptrap.testproto", "name", "Proto SNMP Trap Changed {#SNMPINDEX}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmptrap.testproto", "valuetype", "log"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmptrap.testproto", "history", "5h"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmptrap.testproto", "tag.0.key", "source"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmptrap.testproto", "tag.0.value", "trap"),
				),
			},
			{ // import
				ResourceName:      "zabbix_proto_item_snmptrap.testproto",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
