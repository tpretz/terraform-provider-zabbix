package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceItemSnmpTrap(t *testing.T) {
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
resource "zabbix_item_snmptrap" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "snmptrap[.*]"

	name = "Ext Item"
	valuetype = "text"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_snmptrap.testitem", "key", "snmptrap[.*]"),
					resource.TestCheckResourceAttr("zabbix_item_snmptrap.testitem", "name", "Ext Item"),
					resource.TestCheckResourceAttr("zabbix_item_snmptrap.testitem", "valuetype", "text"),
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
resource "zabbix_item_snmptrap" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "snmptrap.fallback"

	name = "Ext Item A"
	valuetype = "unsigned"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_snmptrap.testitem", "key", "snmptrap.fallback"),
					resource.TestCheckResourceAttr("zabbix_item_snmptrap.testitem", "name", "Ext Item A"),
					resource.TestCheckResourceAttr("zabbix_item_snmptrap.testitem", "valuetype", "unsigned"),
				),
			},
		},
	})
}
