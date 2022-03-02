package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccResourceItemSnmp(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{ // lazy init
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
`,
			},

			// ge version 5, just simple snmp options onlyu

			// lt v5 all attributes are on this

			{ // simple create
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version >= 50000, nil
				},
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_hostgroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_snmp" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "script[\"abc\"]"

	name = "Ext Item"
	valuetype = "text"

	snmp_oid = "1.2.2.4"
	snmp_version = "2"
	#snmp_community = "cheese"
}
resource "zabbix_item_snmp" "testitem3" {
	hostid = zabbix_template.testtmpl.id
	key = "script[\"abcdef\"]"

	name = "Ext Item 3"
	valuetype = "text"

	snmp_oid = "1.2.2.5"
	snmp_version = "3"
	#snmp3_authpassphrase = ""
	#snmp3_authprotocol = ""
	#snmp3_contextname = ""
	#snmp3_privpassphrase = ""
	#snmp3_privprotocol = ""
	#snmp3_securitylevel = ""
	#snmp3_securityname = ""
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "key", "script[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "name", "Ext Item"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "valuetype", "text"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "snmp_version", "2"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "snmp_community", "{$SNMP_COMMUNITY}"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "snmp_oid", "1.2.2.4"),

					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp_oid", "1.2.2.5"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp_version", "3"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp3_authpassphrase", "{$SNMP3_AUTHPASSPHRASE}"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp3_authprotocol", "sha"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp3_contextname", "{$SNMP3_CONTEXTNAME}"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp3_privpassphrase", "{$SNMP3_PRIVPASSPHRASE}"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp3_privprotocol", "aes"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp3_securitylevel", "authpriv"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp3_securityname", "{$SNMP3_SECURITYNAME}"),
				),
			},
			{ // change from defaults
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version < 50000, nil
				},
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_hostgroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_snmp" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "script[\"abc\"]"

	name = "Ext Item"
	valuetype = "text"

	snmp_oid = "1.2.3.4"
	snmp_version = "2"
	snmp_community = "cheese"
}
resource "zabbix_item_snmp" "testitem3" {
	hostid = zabbix_template.testtmpl.id
	key = "script[\"abcdef\"]"

	name = "Ext Item 3"
	valuetype = "text"

	snmp_oid = "1.2.3.5"
	snmp_version = "3"
	snmp3_authpassphrase = "bob"
	snmp3_authprotocol = "md5"
	snmp3_contextname = "fish"
	snmp3_privpassphrase = "cheese"
	snmp3_privprotocol = "des"
	snmp3_securitylevel = "authnopriv"
	snmp3_securityname = "dave"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "key", "script[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "name", "Ext Item"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "valuetype", "text"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "snmp_version", "2"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "snmp_community", "cheese"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "snmp_oid", "1.2.3.4"),

					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp_oid", "1.2.3.5"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp_version", "3"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp3_authpassphrase", "bob"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp3_authprotocol", "md5"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp3_contextname", "fish"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp3_privpassphrase", "cheese"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp3_privprotocol", "des"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp3_securitylevel", "authnopriv"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem3", "snmp3_securityname", "dave"),
				),
			},
			{ // simple create (>=v5)
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version < 50000, nil
				},
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_hostgroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_snmp" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "script[\"abc\"]"

	name = "Ext Item"
	valuetype = "text"

	snmp_oid = "1.2.2.5"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "key", "script[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "name", "Ext Item"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "valuetype", "text"),
					resource.TestCheckResourceAttr("zabbix_item_snmp.testitem", "snmp_oid", "1.2.2.5"),
				),
			},
			{ // simple modify (>=v5)
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version < 50000, nil
				},
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_hostgroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_snmp" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "script[\"abc\"]"

	name = "Ext Item"
	valuetype = "text"

	snmp_oid = "1.2.3.5"
}
`,
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
