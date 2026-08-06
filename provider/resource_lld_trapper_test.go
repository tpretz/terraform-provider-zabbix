package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceLLDTrapper(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // lazy init
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
`),
			},
			{ // simple create
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_lld_trapper" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.trapper[\"abc\"]"

	name = "LLD Trapper Rule"
	delay = "0"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "key", "lld.trapper[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "name", "LLD Trapper Rule"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "delay", "0"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "lifetime", "30d"),
				),
			},
			{ // modify: rename, change key, lifetime, add a filter condition (delay stays 0: required for trapper-type discovery rules)
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_lld_trapper" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.trapper[\"def\"]"

	name = "LLD Trapper Rule Renamed"
	delay = "0"
	lifetime = "7d"

	condition {
		macro = "{#FOO}"
		value = "bar"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "key", "lld.trapper[\"def\"]"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "name", "LLD Trapper Rule Renamed"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "delay", "0"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "lifetime", "7d"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_lld_trapper.testlld", "condition.*", map[string]string{
						"macro": "{#FOO}",
						"value": "bar",
					}),
				),
			},
			{ // several conditions at once, deliberately not in the order
				// Zabbix will hand them back. `condition` is a set: 6.0
				// returns conditions in submission order, 7.2+ returns them
				// sorted by the formula id it assigned, and neither is
				// something the config can or should pin down.
				//
				// This step also updates a rule that already has a condition
				// in state, which is the case that used to fail on 7.2+: the
				// provider echoed back the server-assigned formula ids and the
				// server rejected them with `value must be empty`.
				Config: hcl(t, lldTrapperConditionHCL(`
	condition {
		macro = "{#CCC}"
		value = "three"
	}
	condition {
		macro = "{#AAA}"
		value = "one"
	}
	condition {
		macro = "{#BBB}"
		value = "two"
		operator = "notmatch"
	}
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "condition.#", "3"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "evaltype", "andor"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_lld_trapper.testlld", "condition.*", map[string]string{
						"macro":    "{#AAA}",
						"value":    "one",
						"operator": "match",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_lld_trapper.testlld", "condition.*", map[string]string{
						"macro":    "{#BBB}",
						"value":    "two",
						"operator": "notmatch",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_lld_trapper.testlld", "condition.*", map[string]string{
						"macro":    "{#CCC}",
						"value":    "three",
						"operator": "match",
					}),
				),
			},
			{ // the same three conditions, written in yet another order: must
				// plan clean
				Config: hcl(t, lldTrapperConditionHCL(`
	condition {
		macro = "{#BBB}"
		value = "two"
		operator = "notmatch"
	}
	condition {
		macro = "{#CCC}"
		value = "three"
	}
	condition {
		macro = "{#AAA}"
		value = "one"
	}
`)),
				PlanOnly: true,
			},
			{ // drop one condition and edit another
				Config: hcl(t, lldTrapperConditionHCL(`
	condition {
		macro = "{#AAA}"
		value = "one-changed"
	}
	condition {
		macro = "{#CCC}"
		value = "three"
	}
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "condition.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_lld_trapper.testlld", "condition.*", map[string]string{
						"macro": "{#AAA}",
						"value": "one-changed",
					}),
				),
			},
			{ // evaltype "custom": here the formula ids are the user's to
				// choose, and the provider has to send them. Note that Zabbix
				// renumbers the ids into the order they first appear in the
				// formula, so a formula written out of order comes back
				// rewritten -- keep it canonical.
				Config: hcl(t, lldTrapperConditionHCL(`
	evaltype = "custom"
	formula = "A or (B and C)"

	condition {
		id = "C"
		macro = "{#CCC}"
		value = "three"
	}
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
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "evaltype", "custom"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "formula", "A or (B and C)"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "condition.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_lld_trapper.testlld", "condition.*", map[string]string{
						"id":    "A",
						"macro": "{#AAA}",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_lld_trapper.testlld", "condition.*", map[string]string{
						"id":    "B",
						"macro": "{#BBB}",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_lld_trapper.testlld", "condition.*", map[string]string{
						"id":    "C",
						"macro": "{#CCC}",
					}),
				),
			},
			{ // import
				ResourceName:      "zabbix_lld_trapper.testlld",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// lldTrapperConditionHCL wraps a block of filter conditions in the rest of the
// fixture, so the condition steps differ only by the thing under test.
func lldTrapperConditionHCL(conditions string) string {
	return `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_lld_trapper" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.trapper[\"def\"]"

	name = "LLD Trapper Rule Renamed"
	delay = "0"
	lifetime = "7d"
` + conditions + `}
`
}
