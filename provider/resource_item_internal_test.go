package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceItemInternal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
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
resource "zabbix_item_internal" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "script[\"abc\"]"

	name = "Ext Item"
	valuetype = "text"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_internal.testitem", "key", "script[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_item_internal.testitem", "name", "Ext Item"),
					resource.TestCheckResourceAttr("zabbix_item_internal.testitem", "valuetype", "text"),
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
resource "zabbix_item_internal" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "scriptb[\"abc\"]"

	name = "Ext Item A"
	valuetype = "unsigned"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_internal.testitem", "key", "scriptb[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_item_internal.testitem", "name", "Ext Item A"),
					resource.TestCheckResourceAttr("zabbix_item_internal.testitem", "valuetype", "unsigned"),
				),
			},
		},
	})
}
