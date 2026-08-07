package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// E2 -- negative paths (PLAN.md Phase 8).
//
// Before this file the whole suite carried four ExpectError assertions, all
// on zabbix_proxy and zabbix_templategroup. Everything else was tested only
// with input it was expected to accept, so a validator that had been dropped,
// or one that had never been wired up, looked exactly like one that worked.
//
// Three kinds of rejection are covered here, and they fail in three different
// places:
//
//   - schema validation, before any API call -- the enum cases below and the
//     pinned LLD delay. schema_enum_test.go asserts the same validators
//     attribute by attribute without a server; these steps prove the
//     diagnostic actually reaches the user through a real terraform run, and
//     that the attribute is reachable in HCL under the name the validator is
//     attached to.
//   - provider logic, in the mod func -- the proxy operating-mode mismatches.
//   - the server, which is the only thing that can judge an LLD filter
//     formula.
//
// Every step here fails during plan or apply and creates nothing, so they can
// share fixture names freely and leave no state behind.
//
// Two E2 cases already had coverage and are not repeated:
// TestAccResourceProxyModeMismatch (address and allowed_addresses on the
// wrong mode) and TestAccResourceTemplategroupUnsupported (zabbix_templategroup
// below 6.2).

// enumRejected matches the message validation.StringInSlice produces. Each
// step below puts exactly one invalid enum value in an otherwise valid
// configuration, so a match can only have come from the attribute under test.
var enumRejected = regexp.MustCompile("to be one of")

const negTemplateHCL = `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-negative-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host   = "test-negative-template"
}
`

const negGroupHCL = `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-negative-group"
}
`

// negHostHCL puts extra attributes on a host, negHostIfaceHCL inside its
// interface block.
func negHostHCL(extra string) string {
	return negGroupHCL + `
resource "zabbix_host" "testhost" {
	host   = "test-negative-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
` + extra + `
}
`
}

func negHostIfaceHCL(extra string) string {
	return negGroupHCL + `
resource "zabbix_host" "testhost" {
	host   = "test-negative-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		ip = "127.0.0.1"
` + extra + `
	}
}
`
}

// negGraphHCL puts extra attributes on a graph, negGraphItemHCL inside one of
// its item blocks.
func negGraphHCL(extra, itemExtra string) string {
	return negTemplateHCL + `
resource "zabbix_item_agent" "testitem" {
	hostid    = zabbix_template.testtmpl.id
	key       = "negative.item"
	name      = "Negative Item"
	valuetype = "unsigned"
}
resource "zabbix_graph" "testgraph" {
	name   = "test-negative-graph"
	width  = "600"
	height = "400"
` + extra + `

	item {
		color  = "FFFF00"
		itemid = zabbix_item_agent.testitem.id
` + itemExtra + `
	}
}
`
}

func negTriggerHCL(extra string) string {
	return negTemplateHCL + `
resource "zabbix_item_trapper" "testitem" {
	hostid    = zabbix_template.testtmpl.id
	key       = "negative.trapper"
	name      = "Negative Trapper"
	valuetype = "unsigned"
}
resource "zabbix_trigger" "testtrigger" {
	name       = "test-negative-trigger"
	expression = "last(/test-negative-template/negative.trapper)>10"
` + extra + `

	depends_on = [zabbix_item_trapper.testitem]
}
`
}

func negLLDHCL(extra string) string {
	return negTemplateHCL + `
resource "zabbix_lld_agent" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key    = "negative.lld"
	name   = "Negative LLD"
` + extra + `
}
`
}

