package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// Clearability -- can what the user set be unset again?
//
// This is the scalar twin of `C6` and it has one cause: **Zabbix reads an
// absent property as "leave as is".** An `omitempty` struct tag on a property
// the user is allowed to empty therefore means the clear is never sent, the
// server keeps what it had, the read puts it straight back into state, and the
// user gets a diff that reapplies forever and never converges. Nothing about
// setting the value ever fails, which is why every criterion that asks "does
// this attribute behave correctly when set?" walks past it.
//
// TestAccBoundaryHttpItemEmptyStrings (E6) covers the four HTTP strings that
// have no default. The attributes here are the ones a *default* hides: the
// user has to write the empty value out deliberately, so no fixture ever did.
//
// Every test ends by returning the attribute to its default, and the
// harness's own empty-plan requirement after each step is the assertion --
// a clear that does not reach the server fails the step by itself.

const clearableTemplateHCL = `
resource "zabbix_templategroup" "testcleargrp" {
	name = "test-clearable-template-group"
}
resource "zabbix_template" "testcleartmpl" {
	groups = [ zabbix_templategroup.testcleargrp.id ]
	host   = "test-clearable-template"
}
`

// TestAccClearableHttpItemStatusCodes -- `status_codes` defaults to "200", and
// "" is the Zabbix way to say "accept any response code". Every supported
// server accepts the empty value; with `omitempty` on Item.StatusCodes the
// provider could never send it.
func TestAccClearableHttpItemStatusCodes(t *testing.T) {
	item := func(body string) string {
		return hcl(t, clearableTemplateHCL+`
resource "zabbix_item_http" "testclear" {
	hostid    = zabbix_template.testcleartmpl.id
	key       = "test.clear.http"
	name      = "Test Clearable Http"
	valuetype = "text"
	url       = "http://localhost/clear"
`+body+`
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // the default
				Config: item(``),
				Check: resource.TestCheckResourceAttr(
					"zabbix_item_http.testclear", "status_codes", "200"),
			},
			{ // a non-default value, so the clear has something to undo
				Config: item(`	status_codes = "200,201"`),
				Check: resource.TestCheckResourceAttr(
					"zabbix_item_http.testclear", "status_codes", "200,201"),
			},
			{ // the clear
				Config: item(`	status_codes = ""`),
				Check: resource.TestCheckResourceAttr(
					"zabbix_item_http.testclear", "status_codes", ""),
			},
			{ // and back to the default, which is the same journey in reverse
				Config: item(``),
				Check: resource.TestCheckResourceAttr(
					"zabbix_item_http.testclear", "status_codes", "200"),
			},
		},
	})
}

// TestAccClearableHttpItemTimeout -- 7.0 gave items a global timeout setting
// and made "" mean "use it". 6.0 has no such thing and rejects the empty
// value outright ("Invalid parameter /timeout: cannot be empty"), so the
// clear is gated -- but the *provider* must be able to express it either way,
// and on 6.0 the user should get that server error rather than a silent
// no-op followed by a diff that never settles.
func TestAccClearableHttpItemTimeout(t *testing.T) {
	item := func(body string) string {
		return hcl(t, clearableTemplateHCL+`
resource "zabbix_item_http" "testclear" {
	hostid    = zabbix_template.testcleartmpl.id
	key       = "test.clear.http.timeout"
	name      = "Test Clearable Http Timeout"
	valuetype = "text"
	url       = "http://localhost/clear"
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
				Config:   item(`	timeout = "10s"`),
				SkipFunc: skipBelow(t, zabbix.V70),
				Check: resource.TestCheckResourceAttr(
					"zabbix_item_http.testclear", "timeout", "10s"),
			},
			{
				Config:   item(`	timeout = ""`),
				SkipFunc: skipBelow(t, zabbix.V70),
				Check: resource.TestCheckResourceAttr(
					"zabbix_item_http.testclear", "timeout", ""),
			},
			{
				Config:   item(``),
				SkipFunc: skipBelow(t, zabbix.V70),
				Check: resource.TestCheckResourceAttr(
					"zabbix_item_http.testclear", "timeout", "3s"),
			},
		},
	})
}

