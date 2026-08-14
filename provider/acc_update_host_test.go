package provider

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// U1/U2 for zabbix_host (PLAN.md § "The unit of work").
//
// zabbix_host is the largest resource in the provider by some distance -- 110
// of the 228 attribute declarations, 71 of them inventory fields -- and unlike
// the item and discovery-rule families its attributes are NOT shared: the
// resource is built by hostResourceSchema, which copies every *schema.Schema
// out of hostSchemaBase, so each one is its own declaration. Nothing else in
// the provider covers them.
//
// Split three ways, by what each needs from the fixture rather than by taste:
// the host body, the interface block (which needs two interfaces of one type
// to show `main` moving), and inventory (which is 71 near-identical strings
// and is generated rather than written out).

// TestAccUpdateHost changes every attribute of the host object itself.
//
// tls_connect and tls_accept need three steps rather than two, because
// tls_issuer and tls_subject only apply to certificate encryption and
// tls_psk_identity and tls_psk only to PSK: a single pair of steps can reach
// one or the other but not both.
func TestAccUpdateHost(t *testing.T) {
	const addr = "zabbix_host.testhost"

	host := func(body string) string {
		return hcl(t, `
resource "zabbix_hostgroup" "testgrpa" {
	name = "test-update-host-group-a"
}
resource "zabbix_hostgroup" "testgrpb" {
	name = "test-update-host-group-b"
}
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-update-host-template-group"
}
resource "zabbix_template" "testtmpla" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host   = "test-update-host-template-a"
}
resource "zabbix_template" "testtmplb" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host   = "test-update-host-template-b"
}
resource "zabbix_proxy" "testproxya" {
	name           = "test-update-host-proxy-a"
	operating_mode = "active"
}
resource "zabbix_proxy" "testproxyb" {
	name           = "test-update-host-proxy-b"
	operating_mode = "active"
}
resource "zabbix_host" "testhost" {
	interface {
		ip = "127.0.0.1"
	}
`+body+`
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: host(`
	host           = "test-update-host-a"
	name           = "Update Host A"
	enabled        = true
	groups         = [ zabbix_hostgroup.testgrpa.id ]
	templates      = [ zabbix_template.testtmpla.id ]
	proxyid        = zabbix_proxy.testproxya.id
	inventory_mode = "manual"
	ipmi_authtype  = "default"
	ipmi_privilege = "user"
	ipmi_username  = "ipmi-a"
	ipmi_password  = "secret-a"
	tls_connect    = "unencrypted"
	tls_accept     = "unencrypted"
	macro {
		name  = "{$UPDATE_A}"
		value = "value-a"
	}
	tag {
		key   = "tag-a"
		value = "value-a"
	}
`),
				Check: testAccCheckServerAttrs(addr, serverHost, map[string]string{
					"host":           "test-update-host-a",
					"name":           "Update Host A",
					"status":         "0",
					"inventory_mode": "0",
					"ipmi_authtype":  "-1",
					"ipmi_privilege": "2",
					"ipmi_username":  "ipmi-a",
					"ipmi_password":  "secret-a",
					"tls_connect":    "1",
					"tls_accept":     "1",
				}),
			},
			{ // everything but the encryption pair, changed in life
				Config: host(`
	host           = "test-update-host-b"
	name           = "Update Host B"
	enabled        = false
	groups         = [ zabbix_hostgroup.testgrpb.id ]
	templates      = [ zabbix_template.testtmplb.id ]
	proxyid        = zabbix_proxy.testproxyb.id
	inventory_mode = "automatic"
	ipmi_authtype  = "md5"
	ipmi_privilege = "admin"
	ipmi_username  = "ipmi-b"
	ipmi_password  = "secret-b"
	tls_connect    = "cert"
	tls_accept     = "cert"
	tls_issuer     = "CN=issuer-b"
	tls_subject    = "CN=subject-b"
	macro {
		name  = "{$UPDATE_B}"
		value = "value-b"
	}
	tag {
		key   = "tag-b"
		value = "value-b"
	}
`),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerAttrs(addr, serverHost, map[string]string{
						"host":           "test-update-host-b",
						"name":           "Update Host B",
						"status":         "1",
						"inventory_mode": "1",
						"ipmi_authtype":  "2",
						"ipmi_privilege": "4",
						"ipmi_username":  "ipmi-b",
						"ipmi_password":  "secret-b",
						"tls_connect":    "4",
						"tls_accept":     "4",
						"tls_issuer":     "CN=issuer-b",
						"tls_subject":    "CN=subject-b",
					}),
					testAccCheckHostProxy(addr, "zabbix_proxy.testproxyb"),
					testAccCheckHostLinkedTo(addr, "zabbix_template.testtmplb"),
					testAccCheckHostInGroup(addr, "zabbix_hostgroup.testgrpb"),
					testAccCheckServerElem(addr, serverHost, "macros", "macro", "{$UPDATE_B}", map[string]string{
						"value": "value-b",
					}),
					testAccCheckServerElem(addr, serverHost, "tags", "tag", "tag-b", map[string]string{
						"value": "value-b",
					}),
				),
			},
			{ // the PSK half of the encryption pair. tls_psk_identity and
				// tls_psk are write-only -- no supported version returns them
				// from host.get -- so the server-side assertion here is the
				// mode, and the values themselves are asserted in state.
				Config: host(`
	host             = "test-update-host-b"
	name             = "Update Host B"
	enabled          = false
	groups           = [ zabbix_hostgroup.testgrpb.id ]
	templates        = [ zabbix_template.testtmplb.id ]
	proxyid          = zabbix_proxy.testproxyb.id
	inventory_mode   = "automatic"
	tls_connect      = "psk"
	tls_accept       = "psk"
	tls_psk_identity = "update-psk-b"
	tls_psk          = "cafe1234cafe1234cafe1234cafe1234"
`),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerAttrs(addr, serverHost, map[string]string{
						"tls_connect": "2",
						"tls_accept":  "2",
						"tls_issuer":  "",
						"tls_subject": "",
					}),
					resource.TestCheckResourceAttr(addr, "tls_psk_identity", "update-psk-b"),
					resource.TestCheckResourceAttr(addr, "tls_psk", "cafe1234cafe1234cafe1234cafe1234"),
				),
			},
		},
	})
}

// testAccCheckHostProxy asserts which proxy the server has the host on.
// ProxiesGet is not involved: the host side of the link is what matters, and
// its property name changed in 7.0, so the check reads whichever the server
// returned.
func testAccCheckHostProxy(addr, proxyAddr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		proxyID, err := testAccStateID(s, proxyAddr)
		if err != nil {
			return err
		}
		api, ok := testAccProvider.Meta().(*zabbix.API)
		if !ok || api == nil {
			return fmt.Errorf("testAccCheckHostProxy: provider not configured")
		}
		prop := "proxy_hostid"
		if api.Config.Version >= zabbix.V70 {
			prop = "proxyid"
		}
		return testAccCheckServerAttrs(addr, serverHost, map[string]string{prop: proxyID})(s)
	}
}

// testAccCheckHostLinkedTo asserts the host's linked templates as the server
// holds them, by the id of the template it should be linked to.
func testAccCheckHostLinkedTo(addr, tmplAddr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccStateID(s, tmplAddr)
		if err != nil {
			return err
		}
		return testAccCheckServerElem(addr, serverHost, "parentTemplates", "templateid", id, map[string]string{
			"templateid": id,
		})(s)
	}
}

// testAccCheckHostInGroup asserts the host's groups as the server holds them.
// The property Zabbix returns them under was renamed in 7.2, so the check
// tries whichever one came back.
func testAccCheckHostInGroup(addr, groupAddr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccStateID(s, groupAddr)
		if err != nil {
			return err
		}
		api, ok := testAccProvider.Meta().(*zabbix.API)
		if !ok || api == nil {
			return fmt.Errorf("testAccCheckHostInGroup: provider not configured")
		}
		prop := "groups"
		if api.Config.Version >= zabbix.V72 {
			prop = "hostgroups"
		}
		return testAccCheckServerElem(addr, serverHost, prop, "groupid", id, map[string]string{
			"groupid": id,
		})(s)
	}
}

// TestAccUpdateHostInterface changes every attribute of the interface block.
//
// Five interfaces, because the block cannot be covered with fewer:
//
//   - two agent interfaces, so that `main` can be shown moving. Zabbix
//     requires exactly one main interface per type, so a single interface can
//     never have main changed at all.
//   - an SNMP v2 interface for snmp_community and snmp_bulk, and a separate
//     SNMP v3 one for the six snmp3_* attributes. Keeping them apart means
//     each attribute has a real before and after rather than appearing and
//     disappearing with the version.
//   - a fifth interface that changes type outright, jmx to ipmi, which is the
//     only way `type` is ever changed in life. In the TypeSet model that is a
//     removal plus an addition within one host update; Zabbix does not let an
//     interface change type in place.
func TestAccUpdateHostInterface(t *testing.T) {
	const addr = "zabbix_host.testhost"

	host := func(body string) string {
		return `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-update-hostiface-group"
}
resource "zabbix_host" "testhost" {
	host   = "test-update-hostiface-host"
	groups = [ zabbix_hostgroup.testgrp.id ]
` + body + `
}
`
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: host(`
	interface {
		type = "agent"
		ip   = "127.0.0.1"
		port = 10050
		main = true
	}
	interface {
		type = "agent"
		dns  = "agent-b.example.com"
		port = 10051
		main = false
	}
	interface {
		type           = "snmp"
		ip             = "127.0.0.3"
		port           = 161
		main           = true
		snmp_version   = "2"
		snmp_community = "community-a"
		snmp_bulk      = true
	}
	interface {
		type                 = "snmp"
		ip                   = "127.0.0.5"
		port                 = 1161
		main                 = false
		snmp_version         = "3"
		snmp3_securityname   = "sec-name-a"
		snmp3_securitylevel  = "authnopriv"
		snmp3_authprotocol   = "sha"
		snmp3_authpassphrase = "auth-pass-a"
		snmp3_privprotocol   = "aes"
		snmp3_privpassphrase = "priv-pass-a"
		snmp3_contextname    = "context-a"
	}
	interface {
		type = "jmx"
		dns  = "jmx.example.com"
		port = 8686
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHostInterfaceCount(addr, 5),
					testAccCheckServerElem(addr, serverHost, "interfaces", "port", "10050", map[string]string{
						"type":  "1",
						"ip":    "127.0.0.1",
						"main":  "1",
						"useip": "1",
					}),
					testAccCheckServerElem(addr, serverHost, "interfaces", "port", "8686", map[string]string{
						"type": "4", // jmx
					}),
				),
			},
			{ // every attribute of every interface changed, in life
				Config: host(`
	interface {
		type = "agent"
		dns  = "agent-a.example.com"
		port = 10060
		main = false
	}
	interface {
		type = "agent"
		ip   = "127.0.0.2"
		port = 10061
		main = true
	}
	interface {
		type           = "snmp"
		ip             = "127.0.0.4"
		port           = 162
		main           = true
		snmp_version   = "2"
		snmp_community = "community-b"
		snmp_bulk      = false
	}
	interface {
		type                 = "snmp"
		ip                   = "127.0.0.6"
		port                 = 1162
		main                 = false
		snmp_version         = "3"
		snmp3_securityname   = "sec-name-b"
		snmp3_securitylevel  = "authpriv"
		snmp3_authprotocol   = "md5"
		snmp3_authpassphrase = "auth-pass-b"
		snmp3_privprotocol   = "des"
		snmp3_privpassphrase = "priv-pass-b"
		snmp3_contextname    = "context-b"
	}
	interface {
		type = "ipmi"
		ip   = "127.0.0.7"
		port = 623
	}
`),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHostInterfaceCount(addr, 5),
					// the agent interface that is now dns-addressed and not main
					testAccCheckServerElem(addr, serverHost, "interfaces", "port", "10060", map[string]string{
						"type":  "1",
						"dns":   "agent-a.example.com",
						"ip":    "",
						"main":  "0",
						"useip": "0",
					}),
					// and the one main moved to
					testAccCheckServerElem(addr, serverHost, "interfaces", "port", "10061", map[string]string{
						"type":  "1",
						"ip":    "127.0.0.2",
						"main":  "1",
						"useip": "1",
					}),
					testAccCheckServerElem(addr, serverHost, "interfaces", "port", "162", map[string]string{
						"type":              "2",
						"ip":                "127.0.0.4",
						"details.version":   "2",
						"details.community": "community-b",
						"details.bulk":      "0",
					}),
					testAccCheckServerElem(addr, serverHost, "interfaces", "port", "1162", map[string]string{
						"type":                   "2",
						"ip":                     "127.0.0.6",
						"details.version":        "3",
						"details.securityname":   "sec-name-b",
						"details.securitylevel":  "2",
						"details.authprotocol":   "0",
						"details.authpassphrase": "auth-pass-b",
						"details.privprotocol":   "0",
						"details.privpassphrase": "priv-pass-b",
						"details.contextname":    "context-b",
					}),
					// the interface whose type changed outright
					testAccCheckServerElem(addr, serverHost, "interfaces", "port", "623", map[string]string{
						"type": "3", // ipmi
						"ip":   "127.0.0.7",
					}),
				),
			},
		},
	})
}