// TestAccNegativeEnums walks the enum-validated attributes of every resource
// that has one and requires each to reject a value outside its list. They are
// steps of a single test rather than separate tests because none of them
// creates anything: validation fails before the graph is walked, so the steps
// are independent and share one terraform working directory.
func TestAccNegativeEnums(t *testing.T) {
	cases := []struct{ what, config string }{
		// itemCommonSchema, shared by all twenty item and prototype resources
		{"item valuetype", negTemplateHCL + `
resource "zabbix_item_agent" "testitem" {
	hostid    = zabbix_template.testtmpl.id
	key       = "negative.item"
	name      = "Negative Item"
	valuetype = "flooat"
}
`},
		// resource_http_common.go
		{"http request_method", negTemplateHCL + `
resource "zabbix_item_http" "testitem" {
	hostid         = zabbix_template.testtmpl.id
	key            = "negative.http"
	name           = "Negative Http"
	valuetype      = "text"
	url            = "http://localhost/negative"
	request_method = "fetch"
}
`},
		{"http post_type", negTemplateHCL + `
resource "zabbix_item_http" "testitem" {
	hostid    = zabbix_template.testtmpl.id
	key       = "negative.http"
	name      = "Negative Http"
	valuetype = "text"
	url       = "http://localhost/negative"
	post_type = "yaml"
}
`},
		{"http retrieve_mode", negTemplateHCL + `
resource "zabbix_item_http" "testitem" {
	hostid        = zabbix_template.testtmpl.id
	key           = "negative.http"
	name          = "Negative Http"
	valuetype     = "text"
	url           = "http://localhost/negative"
	retrieve_mode = "trailers"
}
`},
		{"http auth_type", negTemplateHCL + `
resource "zabbix_item_http" "testitem" {
	hostid    = zabbix_template.testtmpl.id
	key       = "negative.http"
	name      = "Negative Http"
	valuetype = "text"
	url       = "http://localhost/negative"
	auth_type = "oauth"
}
`},

		// common_lld.go
		{"lld evaltype", negLLDHCL(`	evaltype = "xor"`)},
		{"lld condition operator", negLLDHCL(`
	condition {
		macro    = "{#FSNAME}"
		value    = "^/$"
		operator = "startswith"
	}
`)},

		// resource_host.go
		{"host inventory_mode", negHostHCL(`	inventory_mode = "semiautomatic"`)},
		{"host ipmi_authtype", negHostHCL(`	ipmi_authtype = "md4"`)},
		{"host ipmi_privilege", negHostHCL(`	ipmi_privilege = "root"`)},
		{"host tls_connect", negHostHCL(`	tls_connect = "tls"`)},
		{"host tls_accept", negHostHCL(`	tls_accept = "tls"`)},
		{"host interface type", negHostIfaceHCL(`		type = "wmi"`)},
		{"host interface snmp_version", negHostIfaceHCL(`
		type         = "snmp"
		snmp_version = "4"
`)},
		{"host interface snmp3_authprotocol", negHostIfaceHCL(`
		type               = "snmp"
		snmp_version       = "3"
		snmp3_authprotocol = "sha256"
`)},
		{"host interface snmp3_privprotocol", negHostIfaceHCL(`
		type               = "snmp"
		snmp_version       = "3"
		snmp3_privprotocol = "3des"
`)},
		{"host interface snmp3_securitylevel", negHostIfaceHCL(`
		type                = "snmp"
		snmp_version        = "3"
		snmp3_securitylevel = "authonly"
`)},

		// resource_graph.go -- also covers zabbix_proto_graph, which is built
		// from the same schemaGraph
		{"graph type", negGraphHCL(`	type = "candlestick"`, ``)},
		{"graph ymin_type", negGraphHCL(`	ymin_type = "auto"`, ``)},
		{"graph ymax_type", negGraphHCL(`	ymax_type = "auto"`, ``)},
		{"graph item function", negGraphHCL(``, `		function = "median"`)},
		{"graph item drawtype", negGraphHCL(``, `		drawtype = "dotted"`)},
		{"graph item type", negGraphHCL(``, `		type = "stacked"`)},
		{"graph item yaxis_side", negGraphHCL(``, `		yaxis_side = "top"`)},

		// resource_trigger.go -- also covers zabbix_proto_trigger
		{"trigger priority", negTriggerHCL(`	priority = "critical"`)},
		{"trigger correlation_mode", negTriggerHCL(`
	correlation_mode = "event"
	correlation_tag  = "svc"
`)},

		// resource_proxy.go
		{"proxy operating_mode", `
resource "zabbix_proxy" "testproxy" {
	name           = "test-negative-proxy"
	operating_mode = "bidirectional"
}
`},
		{"proxy tls_connect", `
resource "zabbix_proxy" "testproxy" {
	name        = "test-negative-proxy"
	tls_connect = "tls"
}
`},
		{"proxy tls_accept", `
resource "zabbix_proxy" "testproxy" {
	name       = "test-negative-proxy"
	tls_accept = "tls"
}
`},
	}

	steps := make([]resource.TestStep, 0, len(cases)+1)
	for _, c := range cases {
		steps = append(steps, resource.TestStep{
			Config:      hcl(t, c.config),
			ExpectError: enumRejected,
		})
	}

	// The post-test destroy re-validates the configuration left by the final
	// step, so a test whose last config is deliberately invalid cannot be torn
	// down: it fails with the very error it was asserting. End on something
	// valid. It also doubles as proof that the twenty-nine steps above created
	// nothing -- this group is the only object the destroy has to remove, and
	// CheckDestroy would report anything else that had been left behind.
	steps = append(steps, resource.TestStep{
		Config: negGroupHCL,
		Check:  resource.TestCheckResourceAttr("zabbix_hostgroup.testgrp", "name", "test-negative-group"),
	})

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps:             steps,
	})
}

