package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceItemCalculated(t *testing.T) {
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
resource "zabbix_item_calculated" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "hello"

	name = "Test Item"
	valuetype = "unsigned"

	formula = "avg(/Zabbix Server/zabbix[wcache,values],10m)"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_calculated.testitem", "key", "hello"),
					resource.TestCheckResourceAttr("zabbix_item_calculated.testitem", "name", "Test Item"),
					resource.TestCheckResourceAttr("zabbix_item_calculated.testitem", "valuetype", "unsigned"),
					resource.TestCheckResourceAttr("zabbix_item_calculated.testitem", "formula", "avg(/Zabbix Server/zabbix[wcache,values],10m)"),
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
resource "zabbix_item_calculated" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "goodbye"

	name = "Test Item Changed"
	valuetype = "float"
	formula = "max(/Zabbix Server/zabbix[wcache,values],10m)"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_calculated.testitem", "key", "goodbye"),
					resource.TestCheckResourceAttr("zabbix_item_calculated.testitem", "name", "Test Item Changed"),
					resource.TestCheckResourceAttr("zabbix_item_calculated.testitem", "valuetype", "float"),
					resource.TestCheckResourceAttr("zabbix_item_calculated.testitem", "formula", "max(/Zabbix Server/zabbix[wcache,values],10m)"),
				),
			},
		},
	})
}
