package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// E6 -- scalar boundaries (PLAN.md Phase 8).
//
// Three things, all of which a happy-path fixture steps straight over:
//
//   - **Emptying an optional string.** This is the scalar twin of C6, and it
//     has exactly the same cause. Most string fields on the API client's
//     structs carry `omitempty`, so a value set back to "" is not sent at
//     all, the server keeps what it had, and the read that follows writes the
//     old value back into state. The result is a diff that reapplies forever
//     and never converges. Nothing has to assert it explicitly: the test
//     harness requires the plan after a step to be empty, so a clear that
//     does not reach the server fails the step by itself.
//
//   - **Unicode.** Names and descriptions go out as JSON and come back
//     through the database's collation. Anything that mangles them -- a
//     length check counting bytes, a round trip through a lossy encoding --
//     shows up as a value that differs from the one written, which again is
//     a plan that will not settle.
//
//   - **Values Zabbix treats specially.** Macro syntax inside a macro value,
//     a context macro name, an empty tag value, and the characters that have
//     to survive JSON escaping.
//
// Every test here ends with a PlanOnly step. That is the assertion: the
// configuration must be reachable *and stable*, because a value that writes
// but does not read back identically is the failure mode all three share.

const boundaryTemplateHCL = `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-boundary-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host   = "test-boundary-template"
}
`

// TestAccBoundaryHttpItemEmptyStrings covers the four optional strings on an
// HTTP item that have no default and are `omitempty` on the wire: posts,
// username, password and proxy. Setting them and then removing them again is
// the whole test -- if the removal is dropped, the step's own plan is not
// empty and the harness fails it.
func TestAccBoundaryHttpItemEmptyStrings(t *testing.T) {
	item := func(extra string) string {
		return hcl(t, boundaryTemplateHCL+`
resource "zabbix_item_http" "testitem" {
	hostid    = zabbix_template.testtmpl.id
	key       = "boundary.http"
	name      = "Boundary Http"
	valuetype = "text"
	url       = "http://localhost/boundary"
`+extra+`
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // omitted entirely: the baseline every attribute must return to
				Config: item(``),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "posts", ""),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "username", ""),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "password", ""),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "proxy", ""),
				),
			},
			{ // all four set
				Config: item(`
	posts     = "{\"probe\": true}"
	post_type = "json"
	username  = "boundary-user"
	password  = "boundary-pass"
	proxy     = "http://proxy.example.com:3128"
	auth_type = "basic"
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "posts", `{"probe": true}`),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "username", "boundary-user"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "password", "boundary-pass"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "proxy", "http://proxy.example.com:3128"),
				),
			},
			{ // written as empty rather than omitted -- the two are the same
				// to Terraform but only one of them is what a user types
				Config: item(`
	posts     = ""
	username  = ""
	password  = ""
	proxy     = ""
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "posts", ""),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "username", ""),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "password", ""),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "proxy", ""),
				),
			},
			{ // and the cleared state is stable
				Config:   item(``),
				PlanOnly: true,
			},
		},
	})
}

// TestAccBoundaryTriggerEmptyStrings does the same for a trigger's optional
// prose and link attributes. url, recovery_expression and correlation_tag are
// all `omitempty` on the Trigger struct, so all three are candidates for a
// clear that never reaches the server.
func TestAccBoundaryTriggerEmptyStrings(t *testing.T) {
	trigger := func(extra string) string {
		return hcl(t, boundaryTemplateHCL+`
resource "zabbix_item_trapper" "testitem" {
	hostid    = zabbix_template.testtmpl.id
	key       = "boundary.trapper"
	name      = "Boundary Trapper"
	valuetype = "unsigned"
}
resource "zabbix_trigger" "testtrigger" {
	name       = "Boundary Trigger"
	expression = "last(/test-boundary-template/boundary.trapper)>10"
`+extra+`

	depends_on = [zabbix_item_trapper.testitem]
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: trigger(``),
			},
			{ // everything optional filled in
				Config: trigger(`
	comments            = "boundary comment"
	event_name          = "Boundary Event"
	opdata              = "value is {ITEM.LASTVALUE}"
	url                 = "http://example.com/runbook"
	recovery_expression = "last(/test-boundary-template/boundary.trapper)<5"
	correlation_mode    = "tag"
	correlation_tag     = "boundary"
	manual_close        = true
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "url", "http://example.com/runbook"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "correlation_tag", "boundary"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "recovery_expression",
						"last(/test-boundary-template/boundary.trapper)<5"),
				),
			},
			{ // all of it emptied again in one update
				Config: trigger(`
	comments            = ""
	event_name          = ""
	opdata              = ""
	url                 = ""
	recovery_expression = ""
	correlation_mode    = "all"
	correlation_tag     = ""
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "comments", ""),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "event_name", ""),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "opdata", ""),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "url", ""),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "recovery_expression", ""),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "correlation_tag", ""),
				),
			},
			{
				Config: trigger(`
	comments            = ""
	event_name          = ""
	opdata              = ""
	url                 = ""
	recovery_expression = ""
	correlation_mode    = "all"
	correlation_tag     = ""
`),
				PlanOnly: true,
			},
		},
	})
}