// TestAccNegativeLLDZeroDelay covers lldZeroDelaySchema. Trapper and dependent
// discovery rules are not polled and Zabbix requires delay == 0 for both. The
// provider pins the value in the schema rather than passing anything through
// and letting the server object, so the rejection has to happen at plan time
// -- and if the pin were ever replaced by the shared lldDelaySchema, the
// resource would go back to defaulting to 3600 and failing on create with a
// message about a field the user never wrote.
func TestAccNegativeLLDZeroDelay(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: hcl(t, negTemplateHCL+`
resource "zabbix_lld_trapper" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key    = "negative.lld.trapper"
	name   = "Negative LLD Trapper"
	delay  = "5m"
}
`),
				ExpectError: enumRejected,
			},
			{
				Config: hcl(t, negTemplateHCL+`
resource "zabbix_item_trapper" "testmaster" {
	hostid    = zabbix_template.testtmpl.id
	key       = "negative.master"
	name      = "Negative Master"
	valuetype = "text"
}
resource "zabbix_lld_dependent" "testlld" {
	hostid        = zabbix_template.testtmpl.id
	key           = "negative.lld.dependent"
	name          = "Negative LLD Dependent"
	master_itemid = zabbix_item_trapper.testmaster.id
	delay         = "1h"
}
`),
				ExpectError: enumRejected,
			},
			{ // "0" is the one value it takes, and it is also the default, so
				// prove the pin is a pin and not a blanket refusal
				Config: hcl(t, negTemplateHCL+`
resource "zabbix_lld_trapper" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key    = "negative.lld.trapper"
	name   = "Negative LLD Trapper"
	delay  = "0"
}
`),
				Check: resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "delay", "0"),
			},
		},
	})
}

// TestAccNegativeLLDCustomFormula covers the one filter case the provider
// cannot judge for itself. Under evaltype "custom" the formula and the
// condition ids come from the configuration -- everywhere else Zabbix assigns
// the ids -- so a formula that does not parse, or that names an id no
// condition carries, can only be rejected by the server. What matters is that
// it *is* rejected rather than silently reduced to some other evaluation
// order, which is what the provider echoing ids back at a server that had
// renumbered them used to look like.
//
// The wording of the rejection is not the same on every server, so each
// pattern below is an alternation of what the matrix actually returns rather
// than one message assumed to be universal. Probed directly:
//
//	6.0.48        Incorrect custom expression "A and (B" for discovery rule
//	              "...": check expression starting from "B".
//	              Condition "Z" used in formula "A and Z" ... is not defined.
//	              Incorrect custom expression "" ...: expression is empty.
//	7.0 / 7.4 / 8.0
//	              Invalid parameter "/1/filter/formula": incorrect syntax near "B".
//	              Invalid parameter "/1/filter/formula": missing filter condition "Z".
//	              Invalid parameter "/1/filter/formula": cannot be empty.
//
// The last case diverges further: a condition with no id is "the parameter
// \"formulaid\" is missing" from 7.0 but, on 6.0, the same
// condition-is-not-defined error as the second case. An alternation rather
// than a version gate because the matrix jumps 6.0 -> 7.0 and the exact
// release the wording changed in was not established.
func TestAccNegativeLLDCustomFormula(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // unbalanced parenthesis
				Config: hcl(t, lldTrapperConditionHCL(`
	evaltype = "custom"
	formula = "A and (B"

	condition {
		id = "A"
		macro = "{#AAA}"
		value = "one"
	}
	condition {
		id = "B"
		macro = "{#BBB}"
		value = "two"
	}
`)),
				ExpectError: regexp.MustCompile(`incorrect syntax near|Incorrect custom expression`),
			},
			{ // names an id no condition carries
				Config: hcl(t, lldTrapperConditionHCL(`
	evaltype = "custom"
	formula = "A and Z"

	condition {
		id = "A"
		macro = "{#AAA}"
		value = "one"
	}
`)),
				ExpectError: regexp.MustCompile(`missing filter condition "Z"|Condition "Z" used in formula`),
			},
			{ // custom evaltype with no formula at all
				Config: hcl(t, lldTrapperConditionHCL(`
	evaltype = "custom"

	condition {
		id = "A"
		macro = "{#AAA}"
		value = "one"
	}
`)),
				ExpectError: regexp.MustCompile(`cannot be empty|expression is empty`),
			},
			{ // a condition with no id, which custom evaltype requires
				Config: hcl(t, lldTrapperConditionHCL(`
	evaltype = "custom"
	formula = "A"

	condition {
		macro = "{#AAA}"
		value = "one"
	}
`)),
				ExpectError: regexp.MustCompile(`"formulaid" is missing|Condition "A" used in formula`),
			},
		},
	})
}

// TestAccNegativeProxyActivePort completes the operating-mode mismatch set
// begun in TestAccResourceProxyModeMismatch, which covers address on an
// active proxy and allowed_addresses on a passive one. port is the third
// endpoint attribute and takes the same path, but through a different entry
// in the check loop -- and unlike address it has a non-empty default, so a
// mistake in the "value != default" test would show up here and nowhere else.
func TestAccNegativeProxyActivePort(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: `
resource "zabbix_proxy" "testproxy" {
	name = "test-negative-proxy"
	port = "10999"
}
`,
				ExpectError: regexp.MustCompile("port applies to passive proxies only"),
			},
			{ // the default is not a mismatch: writing it explicitly on an
				// active proxy must still be accepted
				Config: `
resource "zabbix_proxy" "testproxy" {
	name    = "test-negative-proxy"
	address = "127.0.0.1"
	port    = "10051"
}
`,
				Check: resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "operating_mode", "active"),
			},
		},
	})
}
