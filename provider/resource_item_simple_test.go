package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceItemSimple(t *testing.T) {
	id := resource.UniqueId()
	groupName := "test-group-" + id
	tmplHost := "test-template-" + id

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{ // simple create
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
	name = %q 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_hostgroup.testgrp.id ]
	host = %q
}
resource "zabbix_item_simple" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "script[\"abc\"]"

	name = "Ext Item"
	valuetype = "text"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_simple.testitem", "key", "script[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_item_simple.testitem", "name", "Ext Item"),
					resource.TestCheckResourceAttr("zabbix_item_simple.testitem", "valuetype", "text"),
				),
			},
			{ // change values
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
	name = %q 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_hostgroup.testgrp.id ]
	host = %q
}
resource "zabbix_item_simple" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "scriptb[\"abc\"]"

	name = "Ext Item A"
	valuetype = "unsigned"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_simple.testitem", "key", "scriptb[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_item_simple.testitem", "name", "Ext Item A"),
					resource.TestCheckResourceAttr("zabbix_item_simple.testitem", "valuetype", "unsigned"),
				),
			},
		},
	})
}
