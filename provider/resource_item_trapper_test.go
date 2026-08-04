package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceItemTrapper(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // simple create
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
	key = "trapper[.*]"

	name = "Ext Item"
	valuetype = "text"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "key", "trapper[.*]"),
					resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "name", "Ext Item"),
					resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "valuetype", "text"),
				),
			},
			{ // change values
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
	key = "trapper.fallback"

	name = "Ext Item A"
	valuetype = "unsigned"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "key", "trapper.fallback"),
					resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "name", "Ext Item A"),
					resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "valuetype", "unsigned"),
				),
			},
		},
	})
}
