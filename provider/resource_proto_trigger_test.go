package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// protoTriggerFixtureHCL is the shared scaffolding: a template, a discovery
// rule, and an item prototype for the trigger prototype's expression to
// reference. A trigger prototype has no ruleid of its own -- Zabbix infers the
// owning rule from the prototypes named in the expression -- so there must be
// at least one item prototype in the expression or the create is rejected.
const protoTriggerFixtureHCL = `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template"
}
resource "zabbix_lld_trapper" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "test.lld.rule"
	name = "Test LLD Rule"
	delay = "0"
}
resource "zabbix_proto_item_trapper" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "trapper[{#FSNAME}]"

	name = "Proto Trapper {#FSNAME}"
	valuetype = "float"
}
`

func TestAccResourceProtoTrigger(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // create
				Config: hcl(t, protoTriggerFixtureHCL+`
resource "zabbix_proto_trigger" "testtrigger" {
	name = "Proto Trigger {#FSNAME}"
	expression = "last(/test-template/trapper[{#FSNAME}])=0"

	depends_on = [zabbix_proto_item_trapper.testproto]
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "name", "Proto Trigger {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "expression", "last(/test-template/trapper[{#FSNAME}])=0"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "priority", "not_classified"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "enabled", "true"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "manual_close", "false"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "multiple", "false"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "recovery_none", "false"),
				),
			},
			{ // update: severity, comments, url, manual close, disable, tag,
				// and a recovery expression (recovery mode 1)
				Config: hcl(t, protoTriggerFixtureHCL+`
resource "zabbix_proto_trigger" "testtrigger" {
	name = "Proto Trigger Renamed {#FSNAME}"
	expression = "last(/test-template/trapper[{#FSNAME}])>10"
	recovery_expression = "last(/test-template/trapper[{#FSNAME}])<5"
	comments = "proto trigger comment"
	priority = "average"
	enabled = false
	manual_close = true
	multiple = true
	url = "http://example.com/proto"

	tag {
		key = "protokey"
		value = "protovalue"
	}

	depends_on = [zabbix_proto_item_trapper.testproto]
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "name", "Proto Trigger Renamed {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "expression", "last(/test-template/trapper[{#FSNAME}])>10"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "recovery_expression", "last(/test-template/trapper[{#FSNAME}])<5"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "comments", "proto trigger comment"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "priority", "average"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "enabled", "false"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "manual_close", "true"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "multiple", "true"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "url", "http://example.com/proto"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "tag.0.key", "protokey"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "tag.0.value", "protovalue"),
				),
			},
			{ // import
				ResourceName:      "zabbix_proto_trigger.testtrigger",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
