package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// triggerFixtureHCL is the scaffolding the trigger steps share: a template, an
// item to write expressions against, and three further triggers to depend on.
// Three, all of the same kind, because `dependencies` is a set of ids and
// nothing but the id distinguishes one element from another.
const triggerFixtureHCL = `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_trapper" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "trapper.test"

	name = "Trapper Item"
	valuetype = "float"
}
resource "zabbix_trigger" "testtriggerdep" {
	name = "Test Trigger Dependency"
	expression = "last(/test-template/trapper.test)=2"

	depends_on = [zabbix_item_trapper.testitem]
}
resource "zabbix_trigger" "testtriggerdep2" {
	name = "Test Trigger Dependency 2"
	expression = "last(/test-template/trapper.test)=3"

	depends_on = [zabbix_item_trapper.testitem]
}
resource "zabbix_trigger" "testtriggerdep3" {
	name = "Test Trigger Dependency 3"
	expression = "last(/test-template/trapper.test)=4"

	depends_on = [zabbix_item_trapper.testitem]
}
`

// triggerHCL wraps a zabbix_trigger body in the fixture above.
func triggerHCL(body string) string {
	return triggerFixtureHCL + `
resource "zabbix_trigger" "testtrigger" {
	name = "Test Trigger Renamed"
	expression = "last(/test-template/trapper.test)=1"
	comments = "test comment"
	priority = "high"
	enabled = false
	manual_close = true
	url = "http://example.com"

	depends_on = [zabbix_item_trapper.testitem]
` + body + `}
`
}

func TestAccResourceTrigger(t *testing.T) {
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
			{ // simple create: item to reference, plus a trigger on it
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_trapper" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "trapper.test"

	name = "Trapper Item"
	valuetype = "float"
}
resource "zabbix_trigger" "testtrigger" {
	name = "Test Trigger"
	expression = "last(/test-template/trapper.test)=0"

	depends_on = [zabbix_item_trapper.testitem]
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "name", "Test Trigger"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "expression", "last(/test-template/trapper.test)=0"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "priority", "not_classified"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "enabled", "true"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "manual_close", "false"),
					// C1: neither optional collection is set
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "tag.#", "0"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "dependencies.#", "0"),
				),
			},
			{ // modify: priority, comments, url, manual_close, disable, and
				// C2 for `tag`
				Config: hcl(t, triggerHCL(`
	tag {
		key = "testtag"
		value = "testvalue"
	}
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "name", "Test Trigger Renamed"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "expression", "last(/test-template/trapper.test)=1"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "comments", "test comment"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "priority", "high"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "enabled", "false"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "manual_close", "true"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "url", "http://example.com"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "tag.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_trigger.testtrigger", "tag.*", map[string]string{
						"key":   "testtag",
						"value": "testvalue",
					}),
				),
			},
			{ // C3 for both collections at once: three tags (two sharing a
				// key, so content and not the key identifies an element) and
				// three dependencies
				Config: hcl(t, triggerHCL(`
	tag {
		key = "testtag"
		value = "testvalue"
	}
	tag {
		key = "testtag"
		value = "othervalue"
	}
	tag {
		key = "scope"
		value = "availability"
	}

	dependencies = [
		zabbix_trigger.testtriggerdep.id,
		zabbix_trigger.testtriggerdep2.id,
		zabbix_trigger.testtriggerdep3.id,
	]
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "tag.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_trigger.testtrigger", "tag.*", map[string]string{
						"key":   "testtag",
						"value": "testvalue",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_trigger.testtrigger", "tag.*", map[string]string{
						"key":   "testtag",
						"value": "othervalue",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_trigger.testtrigger", "tag.*", map[string]string{
						"key":   "scope",
						"value": "availability",
					}),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "dependencies.#", "3"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_trigger.testtrigger", "dependencies.*", "zabbix_trigger.testtriggerdep", "id"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_trigger.testtrigger", "dependencies.*", "zabbix_trigger.testtriggerdep2", "id"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_trigger.testtrigger", "dependencies.*", "zabbix_trigger.testtriggerdep3", "id"),
					testAccCheckTriggerDependencyCount("zabbix_trigger.testtrigger", 3),
				),
			},
			{ // C4: the same elements in a different order. Both are sets, so
				// this must plan clean.
				Config: hcl(t, triggerHCL(`
	tag {
		key = "scope"
		value = "availability"
	}
	tag {
		key = "testtag"
		value = "othervalue"
	}
	tag {
		key = "testtag"
		value = "testvalue"
	}

	dependencies = [
		zabbix_trigger.testtriggerdep3.id,
		zabbix_trigger.testtriggerdep.id,
		zabbix_trigger.testtriggerdep2.id,
	]
`)),
				PlanOnly: true,
			},
			{ // C7: import with both collections at full size
				ResourceName:      "zabbix_trigger.testtrigger",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{ // C5: edit one tag, leave the other two and every dependency
				// alone
				Config: hcl(t, triggerHCL(`
	tag {
		key = "testtag"
		value = "testvalue"
	}
	tag {
		key = "testtag"
		value = "othervalue"
	}
	tag {
		key = "scope"
		value = "performance"
	}

	dependencies = [
		zabbix_trigger.testtriggerdep.id,
		zabbix_trigger.testtriggerdep2.id,
		zabbix_trigger.testtriggerdep3.id,
	]
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "tag.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_trigger.testtrigger", "tag.*", map[string]string{
						"key":   "scope",
						"value": "performance",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_trigger.testtrigger", "tag.*", map[string]string{
						"key":   "testtag",
						"value": "testvalue",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_trigger.testtrigger", "tag.*", map[string]string{
						"key":   "testtag",
						"value": "othervalue",
					}),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "dependencies.#", "3"),
					testAccCheckTriggerDependencyCount("zabbix_trigger.testtrigger", 3),
				),
			},
			{ // C6: three dependencies down to one, one tag removed
				Config: hcl(t, triggerHCL(`
	tag {
		key = "testtag"
		value = "testvalue"
	}
	tag {
		key = "scope"
		value = "performance"
	}

	dependencies = [
		zabbix_trigger.testtriggerdep2.id,
	]
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "tag.#", "2"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "dependencies.#", "1"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_trigger.testtrigger", "dependencies.*", "zabbix_trigger.testtriggerdep2", "id"),
					testAccCheckTriggerDependencyCount("zabbix_trigger.testtrigger", 1),
					testAccCheckTriggerTagCount("zabbix_trigger.testtrigger", 2),
				),
			},
			{ // C6 to zero, both collections. Checked against the server as
				// well as against state: trigger.update replaces dependencies
				// and tags wholesale, so an omitted collection is a deletion
				// -- unless the provider omits the property itself, in which
				// case the server keeps what it had and only state moves.
				Config: hcl(t, triggerHCL(``)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "tag.#", "0"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrigger", "dependencies.#", "0"),
					testAccCheckTriggerDependencyCount("zabbix_trigger.testtrigger", 0),
					testAccCheckTriggerTagCount("zabbix_trigger.testtrigger", 0),
				),
			},
			{ // C1: and empty is stable
				Config:   hcl(t, triggerHCL(``)),
				PlanOnly: true,
			},
		},
	})
}
