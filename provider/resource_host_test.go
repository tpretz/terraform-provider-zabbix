package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceHost(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // simple create
				Config: hcl(t, `
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
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "host", "test-host"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "inventory_mode", "disabled"),
				),
			},
			{ // enable inventory, set something
				Config: hcl(t, `
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
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "inventory.0.location", "test location A"),
				),
			},
			{ // change something in inventory, also change mode of inventory
				Config: hcl(t, `
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
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "inventory.0.location", "test location B"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "inventory_mode", "automatic"),
				),
			},
			{ // add a second interface, change interface types, add a macro too
				Config: hcl(t, `
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
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "host", "test-host-renamed"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "macro.0.value", "fish"),
					// `interface` is a set: identify elements by content, not
					// by position.
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testhost", "interface.*", map[string]string{
						"dns":  "localhost",
						"type": "agent",
						"port": "1234",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testhost", "interface.*", map[string]string{
						"dns":  "bob",
						"type": "jmx",
						"port": "8686",
					}),
				),
			},

			// relate to a proxy (tricky as we don't manage those resources ... yet, manual setup api call may be warrented)
			{ // relate to a template, and disable
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group" 
}
resource "zabbix_template" "testtmpl" {
	host = "test-template"
	name = "test-template"
	groups = [ zabbix_templategroup.testtmplgrp.id ]
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
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "enabled", "false"),
				),
			},
			{ // remove macros
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group" 
}
resource "zabbix_template" "testtmpl" {
	host = "test-template"
	name = "test-template"
	groups = [ zabbix_templategroup.testtmplgrp.id ]
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
`),
			},
			{ // add a tag
				Config: hcl(t, `
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
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.0.key", "testtag"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.0.value", "testvalue"),
				),
			},
			{ // change the tag values
				Config: hcl(t, `
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
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.0.key", "testtag"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.0.value", "testvalue1"),
				),
			},
			{ // add a second tag
				Config: hcl(t, `
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
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.0.key", "testtag"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.0.value", "testvalue1"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.1.key", "testtagb"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.1.value", "testvalue2"),
				),
			},
			{ // snmp attributes, v1, also clear tags
				Config: hcl(t, `
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
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testhost", "interface.*", map[string]string{
						"snmp_version":   "1",
						"snmp_community": "testc",
						"snmp_bulk":      "false",
					}),
				),
			},
			{ // snmp attributes, v2
				Config: hcl(t, `
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
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testhost", "interface.*", map[string]string{
						"snmp_version":   "2",
						"snmp_community": "testc",
						"snmp_bulk":      "false",
					}),
				),
			},
			{ // snmp attributes, v3
				Config: hcl(t, `
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
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testhost", "interface.*", map[string]string{
						"snmp_bulk":            "true",
						"snmp_version":         "3",
						"snmp3_securityname":   "testc",
						"snmp3_securitylevel":  "authpriv",
						"snmp3_authpassphrase": "testauthp",
						"snmp3_privpassphrase": "testprivp",
						"snmp3_authprotocol":   "sha",
						"snmp3_privprotocol":   "aes",
						"snmp3_contextname":    "testcname",
					}),
				),
			},
			{ // snmp attributes, v3, change to some that eval to "0"
				Config: hcl(t, `
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
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testhost", "interface.*", map[string]string{
						"snmp_bulk":            "true",
						"snmp_version":         "3",
						"snmp3_securityname":   "testc",
						"snmp3_securitylevel":  "noauthnopriv",
						"snmp3_authpassphrase": "testauthp",
						"snmp3_privpassphrase": "testprivp",
						"snmp3_authprotocol":   "md5",
						"snmp3_privprotocol":   "des",
						"snmp3_contextname":    "testcname",
					}),
				),
			},
			{ // IPMI access and certificate encryption
				Config: hcl(t, `
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

	ipmi_authtype  = "md5"
	ipmi_privilege = "admin"
	ipmi_username  = "ipmiuser"
	ipmi_password  = "ipmipass"

	tls_connect = "cert"
	tls_accept  = "cert"
	tls_issuer  = "CN=Test CA"
	tls_subject = "CN=test-host"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "ipmi_authtype", "md5"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "ipmi_privilege", "admin"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "ipmi_username", "ipmiuser"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "ipmi_password", "ipmipass"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tls_connect", "cert"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tls_accept", "cert"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tls_issuer", "CN=Test CA"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tls_subject", "CN=test-host"),
				),
			},
			{ // import: everything above round-trips, including ipmi_password,
				// which host.get does return
				ResourceName:      "zabbix_host.testhost",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{ // switch to pre-shared key encryption
				Config: hcl(t, `
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

	tls_connect      = "psk"
	tls_accept       = "psk"
	tls_psk_identity = "test-psk-id"
	tls_psk          = "0123456789abcdef0123456789abcdef"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tls_connect", "psk"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tls_accept", "psk"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tls_psk_identity", "test-psk-id"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tls_psk", "0123456789abcdef0123456789abcdef"),
					// the cert attributes are cleared by the same update
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tls_issuer", ""),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tls_subject", ""),
					// and IPMI reverts to the Zabbix defaults
					resource.TestCheckResourceAttr("zabbix_host.testhost", "ipmi_authtype", "default"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "ipmi_privilege", "user"),
				),
			},
			{ // tls_psk_identity and tls_psk are the only attributes excluded
				// from import verification: host.get never returns them on any
				// version, even when asked for by name, so an imported host
				// cannot know what they were.
				ResourceName:            "zabbix_host.testhost",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"tls_psk_identity", "tls_psk"},
			},
			// remove / replace templates (with items, check they are cleaned up)
		},
	})
}

