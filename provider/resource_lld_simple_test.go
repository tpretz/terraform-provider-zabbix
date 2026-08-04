package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceLLDSimple(t *testing.T) {
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
resource "zabbix_lld_simple" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.simple[\"abc\"]"

	name = "LLD Simple Rule"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_simple.testlld", "key", "lld.simple[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_lld_simple.testlld", "name", "LLD Simple Rule"),
					resource.TestCheckResourceAttr("zabbix_lld_simple.testlld", "delay", "3600"),
				),
			},
			{ // simple modify
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_lld_simple" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.simple[\"def\"]"

	name = "LLD Simple Rule Renamed"
	delay = "2m"
	lifetime = "14d"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_simple.testlld", "key", "lld.simple[\"def\"]"),
					resource.TestCheckResourceAttr("zabbix_lld_simple.testlld", "name", "LLD Simple Rule Renamed"),
					resource.TestCheckResourceAttr("zabbix_lld_simple.testlld", "delay", "2m"),
					resource.TestCheckResourceAttr("zabbix_lld_simple.testlld", "lifetime", "14d"),
				),
			},
			{ // import
				ResourceName:      "zabbix_lld_simple.testlld",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
