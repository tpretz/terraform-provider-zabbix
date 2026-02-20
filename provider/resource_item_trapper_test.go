package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceItemTrapper(t *testing.T) {
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
resource "zabbix_item_trapper" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "trapper[.*]"

	name = "Ext Item"
	valuetype = "text"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "key", "trapper[.*]"),
					resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "name", "Ext Item"),
					resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "valuetype", "text"),
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
resource "zabbix_item_trapper" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "trapper.fallback"

	name = "Ext Item A"
	valuetype = "unsigned"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "key", "trapper.fallback"),
					resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "name", "Ext Item A"),
					resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "valuetype", "unsigned"),
				),
			},
		},
	})
}
