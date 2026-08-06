package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceLLDSnmp(t *testing.T) {
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
			{ // simple create
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_lld_snmp" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.snmp[\"abc\"]"

	name = "LLD Snmp Rule"
	snmp_oid = "1.2.2.5"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_snmp.testlld", "key", "lld.snmp[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_lld_snmp.testlld", "name", "LLD Snmp Rule"),
					resource.TestCheckResourceAttr("zabbix_lld_snmp.testlld", "snmp_oid", "1.2.2.5"),
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
resource "zabbix_lld_snmp" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.snmp[\"abc\"]"

	name = "LLD Snmp Rule Renamed"
	snmp_oid = "1.2.3.5"
	delay = "5m"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_snmp.testlld", "name", "LLD Snmp Rule Renamed"),
					resource.TestCheckResourceAttr("zabbix_lld_snmp.testlld", "snmp_oid", "1.2.3.5"),
					resource.TestCheckResourceAttr("zabbix_lld_snmp.testlld", "delay", "5m"),
				),
			},
			{ // import
				ResourceName:      "zabbix_lld_snmp.testlld",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