// TestAccResourceHostMultiInterface covers what every other host test misses:
// a host with several interfaces, including two of the same type.
//
// `interface` is a TypeSet, so neither the order the blocks are written in nor
// the order host.get hands them back may affect the result. A single-interface
// fixture cannot show that; this one reorders the blocks between steps and
// requires the plan to stay empty.
func TestAccResourceHostMultiInterface(t *testing.T) {
	const groupHCL = `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group"
}
`

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // four interfaces, two of them SNMP: only one interface of a
				// type may be `main`
				Config: hcl(t, groupHCL+`
resource "zabbix_host" "testhost" {
	host   = "test-host-multi"
	groups = [zabbix_hostgroup.testgrp.id]

	interface {
		type = "agent"
		dns  = "localhost"
		port = 1234
	}
	interface {
		type = "jmx"
		dns  = "jmx.example.com"
	}
	interface {
		type           = "snmp"
		ip             = "127.0.0.5"
		snmp_community = "public"
	}
	interface {
		type           = "snmp"
		ip             = "127.0.0.6"
		main           = false
		snmp_community = "private"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.#", "4"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testhost", "interface.*", map[string]string{
						"type": "agent",
						"dns":  "localhost",
						"port": "1234",
						"main": "true",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testhost", "interface.*", map[string]string{
						"type": "jmx",
						"dns":  "jmx.example.com",
						"port": "8686",
						"main": "true",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testhost", "interface.*", map[string]string{
						"type":           "snmp",
						"ip":             "127.0.0.5",
						"port":           "161",
						"main":           "true",
						"snmp_community": "public",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testhost", "interface.*", map[string]string{
						"type":           "snmp",
						"ip":             "127.0.0.6",
						"port":           "161",
						"main":           "false",
						"snmp_community": "private",
					}),
				),
			},
			{ // the same four interfaces in a different order: a set has no
				// order, so this must plan clean
				Config: hcl(t, groupHCL+`
resource "zabbix_host" "testhost" {
	host   = "test-host-multi"
	groups = [zabbix_hostgroup.testgrp.id]

	interface {
		type           = "snmp"
		ip             = "127.0.0.6"
		main           = false
		snmp_community = "private"
	}
	interface {
		type = "jmx"
		dns  = "jmx.example.com"
	}
	interface {
		type           = "snmp"
		ip             = "127.0.0.5"
		snmp_community = "public"
	}
	interface {
		type = "agent"
		dns  = "localhost"
		port = 1234
	}
}
`),
				PlanOnly: true,
			},
			{ // drop one of the two SNMP interfaces and edit another's
				// credentials in the same apply
				Config: hcl(t, groupHCL+`
resource "zabbix_host" "testhost" {
	host   = "test-host-multi"
	groups = [zabbix_hostgroup.testgrp.id]

	interface {
		type = "agent"
		dns  = "localhost"
		port = 1234
	}
	interface {
		type = "jmx"
		dns  = "jmx.example.com"
	}
	interface {
		type           = "snmp"
		ip             = "127.0.0.5"
		snmp_community = "changed"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testhost", "interface.*", map[string]string{
						"type":           "snmp",
						"ip":             "127.0.0.5",
						"snmp_community": "changed",
					}),
				),
			},
			{ // import: all three interfaces round-trip
				ResourceName:      "zabbix_host.testhost",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
