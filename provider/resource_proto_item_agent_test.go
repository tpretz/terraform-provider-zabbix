package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceProtoItemAgent(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // create
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_agent" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "proto.agent[{#FSNAME}]"

	name = "Proto Agent {#FSNAME}"
	valuetype = "unsigned"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_agent.testproto", "key", "proto.agent[{#FSNAME}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_agent.testproto", "name", "Proto Agent {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_agent.testproto", "valuetype", "unsigned"),
					resource.TestCheckResourceAttr("zabbix_proto_item_agent.testproto", "active", "false"),
					resource.TestCheckResourceAttr("zabbix_proto_item_agent.testproto", "delay", "1m"),
					// the point of the exercise: ruleid is round-tripped through
					// itemprototype.get's selectDiscoveryRule, not just echoed
					// back from config
					resource.TestCheckResourceAttrPair(
						"zabbix_proto_item_agent.testproto", "ruleid",
						"zabbix_lld_trapper.testlld", "id"),
					resource.TestCheckResourceAttrPair(
						"zabbix_proto_item_agent.testproto", "hostid",
						"zabbix_template.testtmpl", "id"),
				),
			},
			{ // update: rename, active agent, non-default delay/history, tag
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_agent" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "proto.agent.changed[{#FSNAME}]"

	name = "Proto Agent Changed {#FSNAME}"
	valuetype = "float"

	active = true
	delay = "2m"
	history = "1h"
	trends = "7d"

	tag {
		key = "component"
		value = "proto"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_agent.testproto", "key", "proto.agent.changed[{#FSNAME}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_agent.testproto", "name", "Proto Agent Changed {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_agent.testproto", "valuetype", "float"),
					resource.TestCheckResourceAttr("zabbix_proto_item_agent.testproto", "active", "true"),
					resource.TestCheckResourceAttr("zabbix_proto_item_agent.testproto", "delay", "2m"),
					resource.TestCheckResourceAttr("zabbix_proto_item_agent.testproto", "history", "1h"),
					resource.TestCheckResourceAttr("zabbix_proto_item_agent.testproto", "trends", "7d"),
					resource.TestCheckResourceAttr("zabbix_proto_item_agent.testproto", "tag.0.key", "component"),
					resource.TestCheckResourceAttr("zabbix_proto_item_agent.testproto", "tag.0.value", "proto"),
				),
			},
			{ // import
				ResourceName:      "zabbix_proto_item_agent.testproto",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
