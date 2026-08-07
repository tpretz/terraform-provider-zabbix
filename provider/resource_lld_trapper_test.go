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
			{ // C6, first half: drop one condition and edit another. The
				// removal is checked against the server as well as against
				// state -- discoveryrule.update replaces the filter wholesale,
				// so an omitted condition is a deletion.
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
					testAccCheckLLDConditionCount("zabbix_lld_trapper.testlld", 2),
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
			{ // C7: import at full size, with the collection at three and the
				// caller-supplied formula ids in play
				ResourceName:      "zabbix_lld_trapper.testlld",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{ // C6, second half: back to no conditions at all. The rule keeps
				// existing; only the filter empties. evaltype has to come back
				// to the default with it -- Zabbix rejects "custom" with an
				// empty formula -- so this also covers reverting the two
				// attributes together in one update.
				Config: hcl(t, lldTrapperConditionHCL(``)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "condition.#", "0"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "evaltype", "andor"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "formula", ""),
					testAccCheckLLDConditionCount("zabbix_lld_trapper.testlld", 0),
				),
			},
			{ // C1: and the empty state is stable
				Config:   hcl(t, lldTrapperConditionHCL(``)),
				PlanOnly: true,
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

// lldTrapperPreprocessorHCL wraps a block of preprocessing steps in the rest of
// the fixture.
func lldTrapperPreprocessorHCL(preprocessors string) string {
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
	key = "lld.trapper.pre"

	name = "LLD Trapper Preprocessing"
	delay = "0"
` + preprocessors + `}
`
}

// TestAccResourceLLDPreprocessor is C1-C7 for `preprocessor` on a discovery
// rule.
//
// Discovery rules do not share the item preprocessing code: common_lld.go has
// its own lldGeneratePreprocessors/flattenlldPreprocessors and the client its
// own LLDRule.Preprocessors, so the coverage in TestAccResourceItemAgent says
// nothing about this path. Before this test nothing in the suite exercised LLD
// preprocessing on any version.
func TestAccResourceLLDPreprocessor(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // C1: no preprocessing
				Config: hcl(t, lldTrapperPreprocessorHCL(``)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.#", "0"),
					testAccCheckLLDPreprocessorCount("zabbix_lld_trapper.testlld", 0),
				),
			},
			{ // C2: one step
				Config: hcl(t, lldTrapperPreprocessorHCL(`
	preprocessor {
		type = "12"
		params = [ "$.data" ]
		error_handler = "0"
	}
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.#", "1"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.0.type", "12"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.0.params.0", "$.data"),
					testAccCheckLLDPreprocessorCount("zabbix_lld_trapper.testlld", 1),
				),
			},
			{ // C3: three steps, two of them the same type, so only content
				// and position tell them apart
				Config: hcl(t, lldTrapperPreprocessorHCL(`
	preprocessor {
		type = "12"
		params = [ "$.data" ]
		error_handler = "0"
	}
	preprocessor {
		type = "21"
		params = [ "return value;" ]
		error_handler = "0"
	}
	preprocessor {
		type = "21"
		params = [ "return value.trim();" ]
		error_handler = "0"
	}
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.#", "3"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.0.type", "12"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.1.params.0", "return value;"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.2.params.0", "return value.trim();"),
					testAccCheckLLDPreprocessorCount("zabbix_lld_trapper.testlld", 3),
				),
			},
			{ // C4 for a list, which is the opposite assertion to a set's:
				// preprocessing runs in sequence, so a reorder must produce a
				// diff and the new order must survive the round trip. If
				// discoveryrule.get ever started returning these in some other
				// order, this is the step that would say so.
				Config: hcl(t, lldTrapperPreprocessorHCL(`
	preprocessor {
		type = "21"
		params = [ "return value.trim();" ]
		error_handler = "0"
	}
	preprocessor {
		type = "12"
		params = [ "$.data" ]
		error_handler = "0"
	}
	preprocessor {
		type = "21"
		params = [ "return value;" ]
		error_handler = "0"
	}
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.0.params.0", "return value.trim();"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.1.type", "12"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.2.params.0", "return value;"),
				),
			},
			{ // and the reordered state is stable, not flapping back
				Config: hcl(t, lldTrapperPreprocessorHCL(`
	preprocessor {
		type = "21"
		params = [ "return value.trim();" ]
		error_handler = "0"
	}
	preprocessor {
		type = "12"
		params = [ "$.data" ]
		error_handler = "0"
	}
	preprocessor {
		type = "21"
		params = [ "return value;" ]
		error_handler = "0"
	}
`)),
				PlanOnly: true,
			},
			{ // C7: import at full size
				ResourceName:      "zabbix_lld_trapper.testlld",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{ // C5: edit the middle step's parameters, leave the two either
				// side of it alone
				Config: hcl(t, lldTrapperPreprocessorHCL(`
	preprocessor {
		type = "21"
		params = [ "return value.trim();" ]
		error_handler = "0"
	}
	preprocessor {
		type = "12"
		params = [ "$.items" ]
		error_handler = "0"
	}
	preprocessor {
		type = "21"
		params = [ "return value;" ]
		error_handler = "0"
	}
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.#", "3"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.1.params.0", "$.items"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.0.params.0", "return value.trim();"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.2.params.0", "return value;"),
				),
			},
			{ // C6: three down to one
				Config: hcl(t, lldTrapperPreprocessorHCL(`
	preprocessor {
		type = "12"
		params = [ "$.items" ]
		error_handler = "0"
	}
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.#", "1"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.0.type", "12"),
					testAccCheckLLDPreprocessorCount("zabbix_lld_trapper.testlld", 1),
				),
			},
			{ // C6: and down to none, confirmed on the server
				Config: hcl(t, lldTrapperPreprocessorHCL(``)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.#", "0"),
					testAccCheckLLDPreprocessorCount("zabbix_lld_trapper.testlld", 0),
				),
			},
			{ // C1: and empty is stable
				Config:   hcl(t, lldTrapperPreprocessorHCL(``)),
				PlanOnly: true,
			},
		},
	})
}