// TestAccClearableLLDHttpStatusCodes -- discovery rules do not share the item
// write path. LLDRule carries its own copy of the HTTP fields, so the fix on
// one says nothing about the other.
func TestAccClearableLLDHttpStatusCodes(t *testing.T) {
	rule := func(body string) string {
		return hcl(t, clearableTemplateHCL+`
resource "zabbix_lld_http" "testclear" {
	hostid = zabbix_template.testcleartmpl.id
	key    = "test.clear.lld.http"
	name   = "Test Clearable LLD Http"
	url    = "http://localhost/clear/discovery"
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
				Config: rule(`	status_codes = "200,201"`),
				Check: resource.TestCheckResourceAttr(
					"zabbix_lld_http.testclear", "status_codes", "200,201"),
			},
			{
				Config: rule(`	status_codes = ""`),
				Check: resource.TestCheckResourceAttr(
					"zabbix_lld_http.testclear", "status_codes", ""),
			},
			{
				Config: rule(``),
				Check: resource.TestCheckResourceAttr(
					"zabbix_lld_http.testclear", "status_codes", "200"),
			},
		},
	})
}

// TestAccClearableHostInventory -- host inventory is a map, and Zabbix merges
// what it is sent into what it has. A field left out of the object is
// therefore kept, exactly as an absent scalar property is, so emptying one
// inventory field has to be an explicit "" on the wire.
//
// This is the same failure mode arriving by a different route: the write path
// builds the map with d.GetOk, which reports an empty string as "not set" and
// drops the key.
func TestAccClearableHostInventory(t *testing.T) {
	host := func(body string) string {
		return `
resource "zabbix_hostgroup" "testcleargrp" {
	name = "test-clearable-host-group"
}
resource "zabbix_host" "testclear" {
	host   = "test-clearable-host"
	groups = [ zabbix_hostgroup.testcleargrp.id ]
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
			{ // two fields set
				Config: host(`
	inventory_mode = "manual"
	inventory {
		location = "test location"
		notes    = "test notes"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testclear", "inventory.0.location", "test location"),
					resource.TestCheckResourceAttr("zabbix_host.testclear", "inventory.0.notes", "test notes"),
				),
			},
			{ // one of them dropped from the block
				Config: host(`
	inventory_mode = "manual"
	inventory {
		location = "test location"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testclear", "inventory.0.location", "test location"),
					resource.TestCheckResourceAttr("zabbix_host.testclear", "inventory.0.notes", ""),
				),
			},
			{ // written out as an empty string rather than omitted -- the same
				// thing to Terraform, and it must be the same on the wire
				Config: host(`
	inventory_mode = "manual"
	inventory {
		location = ""
		notes    = "test notes"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testclear", "inventory.0.location", ""),
					resource.TestCheckResourceAttr("zabbix_host.testclear", "inventory.0.notes", "test notes"),
				),
			},
			{ // the whole block removed, inventory still enabled
				Config: host(`
	inventory_mode = "manual"
`),
				Check: resource.TestCheckResourceAttr("zabbix_host.testclear", "inventory.#", "0"),
			},
		},
	})
}

// TestAccClearableSnmpV3Credentials -- the eight snmp3_* attributes all carry
// a macro string as their default, so a user who wants a v3 interface with no
// authentication has to write the empty value out. HostInterfaceDetail tagged
// every one of them `omitempty`, and Zabbix merges a details object key by
// key: an omitted key keeps its stored value, verified against 7.4.
func TestAccClearableSnmpV3Credentials(t *testing.T) {
	host := func(body string) string {
		return `
resource "zabbix_hostgroup" "testclearsnmpgrp" {
	name = "test-clearable-snmp-group"
}
resource "zabbix_host" "testclear" {
	host   = "test-clearable-snmp-host"
	groups = [ zabbix_hostgroup.testclearsnmpgrp.id ]
	interface {
		type         = "snmp"
		ip           = "127.0.0.1"
		snmp_version = "3"
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
			{ // credentials set
				Config: host(`
		snmp3_securitylevel   = "authpriv"
		snmp3_securityname    = "clearuser"
		snmp3_authpassphrase  = "clearauth"
		snmp3_privpassphrase  = "clearpriv"
		snmp3_contextname     = "clearctx"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testclear", "interface.*", map[string]string{
						"snmp3_securityname":   "clearuser",
						"snmp3_authpassphrase": "clearauth",
						"snmp3_privpassphrase": "clearpriv",
						"snmp3_contextname":    "clearctx",
					}),
				),
			},
			{ // and cleared: noauthnopriv is a real configuration and it needs
				// every one of them empty
				Config: host(`
		snmp3_securitylevel   = "noauthnopriv"
		snmp3_securityname    = ""
		snmp3_authpassphrase  = ""
		snmp3_privpassphrase  = ""
		snmp3_contextname     = ""
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testclear", "interface.*", map[string]string{
						"snmp3_securitylevel":  "noauthnopriv",
						"snmp3_securityname":   "",
						"snmp3_authpassphrase": "",
						"snmp3_privpassphrase": "",
						"snmp3_contextname":    "",
					}),
				),
			},
		},
	})
}

// TestAccClearableTriggerRecoveryAndCorrelation -- Trigger.RecoveryExpression
// and Trigger.CorrelationTag are both `omitempty`, and both are reachable from
// the schema. They are safe, and it is worth recording why rather than
// re-deriving it: neither value can be empty while its mode is on, and Zabbix
// clears the field itself when the mode goes off. Verified on 6.0 through 8.0
// by this test, not by reading the documentation.
func TestAccClearableTriggerRecoveryAndCorrelation(t *testing.T) {
	trigger := func(body string) string {
		return hcl(t, clearableTemplateHCL+`
resource "zabbix_item_trapper" "testclear" {
	hostid    = zabbix_template.testcleartmpl.id
	key       = "test.clear.trigger.item"
	name      = "Test Clearable Trigger Item"
	valuetype = "unsigned"
}
resource "zabbix_trigger" "testclear" {
	name       = "Test Clearable Trigger"
	expression = "last(/test-clearable-template/test.clear.trigger.item)=1"

	depends_on = [ zabbix_item_trapper.testclear ]
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
				Config: trigger(`
	recovery_expression = "last(/test-clearable-template/test.clear.trigger.item)=0"
	correlation_mode    = "tag"
	correlation_tag     = "cleartag"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.testclear", "correlation_tag", "cleartag"),
				),
			},
			{ // both back off. "all" is Zabbix's own name for correlation
				// being off -- an OK event closes every problem the trigger
				// raised -- and it is the only way to turn tag correlation
				// back off, since correlation_mode has no "none".
				Config: trigger(`
	correlation_mode = "all"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.testclear", "recovery_expression", ""),
					resource.TestCheckResourceAttr("zabbix_trigger.testclear", "correlation_tag", ""),
				),
			},
		},
	})
}

// TestAccClearableItemUnitsAndDescription -- S9d for the two item display
// properties added in this release. Neither has a `Default:`, so the empty
// value is reachable by deleting the line as well as by writing `""`, and
// both journeys are here: set, write the empty string, and then delete the
// line entirely.
//
// The server re-read is the assertion that matters. Both attributes are
// written back into state by the provider's own read, so a clear that never
// left the provider still looks applied there -- which is exactly the shape
// of the six omitempty defects this release fixed. zabbix.Item.Units and
// .Description therefore carry no omitempty and the write path uses d.Get
// rather than d.GetOk.
//
// The fixture is `unsigned` because from Zabbix 7.0 a non-empty `units` is
// rejected outright on a character, log or text item; see the note on
// TestAccUpdateItemAgent.
func TestAccClearableItemUnitsAndDescription(t *testing.T) {
	const addr = "zabbix_item_trapper.testclear"

	item := func(body string) string {
		return hcl(t, clearableTemplateHCL+`
resource "zabbix_item_trapper" "testclear" {
	hostid    = zabbix_template.testcleartmpl.id
	key       = "test.clear.item.units"
	name      = "Test Clearable Item Units"
	valuetype = "unsigned"
`+body+`
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // both set, so the clear has something to undo
				Config: item(`
	units       = "B"
	description = "a description to clear"
`),
				Check: testAccCheckServerAttrs(addr, serverItem, map[string]string{
					"units":       "B",
					"description": "a description to clear",
				}),
			},
			{ // written out as empty strings
				Config: item(`
	units       = ""
	description = ""
`),
				Check: testAccCheckServerAttrs(addr, serverItem, map[string]string{
					"units":       "",
					"description": "",
				}),
			},
			{ // set again, so that deleting the line is a real change rather
				// than a no-op over the step above
				Config: item(`
	units       = "!B"
	description = "set once more"
`),
				Check: testAccCheckServerAttrs(addr, serverItem, map[string]string{
					"units":       "!B",
					"description": "set once more",
				}),
			},
			{ // and the lines deleted, which is the edit a user actually makes
				Config: item(``),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "units", ""),
					resource.TestCheckResourceAttr(addr, "description", ""),
					testAccCheckServerAttrs(addr, serverItem, map[string]string{
						"units":       "",
						"description": "",
					}),
				),
			},
		},
	})
}

