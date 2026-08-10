package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceHost(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
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
					// `macro` is a set: by content. Plural coverage for the
					// shared common_macro.go machinery is in
					// TestAccResourceTemplateCollections.
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testhost", "macro.*", map[string]string{
						"name":  "{$BOB}",
						"value": "fish",
					}),
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
					// `tag` is a set: identify elements by content.
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testhost", "tag.*", map[string]string{
						"key":   "testtag",
						"value": "testvalue",
					}),
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
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testhost", "tag.*", map[string]string{
						"key":   "testtag",
						"value": "testvalue1",
					}),
				),
			},
			{ // C3: three tags, two of them sharing a key -- Zabbix allows a
				// repeated tag key with different values, so nothing but the
				// whole element identifies it
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
	tag {
		key = "testtag"
		value = "testvalue3"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testhost", "tag.*", map[string]string{
						"key":   "testtag",
						"value": "testvalue1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testhost", "tag.*", map[string]string{
						"key":   "testtag",
						"value": "testvalue3",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testhost", "tag.*", map[string]string{
						"key":   "testtagb",
						"value": "testvalue2",
					}),
				),
			},
			{ // C4: the same three in a different order -- must plan clean
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
		value = "testvalue3"
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
				PlanOnly: true,
			},
			{ // snmp attributes, v1, and C6: every tag removed. Checked
				// against the server too -- host.update replaces the tag array
				// wholesale, so state going empty proves nothing on its own.
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
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tag.#", "0"),
					testAccCheckHostTagCount("zabbix_host.testhost", 0),
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
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
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

// hostCollectionsHCL wraps a zabbix_host body in three host groups and three
// linkable templates.
//
// Note the two group types: below Zabbix 6.2 hcl() rewrites every
// "zabbix_templategroup" to "zabbix_hostgroup" textually, so the template
// group and the host groups must have distinct resource labels *and* distinct
// names or the two collide into one resource on 6.0.
func hostCollectionsHCL(body string) string {
	return `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_hostgroup" "testgrp2" {
	name = "test-group-2"
}
resource "zabbix_hostgroup" "testgrp3" {
	name = "test-group-3"
}
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group"
}
resource "zabbix_template" "testlink1" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template-link-1"
}
resource "zabbix_template" "testlink2" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template-link-2"
}
resource "zabbix_template" "testlink3" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template-link-3"
}
resource "zabbix_host" "testhost" {
	host = "test-host-collections"

	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
` + body + `}
`
}

// TestAccResourceHostCollections is C1-C7 for the two id sets a host carries:
// `groups` and `templates`.
//
// `macro` and `tag` are shared machinery (common_macro.go, common_tag.go) and
// are covered plural in TestAccResourceTemplateCollections and
// TestAccResourceItemAgentTags respectively; a host macro block is the same
// code as a template one. `interface` has its own multi-element test above.
// What is unique to the host here is the update path: removing a template
// requires templates_clear, which is not the same operation as omitting an id.
func TestAccResourceHostCollections(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // C2 for `groups`, C1 for `templates`. `groups` is Required and
				// Zabbix rejects a host with none, so C1 and C6-to-zero do not
				// apply to it.
				Config: hcl(t, hostCollectionsHCL(`
	groups = [ zabbix_hostgroup.testgrp.id ]
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "groups.#", "1"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "templates.#", "0"),
					testAccCheckHostGroupCount("zabbix_host.testhost", 1),
					testAccCheckHostTemplateCount("zabbix_host.testhost", 0),
				),
			},
			{ // C3: three groups and three linked templates
				Config: hcl(t, hostCollectionsHCL(`
	groups = [
		zabbix_hostgroup.testgrp.id,
		zabbix_hostgroup.testgrp2.id,
		zabbix_hostgroup.testgrp3.id,
	]

	templates = [
		zabbix_template.testlink1.id,
		zabbix_template.testlink2.id,
		zabbix_template.testlink3.id,
	]
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "groups.#", "3"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_host.testhost", "groups.*",
						"zabbix_hostgroup.testgrp", "id"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_host.testhost", "groups.*",
						"zabbix_hostgroup.testgrp2", "id"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_host.testhost", "groups.*",
						"zabbix_hostgroup.testgrp3", "id"),
					testAccCheckHostGroupCount("zabbix_host.testhost", 3),

					resource.TestCheckResourceAttr("zabbix_host.testhost", "templates.#", "3"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_host.testhost", "templates.*",
						"zabbix_template.testlink1", "id"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_host.testhost", "templates.*",
						"zabbix_template.testlink2", "id"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_host.testhost", "templates.*",
						"zabbix_template.testlink3", "id"),
					testAccCheckHostTemplateCount("zabbix_host.testhost", 3),
				),
			},
			{ // C4: both rewritten in a different order -- sets, so this must
				// plan clean
				Config: hcl(t, hostCollectionsHCL(`
	groups = [
		zabbix_hostgroup.testgrp3.id,
		zabbix_hostgroup.testgrp.id,
		zabbix_hostgroup.testgrp2.id,
	]

	templates = [
		zabbix_template.testlink3.id,
		zabbix_template.testlink1.id,
		zabbix_template.testlink2.id,
	]
`)),
				PlanOnly: true,
			},
			{ // C7: import with both at full size
				ResourceName:      "zabbix_host.testhost",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{ // C6, first half: one group and one template removed. The
				// template removal is the interesting one -- host.update
				// unlinks only what templates_clear names, so an id dropped
				// from `templates` and nothing else would leave the link in
				// place on the server while state claimed otherwise.
				Config: hcl(t, hostCollectionsHCL(`
	groups = [
		zabbix_hostgroup.testgrp.id,
		zabbix_hostgroup.testgrp3.id,
	]

	templates = [
		zabbix_template.testlink1.id,
		zabbix_template.testlink3.id,
	]
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "groups.#", "2"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "templates.#", "2"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_host.testhost", "templates.*",
						"zabbix_template.testlink1", "id"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_host.testhost", "templates.*",
						"zabbix_template.testlink3", "id"),
					testAccCheckHostGroupCount("zabbix_host.testhost", 2),
					testAccCheckHostTemplateCount("zabbix_host.testhost", 2),
				),
			},
			{ // C6, second half: every template unlinked, groups down to the
				// one Zabbix insists on
				Config: hcl(t, hostCollectionsHCL(`
	groups = [ zabbix_hostgroup.testgrp.id ]
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "groups.#", "1"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "templates.#", "0"),
					testAccCheckHostGroupCount("zabbix_host.testhost", 1),
					testAccCheckHostTemplateCount("zabbix_host.testhost", 0),
				),
			},
			{ // C1: and empty is stable
				Config: hcl(t, hostCollectionsHCL(`
	groups = [ zabbix_hostgroup.testgrp.id ]
`)),
				PlanOnly: true,
			},
		},
	})
}

