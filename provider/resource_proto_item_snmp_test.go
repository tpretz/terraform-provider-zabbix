package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceProtoItemSnmp(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // create
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_snmp" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "proto.snmp[{#SNMPINDEX}]"

	name = "Proto SNMP {#SNMPINDEX}"
	valuetype = "unsigned"

	snmp_oid = "1.3.6.1.2.1.2.2.1.10.{#SNMPINDEX}"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_snmp.testproto", "key", "proto.snmp[{#SNMPINDEX}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmp.testproto", "name", "Proto SNMP {#SNMPINDEX}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmp.testproto", "valuetype", "unsigned"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmp.testproto", "snmp_oid", "1.3.6.1.2.1.2.2.1.10.{#SNMPINDEX}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmp.testproto", "delay", "1m"),
					resource.TestCheckResourceAttrPair(
						"zabbix_proto_item_snmp.testproto", "ruleid",
						"zabbix_lld_trapper.testlld", "id"),
				),
			},
			{ // update: new oid, preprocessing
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_snmp" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "proto.snmp.changed[{#SNMPINDEX}]"

	name = "Proto SNMP Changed {#SNMPINDEX}"
	valuetype = "float"

	snmp_oid = "1.3.6.1.2.1.2.2.1.16.{#SNMPINDEX}"
	delay = "3m"

	preprocessor {
		type = "10"
		error_handler = "0"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_snmp.testproto", "key", "proto.snmp.changed[{#SNMPINDEX}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmp.testproto", "name", "Proto SNMP Changed {#SNMPINDEX}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmp.testproto", "valuetype", "float"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmp.testproto", "snmp_oid", "1.3.6.1.2.1.2.2.1.16.{#SNMPINDEX}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmp.testproto", "delay", "3m"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmp.testproto", "preprocessor.0.type", "10"),
				),
			},
			{ // import
				ResourceName:      "zabbix_proto_item_snmp.testproto",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