// TestAccClearableLLDDescription -- the discovery-rule half. common_lld.go is
// a separate write path from common_item.go rather than a reuse of it, so
// nothing above says anything about a discovery rule; the LLD HTTP fields
// needed their own fix for precisely this reason.
func TestAccClearableLLDDescription(t *testing.T) {
	const addr = "zabbix_lld_trapper.testclear"

	rule := func(body string) string {
		return hcl(t, clearableTemplateHCL+`
resource "zabbix_lld_trapper" "testclear" {
	hostid = zabbix_template.testcleartmpl.id
	key    = "test.clear.lld.description"
	name   = "Test Clearable LLD Description"
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
				Config: rule(`	description = "a rule description to clear"`),
				Check: testAccCheckServerAttrs(addr, serverLLD, map[string]string{
					"description": "a rule description to clear",
				}),
			},
			{
				Config: rule(`	description = ""`),
				Check: testAccCheckServerAttrs(addr, serverLLD, map[string]string{
					"description": "",
				}),
			},
			{
				Config: rule(`	description = "set once more"`),
				Check: testAccCheckServerAttrs(addr, serverLLD, map[string]string{
					"description": "set once more",
				}),
			},
			{ // the line deleted
				Config: rule(``),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "description", ""),
					testAccCheckServerAttrs(addr, serverLLD, map[string]string{
						"description": "",
					}),
				),
			},
		},
	})
}

// TestAccClearableTagValue -- Tag.Value is `omitempty` and a tag value is
// optional, so this looks like the same bug. It is not: tags are a collection
// Zabbix replaces wholesale, so the tag object is rebuilt from scratch and an
// absent value means the default rather than the previous one. Asserted
// rather than assumed, because the reasoning is not obvious from the tag.
func TestAccClearableTagValue(t *testing.T) {
	host := func(body string) string {
		return `
resource "zabbix_hostgroup" "testcleartaggrp" {
	name = "test-clearable-tag-group"
}
resource "zabbix_host" "testclear" {
	host   = "test-clearable-tag-host"
	groups = [ zabbix_hostgroup.testcleartaggrp.id ]
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
	tag {
		key   = "cleartag"
		value = "clearvalue"
	}
`),
				Check: resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testclear", "tag.*", map[string]string{
					"key":   "cleartag",
					"value": "clearvalue",
				}),
			},
			{
				Config: host(`
	tag {
		key = "cleartag"
	}
`),
				Check: resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testclear", "tag.*", map[string]string{
					"key":   "cleartag",
					"value": "",
				}),
			},
		},
	})
}