// TestAccBoundaryHostEmptyStrings covers the host attributes that are
// deliberately *not* omitempty on the Host struct -- IPMIUsername,
// IPMIPassword, TLSIssuer, TLSSubject -- so that clearing them is expressible
// at all. This is the test that keeps that decision from being undone.
func TestAccBoundaryHostEmptyStrings(t *testing.T) {
	host := func(extra string) string {
		return `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-boundary-group"
}
resource "zabbix_host" "testhost" {
	host   = "test-boundary-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
` + extra + `
}
`
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: host(``),
			},
			{
				Config: host(`
	ipmi_username = "boundary-ipmi"
	ipmi_password = "boundary-ipmi-pass"
	tls_issuer    = "CN=Boundary CA"
	tls_subject   = "CN=test-boundary-host"
	tls_connect   = "cert"
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "ipmi_username", "boundary-ipmi"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tls_issuer", "CN=Boundary CA"),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tls_subject", "CN=test-boundary-host"),
				),
			},
			{ // cleared -- tls_connect has to come back with them, since
				// Zabbix will not keep certificate mode without a subject
				Config: host(`
	ipmi_username = ""
	ipmi_password = ""
	tls_issuer    = ""
	tls_subject   = ""
	tls_connect   = "unencrypted"
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "ipmi_username", ""),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "ipmi_password", ""),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tls_issuer", ""),
					resource.TestCheckResourceAttr("zabbix_host.testhost", "tls_subject", ""),
				),
			},
			{
				Config:   host(``),
				PlanOnly: true,
			},
		},
	})
}

// unicodeName exercises three separate hazards in one string: characters
// outside the BMP (an emoji, which is a surrogate pair in UTF-16 and four
// bytes in UTF-8), a combining sequence, and a right-to-left script. Any
// length limit counting bytes rather than characters, or any encoding that is
// not UTF-8 end to end, produces something different from what went in.
const unicodeName = "Тест — 日本語 \U0001F680 café مرحبا"

