package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccResourceItemCalculated(t *testing.T) {
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
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					fmt.Printf("got version %d\n", api.Config.Version)
					return api.Config.Version >= 50400, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [ zabbix_hostgroup.testgrp.id ]
  host   = %q
}
resource "zabbix_item_calculated" "testitem" {
  hostid = zabbix_template.testtmpl.id
  key    = "hello"

  name      = "Test Item"
  valuetype = "unsigned"

  formula = "avg(/Zabbix Server/zabbix[wcache,values],10m)"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_calculated.testitem", "key", "hello"),
					resource.TestCheckResourceAttr("zabbix_item_calculated.testitem", "name", "Test Item"),
					resource.TestCheckResourceAttr("zabbix_item_calculated.testitem", "valuetype", "unsigned"),
					resource.TestCheckResourceAttr("zabbix_item_calculated.testitem", "formula", "avg(/Zabbix Server/zabbix[wcache,values],10m)"),
				),
			},
			{ // change values
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version >= 50400, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [ zabbix_hostgroup.testgrp.id ]
  host   = %q
}
resource "zabbix_item_calculated" "testitem" {
  hostid = zabbix_template.testtmpl.id
  key    = "goodbye"

  name      = "Test Item Changed"
  valuetype = "float"
  formula   = "max(/Zabbix Server/zabbix[wcache,values],10m)"
}
`, groupName, tmplHost),
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
