package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// R1 for zabbix_host (PLAN.md § "The unit of work").
//
// Split off from acc_removal_test.go for the same reason acc_update_host_test.go
// is: hostResourceSchema copies every *schema.Schema out of hostSchemaBase, so
// the host's attributes are declarations of their own and nothing else in the
// provider covers them -- twenty of the sixty-nine defaults are here.
//
// The interface block takes three tests rather than one, because its defaults
// are not simultaneously reachable. `main` needs two interfaces of one type to
// show the revert at all (with one interface the default is the only legal
// value); `type` changes which of the snmp* attributes apply, so removing it
// in the same step as those would prove nothing about them; and the SNMP
// credentials split by version, since Zabbix keeps a v3 interface's security
// fields and a v1/v2 interface's community in the same "details" object but
// only honours the ones the version in force uses.

// TestAccRemoveHostDefaults owns the seven defaults on the host object itself.
//
// tls_connect and tls_accept are set to PSK rather than to certificates
// because a certificate host needs no further attributes, and the PSK pair
// does: the point is that deleting all four lines at once has to leave the
// server with neither the encryption mode nor the identity it went with.
func TestAccRemoveHostDefaults(t *testing.T) {
	const addr = "zabbix_host.testremhost"

	host := func(body string) string {
		return `
resource "zabbix_hostgroup" "testremgrp" {
	name = "test-removal-host-group"
}
resource "zabbix_proxy" "testremhostproxy" {
	name = "test-removal-host-proxy"
}
resource "zabbix_host" "testremhost" {
	host   = "test-removal-host"
	groups = [ zabbix_hostgroup.testremgrp.id ]
	interface {
		ip = "127.0.0.1"
	}
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
	enabled          = false
	inventory_mode   = "manual"
	ipmi_authtype    = "md5"
	ipmi_privilege   = "admin"
	proxyid          = zabbix_proxy.testremhostproxy.id
	tls_connect      = "psk"
	tls_accept       = "psk"
	tls_psk_identity = "test-removal-psk"
	tls_psk          = "b7e3b0e9a94e4f2a8d1c6f5b3a2e9d8c7b6a5f4e3d2c1b0a9f8e7d6c5b4a3f2e"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerAttrs(addr, serverHost, map[string]string{
						"status":         "1",
						"inventory_mode": "0",
						"ipmi_authtype":  "2",
						"ipmi_privilege": "4",
						"tls_connect":    "2",
						"tls_accept":     "2",
					}),
					testAccCheckHostProxy(addr, "zabbix_proxy.testremhostproxy"),
				),
			},
			{ // every line deleted: what is left is the documented minimum
				Config:           host(``),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerAttrs(addr, serverHost, map[string]string{
						"status":         "0",
						"inventory_mode": "-1",
						"ipmi_authtype":  "-1",
						"ipmi_privilege": "2",
						"tls_connect":    "1",
						"tls_accept":     "1",
					}),
					testAccCheckHostNoProxy(addr),
				),
			},
		},
	})
}

// testAccCheckHostNoProxy asserts the server has the host monitored by the
// Zabbix server itself. The property carrying that was renamed in 7.0, and
// both versions of it report "0", which is also what the schema's default
// says -- so this is the server-side half of removing the proxyid line.
func testAccCheckHostNoProxy(addr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		api, ok := testAccProvider.Meta().(*zabbix.API)
		if !ok || api == nil {
			return fmt.Errorf("testAccCheckHostNoProxy: provider not configured")
		}
		prop := "proxy_hostid"
		if api.Config.Version >= zabbix.V70 {
			prop = "proxyid"
		}
		return testAccCheckServerAttrs(addr, serverHost, map[string]string{prop: "0"})(s)
	}
}

// TestAccRemoveHostInterfaceDefaults owns the ten SNMP defaults on the
// interface block.
//
// Two SNMP interfaces rather than one, at different versions: Zabbix stores
// the v1/v2 community and the v3 credentials in the same "details" object but
// only returns the ones the interface's own snmp_version uses, so a single
// interface cannot show both sets of defaults arriving. snmp_version itself is
// deleted from the v1/v2 interface, where its default -- "2" -- is a value the
// server keeps using; deleting it from the v3 one would take the credentials
// out of scope in the same step that asserts them.
func TestAccRemoveHostInterfaceDefaults(t *testing.T) {
	const addr = "zabbix_host.testremifhost"

	host := func(v2, v3 string) string {
		return `
resource "zabbix_hostgroup" "testremifgrp" {
	name = "test-removal-interface-group"
}
resource "zabbix_host" "testremifhost" {
	host   = "test-removal-interface-host"
	groups = [ zabbix_hostgroup.testremifgrp.id ]
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
		port = 161
		main = true
` + v2 + `
	}
	interface {
		type         = "snmp"
		ip           = "127.0.0.1"
		port         = 1161
		main         = false
		snmp_version = "3"
` + v3 + `
	}
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
		snmp_version   = "1"
		snmp_bulk      = false
		snmp_community = "removalpublic"
`, `
		snmp3_securitylevel  = "authnopriv"
		snmp3_securityname   = "removaluser"
		snmp3_authprotocol   = "md5"
		snmp3_privprotocol   = "des"
		snmp3_authpassphrase = "removalauth"
		snmp3_privpassphrase = "removalpriv"
		snmp3_contextname    = "removalctx"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerElem(addr, serverHost, "interfaces", "port", "161", map[string]string{
						"details.version":   "1",
						"details.bulk":      "0",
						"details.community": "removalpublic",
					}),
					testAccCheckServerElem(addr, serverHost, "interfaces", "port", "1161", map[string]string{
						"details.securitylevel":  "1",
						"details.securityname":   "removaluser",
						"details.authprotocol":   "0",
						"details.privprotocol":   "0",
						"details.authpassphrase": "removalauth",
						"details.privpassphrase": "removalpriv",
						"details.contextname":    "removalctx",
					}),
				),
			},
			{ // every snmp line deleted from both
				Config:           host(``, ``),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerElem(addr, serverHost, "interfaces", "port", "161", map[string]string{
						"details.version":   "2",
						"details.bulk":      "1",
						"details.community": "{$SNMP_COMMUNITY}",
					}),
					testAccCheckServerElem(addr, serverHost, "interfaces", "port", "1161", map[string]string{
						"details.securitylevel":  "2",
						"details.securityname":   "{$SNMP3_SECURITYNAME}",
						"details.authprotocol":   "1",
						"details.privprotocol":   "1",
						"details.authpassphrase": "{$SNMP3_AUTHPASSPHRASE}",
						"details.privpassphrase": "{$SNMP3_PRIVPASSPHRASE}",
						"details.contextname":    "{$SNMP3_CONTEXTNAME}",
					}),
				),
			},
		},
	})
}