// TestAccBoundaryUnicode writes non-ASCII into every visible name and
// description the provider has, on objects of four different kinds, and
// requires all of them to come back byte for byte.
func TestAccBoundaryUnicode(t *testing.T) {
	config := hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-boundary-`+unicodeName+`"
}
resource "zabbix_template" "testtmpl" {
	groups      = [ zabbix_templategroup.testtmplgrp.id ]
	host        = "test-boundary-template"
	name        = "`+unicodeName+`"
	description = "`+unicodeName+`"
}
resource "zabbix_item_trapper" "testitem" {
	hostid    = zabbix_template.testtmpl.id
	key       = "boundary.unicode"
	name      = "`+unicodeName+`"
	valuetype = "unsigned"

	tag {
		key   = "unicode"
		value = "`+unicodeName+`"
	}
}
resource "zabbix_trigger" "testtrigger" {
	name       = "`+unicodeName+`"
	expression = "last(/test-boundary-template/boundary.unicode)>10"
	comments   = "`+unicodeName+`"

	depends_on = [zabbix_item_trapper.testitem]
}
`)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tmplGroupAddr(t, "testtmplgrp"), "name", "test-boundary-"+unicodeName),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "name", unicodeName),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "description", unicodeName),
					resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "name", unicodeName),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_item_trapper.testitem", "tag.*", map[string]string{
						"key":   "unicode",
						"value": unicodeName,
					}),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "name", unicodeName),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "comments", unicodeName),
				),
			},
			{ // C7-style: the values have to survive the import path too,
				// which reads them back with no config to compare against
				ResourceName:      "zabbix_template.testtmpl",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// TestAccBoundaryMacroValues covers the values Zabbix gives meaning to.
//
// A user macro's value is stored verbatim and expanded by reference from
// somewhere else, so a value that itself looks like a macro reference, or
// that contains the braces and quotes the macro grammar uses, is the case
// most likely to be mangled by something trying to be clever. The macro
// *name* has its own grammar: a context macro carries a quoted context after
// a colon, and a regex context is introduced by "regex:".
func TestAccBoundaryMacroValues(t *testing.T) {
	config := hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-boundary-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host   = "test-boundary-template"

	macro {
		name  = "{$PLAIN}"
		value = "plain"
	}
	macro {
		name  = "{$LOOKS_LIKE_A_MACRO}"
		value = "{$SOMETHING_ELSE}"
	}
	macro {
		name  = "{$BRACES}"
		value = "a{b}c$d"
	}
	macro {
		name  = "{$QUOTES}"
		value = "he said \"hi\" and \\ then left"
	}
	macro {
		name  = "{$SPACES}"
		value = "  leading and trailing  "
	}
	macro {
		name  = "{$CONTEXT:\"/boot\"}"
		value = "context value"
	}
	macro {
		name  = "{$REGEX:regex:\"^/tmp$\"}"
		value = "regex context value"
	}
	macro {
		name  = "{$UNICODE}"
		value = "`+unicodeName+`"
	}
}
`)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "macro.#", "8"),
					testAccCheckTemplateMacroCount("zabbix_template.testtmpl", 8),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_template.testtmpl", "macro.*", map[string]string{
						"name": "{$LOOKS_LIKE_A_MACRO}", "value": "{$SOMETHING_ELSE}",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_template.testtmpl", "macro.*", map[string]string{
						"name": "{$BRACES}", "value": "a{b}c$d",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_template.testtmpl", "macro.*", map[string]string{
						"name": "{$QUOTES}", "value": `he said "hi" and \ then left`,
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_template.testtmpl", "macro.*", map[string]string{
						"name": "{$SPACES}", "value": "  leading and trailing  ",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_template.testtmpl", "macro.*", map[string]string{
						"name": `{$CONTEXT:"/boot"}`, "value": "context value",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_template.testtmpl", "macro.*", map[string]string{
						"name": `{$REGEX:regex:"^/tmp$"}`, "value": "regex context value",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_template.testtmpl", "macro.*", map[string]string{
						"name": "{$UNICODE}", "value": unicodeName,
					}),
				),
			},
			{ // the set hash and the flatten function have to agree on all
				// eight, which only an import at full size checks
				ResourceName:      "zabbix_template.testtmpl",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// TestAccBoundaryTagValues is the tag equivalent. The empty value is the case
// that matters most: tags with no value are ordinary in Zabbix and are how
// "just a label" is expressed, but "value" is the one attribute of the tag
// element that is Optional, so an empty one has to survive both the set hash
// -- which concatenates key and value -- and the wire.
func TestAccBoundaryTagValues(t *testing.T) {
	config := hcl(t, boundaryTemplateHCL+`
resource "zabbix_item_trapper" "testitem" {
	hostid    = zabbix_template.testtmpl.id
	key       = "boundary.tags"
	name      = "Boundary Tags"
	valuetype = "unsigned"

	tag {
		key = "novalue"
	}
	tag {
		key   = "empty"
		value = ""
	}
	tag {
		key   = "special"
		value = "a:b/c d,e\"f\\g"
	}
	tag {
		key   = "macroish"
		value = "{$NOT_EXPANDED}"
	}
	tag {
		key   = "unicode"
		value = "`+unicodeName+`"
	}
}
`)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					// "novalue" and "empty" differ only in that one omits
					// the value and the other writes it as "": distinct keys,
					// so Zabbix stores both, and an empty value is not the
					// same thing as an absent tag
					resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "tag.#", "5"),
					testAccCheckItemTagCount("zabbix_item_trapper.testitem", 5),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_item_trapper.testitem", "tag.*", map[string]string{
						"key": "empty", "value": "",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_item_trapper.testitem", "tag.*", map[string]string{
						"key": "novalue", "value": "",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_item_trapper.testitem", "tag.*", map[string]string{
						"key": "special", "value": `a:b/c d,e"f\g`,
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_item_trapper.testitem", "tag.*", map[string]string{
						"key": "macroish", "value": "{$NOT_EXPANDED}",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_item_trapper.testitem", "tag.*", map[string]string{
						"key": "unicode", "value": unicodeName,
					}),
				),
			},
			{
				ResourceName:      "zabbix_item_trapper.testitem",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}
