package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceItemDependent(t *testing.T) {
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
resource "zabbix_item_agent" "parentitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitem"

	name = "Test Item"
	valuetype = "text"
}
resource "zabbix_item_dependent" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "depitem"

	name = "Dep Item"
	valuetype = "text"
	master_itemid = zabbix_item_agent.parentitem.id
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_dependent.testitem", "key", "depitem"),
					resource.TestCheckResourceAttr("zabbix_item_dependent.testitem", "name", "Dep Item"),
					resource.TestCheckResourceAttr("zabbix_item_dependent.testitem", "valuetype", "text"),
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
resource "zabbix_item_agent" "parentitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitem"

	name = "Test Item"
	valuetype = "text"
}
resource "zabbix_item_agent" "parentitem2" {
	hostid = zabbix_template.testtmpl.id
	key = "testitem2"

	name = "Test Item 2"
	valuetype = "text"
}
resource "zabbix_item_dependent" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "depitem"

	name = "Dep Item"
	valuetype = "text"
	master_itemid = zabbix_item_agent.parentitem2.id
}
`,
			},
		},
	})
}
