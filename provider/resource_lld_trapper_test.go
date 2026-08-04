package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceLLDTrapper(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // lazy init
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
`),
			},
			{ // simple create
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_lld_trapper" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.trapper[\"abc\"]"

	name = "LLD Trapper Rule"
	delay = "0"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "key", "lld.trapper[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "name", "LLD Trapper Rule"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "delay", "0"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "lifetime", "30d"),
				),
			},
			{ // modify: rename, change key, lifetime, add a filter condition (delay stays 0: required for trapper-type discovery rules)
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_lld_trapper" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.trapper[\"def\"]"

	name = "LLD Trapper Rule Renamed"
	delay = "0"
	lifetime = "7d"

	condition {
		macro = "{#FOO}"
		value = "bar"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "key", "lld.trapper[\"def\"]"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "name", "LLD Trapper Rule Renamed"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "delay", "0"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "lifetime", "7d"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "condition.0.macro", "{#FOO}"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "condition.0.value", "bar"),
				),
			},
			{ // import
				ResourceName:      "zabbix_lld_trapper.testlld",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
