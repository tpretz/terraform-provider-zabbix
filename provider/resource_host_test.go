package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// These only run on zabbix versions >= 5.0
func TestAccResourceHostGT5(t *testing.T) {
	gt5 := os.Getenv("TEST_GT5")
	if gt5 == "" {
		return
	}
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{ // add a tag
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
	}
	tag {
		key = "testtag"
		value = "testvalue"
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.0.key", "testtag"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.0.value", "testvalue"),
				),
			},
			{ // change the tag values
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
	}
	tag {
		key = "testtag"
		value = "testvalue1"
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.0.key", "testtag"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.0.value", "testvalue1"),
				),
			},
			{ // add a second tag
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
	}
	tag {
		key = "testtagb"
		value = "testvalue2"
	}
	tag {
		key = "testtag"
		value = "testvalue1"
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.0.key", "testtag"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.0.value", "testvalue1"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.1.key", "testtagb"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.1.value", "testvalue2"),
				),
			},
			{ // snmp attributes, v1, also clear tags
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
		snmp_version = 1

		snmp_community = "testc"
		snmp_bulk = false
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp_version", "1"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp_community", "testc"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp_bulk", "false"),
				),
			},
			{ // snmp attributes, v2
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
		snmp_version = 2

		snmp_community = "testc"
		snmp_bulk = false
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp_version", "2"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp_community", "testc"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp_bulk", "false"),
				),
			},
			{ // snmp attributes, v3
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
		snmp_version = 3
		snmp_bulk = true

		snmp3_securityname = "testc"
		snmp3_securitylevel = "authpriv"
		snmp3_authpassphrase = "testauthp"
		snmp3_privpassphrase = "testprivp"
		snmp3_authprotocol = "sha"
		snmp3_privprotocol = "aes"
		snmp3_contextname = "testcname"
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp_bulk", "true"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp_version", "3"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp3_securityname", "testc"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp3_securitylevel", "authpriv"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp3_authpassphrase", "testauthp"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp3_privpassphrase", "testprivp"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp3_authprotocol", "sha"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp3_privprotocol", "aes"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp3_contextname", "testcname"),
				),
			},
			{ // snmp attributes, v3, change to some that eval to "0"
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
		snmp_version = 3
		snmp_bulk = true

		snmp3_securityname = "testc"
		snmp3_securitylevel = "noauthnopriv"
		snmp3_authpassphrase = "testauthp"
		snmp3_privpassphrase = "testprivp"
		snmp3_authprotocol = "md5"
		snmp3_privprotocol = "des"
		snmp3_contextname = "testcname"
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp_bulk", "true"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp_version", "3"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp3_securityname", "testc"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp3_securitylevel", "noauthnopriv"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp3_authpassphrase", "testauthp"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp3_privpassphrase", "testprivp"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp3_authprotocol", "md5"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp3_privprotocol", "des"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.snmp3_contextname", "testcname"),
				),
			},
		},
	})

}

func TestAccResourceHost(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{ // simple create
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "host", "test-host"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "inventory_mode", "disabled"),
				),
			},
			{ // enable inventory, set something
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
	}
	inventory_mode = "manual"
    inventory {
		location = "test location A"
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "inventory.0.location", "test location A"),
				),
			},
			{ // change something in inventory, also change mode of inventory
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
	}
	inventory_mode = "automatic"
    inventory {
		location = "test location B"
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "inventory.0.location", "test location B"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "inventory_mode", "automatic"),
				),
			},
			{ // add a second interface, change interface types, add a macro too
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host-renamed"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		dns = "localhost"
		port = 1234
	}

	interface {
		dns = "bob"
		type = "jmx"
	}

	macro {
		value = "fish"
		name = "{$BOB}"
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "host", "test-host-renamed"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "macro.0.value", "fish"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.dns", "localhost"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.type", "agent"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.0.port", "1234"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.1.dns", "bob"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.1.type", "jmx"),
				),
			},

			// relate to a proxy (tricky as we don't manage those resources ... yet, manual setup api call may be warrented)
			{ // relate to a template, and disable
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	host = "test-template"
	name = "test-template"
	groups = [ zabbix_hostgroup.testgrp.id ]
}
resource "zabbix_host" "testhost" {
	host   = "test-host-renamed"
	groups = [zabbix_hostgroup.testgrp.id]
	enabled = false
	interface {
		type = "agent"
		dns = "localhost"
		port = 1234
	}
	templates = [zabbix_template.testtmpl.id]

	interface {
		dns = "bob"
		type = "jmx"
	}

	macro {
		value = "fish"
		name = "{$BOB}"
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "enabled", "false"),
				),
			},
			{ // remove macros
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	host = "test-template"
	name = "test-template"
	groups = [ zabbix_hostgroup.testgrp.id ]
}
resource "zabbix_host" "testhost" {
	host   = "test-host-renamed"
	groups = [zabbix_hostgroup.testgrp.id]
	enabled = false
	interface {
		type = "agent"
		dns = "localhost"
		port = 1234
	}
	templates = [zabbix_template.testtmpl.id]

	interface {
		dns = "bob"
		type = "jmx"
	}
}
`,
			},
			// remove / replace templates (with items, check they are cleaned up)
		},
	})
}
