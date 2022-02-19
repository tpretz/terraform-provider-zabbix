package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceItemSimple(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{ // simple create
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_hostgroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_simple" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "script[\"abc\"]"

	name = "Ext Item"
	valuetype = "text"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_simple.testitem", "key", "script[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_item_simple.testitem", "name", "Ext Item"),
					resource.TestCheckResourceAttr("zabbix_item_simple.testitem", "valuetype", "text"),
				),
			},
			{ // change values
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_hostgroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_simple" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "scriptb[\"abc\"]"

	name = "Ext Item A"
	valuetype = "unsigned"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_simple.testitem", "key", "scriptb[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_item_simple.testitem", "name", "Ext Item A"),
					resource.TestCheckResourceAttr("zabbix_item_simple.testitem", "valuetype", "unsigned"),
				),
			},
		},
	})
}
