package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceLLDDependent(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // lazy init
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
`),
			},
			{ // simple create: master item plus a dependent LLD rule off it
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_trapper" "testmaster" {
	hostid = zabbix_template.testtmpl.id
	key = "trapper.master"

	name = "Master Item"
	valuetype = "text"
}
resource "zabbix_lld_dependent" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.dependent[\"abc\"]"

	name = "LLD Dependent Rule"
	master_itemid = zabbix_item_trapper.testmaster.id
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_dependent.testlld", "key", "lld.dependent[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_lld_dependent.testlld", "name", "LLD Dependent Rule"),
					resource.TestCheckResourceAttrPair("zabbix_lld_dependent.testlld", "master_itemid", "zabbix_item_trapper.testmaster", "id"),
				),
			},
			{ // simple modify: rename, second master item, re-point
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_trapper" "testmaster" {
	hostid = zabbix_template.testtmpl.id
	key = "trapper.master"

	name = "Master Item"
	valuetype = "text"
}
resource "zabbix_item_trapper" "testmaster2" {
	hostid = zabbix_template.testtmpl.id
	key = "trapper.master2"

	name = "Master Item 2"
	valuetype = "text"
}
resource "zabbix_lld_dependent" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.dependent[\"abc\"]"

	name = "LLD Dependent Rule Renamed"
	master_itemid = zabbix_item_trapper.testmaster2.id
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_dependent.testlld", "name", "LLD Dependent Rule Renamed"),
					resource.TestCheckResourceAttrPair("zabbix_lld_dependent.testlld", "master_itemid", "zabbix_item_trapper.testmaster2", "id"),
				),
			},
			{ // import
				ResourceName:      "zabbix_lld_dependent.testlld",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
