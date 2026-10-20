package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// testAccCheckItemTrapperDelay asserts the server holds "0" for a trapper
// item's delay, never the Go zero value (""): the item schema has no "delay"
// attribute for Terraform to assert through (trapper items are not polled),
// so this is the only way to catch itemTrapperModFunc leaving it unset again.
func testAccCheckItemTrapperDelay(addr string) resource.TestCheckFunc {
	return testAccCheckServerAttrs(addr, func(api *zabbix.API, id string) (map[string]interface{}, error) {
		res, err := api.ItemsGet(zabbix.Params{"itemids": []string{id}})
		if err != nil {
			return nil, err
		}
		if len(res) != 1 {
			return nil, fmt.Errorf("item.get returned %d items for id %s, want 1", len(res), id)
		}
		return map[string]interface{}{"delay": res[0].Delay}, nil
	}, map[string]string{"delay": "0"})
}

func TestAccResourceItemTrapper(t *testing.T) {
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
					testAccCheckItemTrapperDelay("zabbix_item_trapper.testitem"),
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