// TestAccRemoveHostInterfaceMain owns the interface block's `main`.
//
// It needs the two interfaces to swap roles rather than simply dropping a
// line: Zabbix requires exactly one primary interface per type, so deleting
// `main = false` while the other interface still says `main = true` is a
// configuration the server rejects, and the revert can only be shown by making
// room for it.
func TestAccRemoveHostInterfaceMain(t *testing.T) {
	const addr = "zabbix_host.testremmainhost"

	host := func(first, second string) string {
		return `
resource "zabbix_hostgroup" "testremmaingrp" {
	name = "test-removal-main-group"
}
resource "zabbix_host" "testremmainhost" {
	host   = "test-removal-main-host"
	groups = [ zabbix_hostgroup.testremmaingrp.id ]
	interface {
		ip   = "127.0.0.1"
		port = 10050
` + first + `
	}
	interface {
		ip   = "127.0.0.2"
		port = 10051
` + second + `
	}
}
`
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: host(`		main = true`, `		main = false`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerElem(addr, serverHost, "interfaces", "port", "10050", map[string]string{"main": "1"}),
					testAccCheckServerElem(addr, serverHost, "interfaces", "port", "10051", map[string]string{"main": "0"}),
				),
			},
			{ // the second interface's line deleted, and the first one told to
				// stand down so that the default has somewhere to land
				Config:           host(`		main = false`, ``),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerElem(addr, serverHost, "interfaces", "port", "10050", map[string]string{"main": "0"}),
					testAccCheckServerElem(addr, serverHost, "interfaces", "port", "10051", map[string]string{"main": "1"}),
				),
			},
		},
	})
}

// TestAccRemoveHostInterfaceType owns the interface block's `type`, which is
// the one default whose removal changes what the rest of the block means: an
// interface with no type is an agent interface, and its port follows.
func TestAccRemoveHostInterfaceType(t *testing.T) {
	const addr = "zabbix_host.testremtypehost"

	host := func(body string) string {
		return `
resource "zabbix_hostgroup" "testremtypegrp" {
	name = "test-removal-type-group"
}
resource "zabbix_host" "testremtypehost" {
	host   = "test-removal-type-host"
	groups = [ zabbix_hostgroup.testremtypegrp.id ]
	interface {
		ip = "127.0.0.1"
` + body + `
	}
}
`
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: host(`		type = "snmp"`),
				Check: testAccCheckServerElem(addr, serverHost, "interfaces", "type", "2", map[string]string{
					"port": "161",
				}),
			},
			{
				Config:           host(``),
				ConfigPlanChecks: expectUpdate(addr),
				Check: testAccCheckServerElem(addr, serverHost, "interfaces", "type", "1", map[string]string{
					"port": "10050",
				}),
			},
		},
	})
}
