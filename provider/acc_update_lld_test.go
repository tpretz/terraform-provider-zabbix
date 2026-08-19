package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// U1/U2 for the discovery-rule family (PLAN.md § "The unit of work").
//
// common_lld.go is a separate implementation from common_item.go rather than a
// reuse of it -- separate schema fragments, separate mod and read funcs,
// separate preprocessing table -- so nothing in acc_update_item_test.go says
// anything about a discovery rule. This covers lldCommonSchema,
// lldDelaySchema, lldInterfaceSchema, lldPreprocessorSchema,
// lldFilterConditionSchema and lldMacroPathSchema in one fixture.
//
// The fixture is a host rather than a template because lldInterfaceSchema's
// interfaceid is only meaningful on one, and because an agent discovery rule
// on a host must name an interface -- the same probe result the item tests
// record.
//
// "active" is deliberately left alone here and covered by
// TestAccUpdateItemAgent instead, which is the same declaration. Zabbix rejects
// an active agent rule that names an interface -- `Invalid parameter
// "/1/interfaceid": value must be 0` on 7.4 -- so the two attributes cannot be
// changed in the same step, and interfaceid is the one only this fixture can
// reach.

// TestAccUpdateLLDAgent changes every attribute a discovery rule has, in one
// step, on a rule that already exists.
func TestAccUpdateLLDAgent(t *testing.T) {
	const addr = "zabbix_lld_agent.testlld"

	lld := func(port, body string) string {
		return `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-update-lld-group"
}
resource "zabbix_host" "testhost" {
	host   = "test-update-lld-host"
	groups = [ zabbix_hostgroup.testgrp.id ]
	interface {
		ip   = "127.0.0.1"
		port = 10050
		main = true
	}
	interface {
		ip   = "127.0.0.2"
		port = 10051
		main = false
	}
}
resource "zabbix_lld_agent" "testlld" {
	hostid      = zabbix_host.testhost.id
	interfaceid = one([ for i in zabbix_host.testhost.interface : i.id if i.port == ` + port + ` ])
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
				Config: lld("10050", `
	key         = "test.update.lld.a"
	name        = "Update LLD A"
	delay       = "1h"
	lifetime    = "30d"
	evaltype    = "andor"
	description = "Update LLD description A"
	condition {
		macro    = "{#AAA}"
		value    = "one"
		operator = "match"
	}
	macro {
		macro = "{#AAA}"
		path  = "$.a"
	}
	preprocessor {
		type   = "xml_xpath"
		params = [ "/root/a" ]
	}
`),
				Check: testAccCheckServerAttrs(addr, serverLLD, map[string]string{
					"key_":            "test.update.lld.a",
					"name":            "Update LLD A",
					"delay":           "1h",
					"lifetime":        "30d",
					"type":            "0", // passive agent
					"filter.evaltype": "0",
					"description":     "Update LLD description A",
				}),
			},
			{ // every one of them changed, in life
				Config: lld("10051", `
	key         = "test.update.lld.b"
	name        = "Update LLD B"
	delay       = "2h"
	lifetime    = "45d"
	evaltype    = "custom"
	formula     = "A and B"
	description = "Update LLD description B"
	condition {
		id       = "A"
		macro    = "{#BBB}"
		value    = "two"
		operator = "notmatch"
	}
	condition {
		id    = "B"
		macro = "{#CCC}"
		value = "three"
	}
	macro {
		macro = "{#BBB}"
		path  = "$.b"
	}
	preprocessor {
		type                 = "jsonpath"
		params               = [ "$.b" ]
		error_handler        = "3"
		error_handler_params = "boom"
	}
`),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerAttrs(addr, serverLLD, map[string]string{
						"key_":                "test.update.lld.b",
						"name":                "Update LLD B",
						"delay":               "2h",
						"lifetime":            "45d",
						"type":                "0", // passive agent: see the note above
						"filter.evaltype":     "3",
						"filter.eval_formula": "A and B",
						"description":         "Update LLD description B",
					}),
					testAccCheckServerElem(addr, serverLLD, "filter.conditions", "macro", "{#BBB}", map[string]string{
						"value":     "two",
						"operator":  "9",
						"formulaid": "A",
					}),
					testAccCheckServerElem(addr, serverLLD, "filter.conditions", "macro", "{#CCC}", map[string]string{
						"value":     "three",
						"operator":  "8",
						"formulaid": "B",
					}),
					testAccCheckServerElem(addr, serverLLD, "lld_macro_paths", "lld_macro", "{#BBB}", map[string]string{
						"path": "$.b",
					}),
					testAccCheckServerElem(addr, serverLLD, "preprocessing", "type", "12", map[string]string{
						"params":               "$.b",
						"error_handler":        "3",
						"error_handler_params": "boom",
					}),
					testAccCheckLLDInterfacePort(addr, "10051"),
				),
			},
		},
	})
}

// testAccCheckLLDInterfacePort is testAccCheckItemInterfacePort for a
// discovery rule: the interfaceid the *server* has, resolved to the interface
// it names, so the assertion is about which interface the rule polls through
// rather than about an id the test would have had to learn from the provider.
func testAccCheckLLDInterfacePort(addr, wantPort string) resource.TestCheckFunc {
	return testAccCheckServerAttrs(addr, serverInterfaceOf(serverLLD), map[string]string{"port": wantPort})
}