// TestAccUpdateHostInventory changes all 71 inventory fields at once.
//
// The configuration is generated from the schema rather than written out, for
// the same reason the coverage guard is generated from the schema: a field
// added to the inventory block is then covered without anybody remembering to
// add it, and cannot sit uncovered behind a fixture that looks complete.
func TestAccUpdateHostInventory(t *testing.T) {
	const addr = "zabbix_host.testhost"

	host := func(suffix string) string {
		return `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-update-hostinv-group"
}
resource "zabbix_host" "testhost" {
	host           = "test-update-hostinv-host"
	groups         = [ zabbix_hostgroup.testgrp.id ]
	inventory_mode = "manual"
	interface {
		ip = "127.0.0.1"
	}
	inventory {
` + hostInventoryHCL(suffix) + `	}
}
`
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: host("a"),
				Check:  testAccCheckServerAttrs(addr, serverHost, hostInventoryWant("a")),
			},
			{ // all 71 changed, in life
				Config:           host("b"),
				ConfigPlanChecks: expectUpdate(addr),
				Check:            testAccCheckServerAttrs(addr, serverHost, hostInventoryWant("b")),
			},
		},
	})
}

// hostInventoryFields lists the inventory attributes straight out of the
// schema, so the fixture cannot fall behind it.
func hostInventoryFields() []string {
	res, ok := Provider().ResourcesMap["zabbix_host"].Schema["inventory"].Elem.(*schema.Resource)
	if !ok {
		return nil
	}
	fields := make([]string, 0, len(res.Schema))
	for k := range res.Schema {
		fields = append(fields, k)
	}
	sort.Strings(fields)
	return fields
}

// hostInventoryValue is the value written to one inventory field. Zabbix
// validates none of these as dates or numbers -- every inventory field is a
// free-text column on every supported version -- but the values are kept short
// because a few of the columns are only 64 characters wide.
func hostInventoryValue(field, suffix string) string {
	return field + "-" + suffix
}

func hostInventoryHCL(suffix string) string {
	var b strings.Builder
	for _, f := range hostInventoryFields() {
		fmt.Fprintf(&b, "\t\t%s = %q\n", f, hostInventoryValue(f, suffix))
	}
	return b.String()
}

func hostInventoryWant(suffix string) map[string]string {
	want := map[string]string{}
	for _, f := range hostInventoryFields() {
		want["inventory."+f] = hostInventoryValue(f, suffix)
	}
	return want
}
