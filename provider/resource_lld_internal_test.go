package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceLLDInternal(t *testing.T) {
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
resource "zabbix_lld_internal" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.internal[\"abc\"]"

	name = "LLD Internal Rule"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_internal.testlld", "key", "lld.internal[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_lld_internal.testlld", "name", "LLD Internal Rule"),
					resource.TestCheckResourceAttr("zabbix_lld_internal.testlld", "delay", "3600"),
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
resource "zabbix_lld_internal" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.internal[\"def\"]"

	name = "LLD Internal Rule Renamed"
	delay = "15m"
	lifetime = "45d"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_internal.testlld", "key", "lld.internal[\"def\"]"),
					resource.TestCheckResourceAttr("zabbix_lld_internal.testlld", "name", "LLD Internal Rule Renamed"),
					resource.TestCheckResourceAttr("zabbix_lld_internal.testlld", "delay", "15m"),
					resource.TestCheckResourceAttr("zabbix_lld_internal.testlld", "lifetime", "45d"),
				),
			},
			{ // import
				ResourceName:      "zabbix_lld_internal.testlld",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