// TestAccResourceHostTemplateClearDestroyed is existingTemplateIds' own case,
// with more than one template linked: one template is removed from the host's
// `templates` and destroyed in the same apply.
func TestAccResourceHostTemplateClearDestroyed(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // two templates linked
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group"
}
resource "zabbix_template" "testlink1" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template-link-1"
}
resource "zabbix_template" "testlink2" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template-link-2"
}
resource "zabbix_host" "testhost" {
	host   = "test-host-collections"
	groups = [ zabbix_hostgroup.testgrp.id ]

	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}

	templates = [
		zabbix_template.testlink1.id,
		zabbix_template.testlink2.id,
	]
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "templates.#", "2"),
					testAccCheckHostTemplateCount("zabbix_host.testhost", 2),
				),
			},
			{ // one of them unlinked and destroyed in the same apply
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group"
}
resource "zabbix_template" "testlink1" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template-link-1"
}
resource "zabbix_host" "testhost" {
	host   = "test-host-collections"
	groups = [ zabbix_hostgroup.testgrp.id ]

	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}

	templates = [
		zabbix_template.testlink1.id,
	]
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "templates.#", "1"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_host.testhost", "templates.*",
						"zabbix_template.testlink1", "id"),
					testAccCheckHostTemplateCount("zabbix_host.testhost", 1),
				),
			},
		},
	})
}

// TestAccResourceHostNoInterface covers C1 and C6-to-zero for host interfaces,
// both of which the Phase 7 audit recorded as "N/A: interface is Required,
// Min 1". That was wrong. Zabbix accepts a host with no interfaces at all on
// 6.0, 7.0, 7.4 and 8.0 — verified by direct API call — and a host carrying
// only calculated, dependent, trapper or internal items has nothing to attach
// one to. The provider used to reject it with "Insufficient interface blocks".
func TestAccResourceHostNoInterface(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // C1: no interface block at all
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.#", "0"),
					testAccCheckHostInterfaceCount("zabbix_host.testhost", 0),
				),
			},
			{ // an interface can still be added afterwards
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.#", "1"),
					testAccCheckHostInterfaceCount("zabbix_host.testhost", 1),
				),
			},
			{ // C6 to zero: and removed again, which omitempty used to prevent
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "interface.#", "0"),
					testAccCheckHostInterfaceCount("zabbix_host.testhost", 0),
				),
			},
			{
				ResourceName:      "zabbix_host.testhost",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
