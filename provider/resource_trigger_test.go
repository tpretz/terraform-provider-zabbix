package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceTrigger(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // lazy init
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
`),
			},
			{ // simple create: item to reference, plus a trigger on it
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_trapper" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "trapper.test"

	name = "Trapper Item"
	valuetype = "float"
}
resource "zabbix_trigger" "testtrigger" {
	name = "Test Trigger"
	expression = "last(/test-template/trapper.test)=0"

	depends_on = [zabbix_item_trapper.testitem]
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "name", "Test Trigger"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "expression", "last(/test-template/trapper.test)=0"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "priority", "not_classified"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "enabled", "true"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "manual_close", "false"),
				),
			},
			{ // modify: priority, comments, url, manual_close, disable, tag
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_trapper" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "trapper.test"

	name = "Trapper Item"
	valuetype = "float"
}
resource "zabbix_trigger" "testtrigger" {
	name = "Test Trigger Renamed"
	expression = "last(/test-template/trapper.test)=1"
	comments = "test comment"
	priority = "high"
	enabled = false
	manual_close = true
	url = "http://example.com"

	tag {
		key = "testtag"
		value = "testvalue"
	}

	depends_on = [zabbix_item_trapper.testitem]
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "name", "Test Trigger Renamed"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "expression", "last(/test-template/trapper.test)=1"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "comments", "test comment"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "priority", "high"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "enabled", "false"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "manual_close", "true"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "url", "http://example.com"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "tag.0.key", "testtag"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "tag.0.value", "testvalue"),
				),
			},
			{ // dependency on another trigger
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_trapper" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "trapper.test"

	name = "Trapper Item"
	valuetype = "float"
}
resource "zabbix_trigger" "testtriggerdep" {
	name = "Test Trigger Dependency"
	expression = "last(/test-template/trapper.test)=2"

	depends_on = [zabbix_item_trapper.testitem]
}
resource "zabbix_trigger" "testtrigger" {
	name = "Test Trigger Renamed"
	expression = "last(/test-template/trapper.test)=1"
	comments = "test comment"
	priority = "high"
	enabled = false
	manual_close = true
	url = "http://example.com"

	tag {
		key = "testtag"
		value = "testvalue"
	}

	dependencies = [zabbix_trigger.testtriggerdep.id]

	depends_on = [zabbix_item_trapper.testitem]
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "dependencies.#", "1"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_trigger.testtrigger", "dependencies.*", "zabbix_trigger.testtriggerdep", "id"),
				),
			},
			{ // import
				ResourceName:      "zabbix_trigger.testtrigger",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
