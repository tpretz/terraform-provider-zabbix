package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceLLDExternal(t *testing.T) {
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
resource "zabbix_lld_external" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.external[\"abc\"]"

	name = "LLD External Rule"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_external.testlld", "key", "lld.external[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_lld_external.testlld", "name", "LLD External Rule"),
					resource.TestCheckResourceAttr("zabbix_lld_external.testlld", "delay", "3600"),
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
resource "zabbix_lld_external" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.external[\"def\"]"

	name = "LLD External Rule Renamed"
	delay = "10m"
	lifetime = "60d"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_external.testlld", "key", "lld.external[\"def\"]"),
					resource.TestCheckResourceAttr("zabbix_lld_external.testlld", "name", "LLD External Rule Renamed"),
					resource.TestCheckResourceAttr("zabbix_lld_external.testlld", "delay", "10m"),
					resource.TestCheckResourceAttr("zabbix_lld_external.testlld", "lifetime", "60d"),
				),
			},
			{ // import
				ResourceName:      "zabbix_lld_external.testlld",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
