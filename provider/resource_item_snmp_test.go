package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceItemSnmp(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
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
resource "zabbix_item_snmp" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "script[\"abc\"]"

	name = "Ext Item"
	valuetype = "text"

	snmp_oid = "1.2.2.5"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "key", "script[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "name", "Ext Item"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "valuetype", "text"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "snmp_oid", "1.2.2.5"),
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
resource "zabbix_item_snmp" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "script[\"abc\"]"

	name = "Ext Item"
	valuetype = "text"

	snmp_oid = "1.2.3.5"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "key", "script[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "name", "Ext Item"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "valuetype", "text"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "snmp_oid", "1.2.3.5"),
				),
			},
		},
	})
}
