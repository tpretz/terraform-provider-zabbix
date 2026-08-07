package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// protoTriggerFixtureHCL is the shared scaffolding: a template, a discovery
// rule, and an item prototype for the trigger prototype's expression to
// reference. A trigger prototype has no ruleid of its own -- Zabbix infers the
// owning rule from the prototypes named in the expression -- so there must be
// at least one item prototype in the expression or the create is rejected.
//
// It also carries the three things `dependencies` needs to be tested plural:
// two further trigger prototypes on the same rule, and one plain trigger. A
// trigger prototype may depend on either kind, so the set holds two different
// species of id and can only tell them apart by content.
const protoTriggerFixtureHCL = `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template"
}
resource "zabbix_lld_trapper" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "test.lld.rule"
	name = "Test LLD Rule"
	delay = "0"
}
resource "zabbix_proto_item_trapper" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "trapper[{#FSNAME}]"

	name = "Proto Trapper {#FSNAME}"
	valuetype = "float"
}
resource "zabbix_item_trapper" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "trapper.plain"

	name = "Plain Trapper Item"
	valuetype = "float"
}
resource "zabbix_proto_trigger" "testprotodep" {
	name = "Proto Trigger Dep {#FSNAME}"
	expression = "last(/test-template/trapper[{#FSNAME}])=101"

	depends_on = [zabbix_proto_item_trapper.testproto]
}
resource "zabbix_proto_trigger" "testprotodep2" {
	name = "Proto Trigger Dep 2 {#FSNAME}"
	expression = "last(/test-template/trapper[{#FSNAME}])=102"

	depends_on = [zabbix_proto_item_trapper.testproto]
}
resource "zabbix_trigger" "testtriggerdep" {
	name = "Plain Trigger Dep"
	expression = "last(/test-template/trapper.plain)=103"

	depends_on = [zabbix_item_trapper.testitem]
}
`

// protoTriggerHCL wraps a zabbix_proto_trigger body in the fixture above.
func protoTriggerHCL(body string) string {
	return protoTriggerFixtureHCL + `
resource "zabbix_proto_trigger" "testtrigger" {
	name = "Proto Trigger Renamed {#FSNAME}"
	expression = "last(/test-template/trapper[{#FSNAME}])>10"
	recovery_expression = "last(/test-template/trapper[{#FSNAME}])<5"
	comments = "proto trigger comment"
	priority = "average"
	enabled = false
	manual_close = true
	multiple = true
	url = "http://example.com/proto"

	depends_on = [zabbix_proto_item_trapper.testproto]
` + body + `}
`
}

func TestAccResourceProtoTrigger(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // create
				Config: hcl(t, protoTriggerFixtureHCL+`
resource "zabbix_proto_trigger" "testtrigger" {
	name = "Proto Trigger {#FSNAME}"
	expression = "last(/test-template/trapper[{#FSNAME}])=0"

	depends_on = [zabbix_proto_item_trapper.testproto]
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "name", "Proto Trigger {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "expression", "last(/test-template/trapper[{#FSNAME}])=0"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "priority", "not_classified"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "enabled", "true"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "manual_close", "false"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "multiple", "false"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "recovery_none", "false"),
					// C1 for both collections
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "tag.#", "0"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "dependencies.#", "0"),
				),
			},
			{ // update: severity, comments, url, manual close, disable, a
				// recovery expression (recovery mode 1), and C2 for both
				// collections
				Config: hcl(t, protoTriggerHCL(`
	tag {
		key = "protokey"
		value = "protovalue"
	}

	dependencies = [ zabbix_proto_trigger.testprotodep.id ]
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "name", "Proto Trigger Renamed {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "expression", "last(/test-template/trapper[{#FSNAME}])>10"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "recovery_expression", "last(/test-template/trapper[{#FSNAME}])<5"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "comments", "proto trigger comment"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "priority", "average"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "enabled", "false"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "manual_close", "true"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "multiple", "true"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "url", "http://example.com/proto"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_proto_trigger.testtrigger", "tag.*", map[string]string{
						"key":   "protokey",
						"value": "protovalue",
					}),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "dependencies.#", "1"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_proto_trigger.testtrigger", "dependencies.*",
						"zabbix_proto_trigger.testprotodep", "id"),
					testAccCheckProtoTriggerDependencyCount("zabbix_proto_trigger.testtrigger", 1),
				),
			},
			{ // C3: three tags (two sharing a key) and three dependencies,
				// two on trigger prototypes and one on a plain trigger
				Config: hcl(t, protoTriggerHCL(`
	tag {
		key = "protokey"
		value = "protovalue"
	}
	tag {
		key = "protokey"
		value = "othervalue"
	}
	tag {
		key = "scope"
		value = "availability"
	}

	dependencies = [
		zabbix_proto_trigger.testprotodep.id,
		zabbix_proto_trigger.testprotodep2.id,
		zabbix_trigger.testtriggerdep.id,
	]
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "tag.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_proto_trigger.testtrigger", "tag.*", map[string]string{
						"key":   "protokey",
						"value": "protovalue",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_proto_trigger.testtrigger", "tag.*", map[string]string{
						"key":   "protokey",
						"value": "othervalue",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_proto_trigger.testtrigger", "tag.*", map[string]string{
						"key":   "scope",
						"value": "availability",
					}),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "dependencies.#", "3"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_proto_trigger.testtrigger", "dependencies.*",
						"zabbix_proto_trigger.testprotodep", "id"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_proto_trigger.testtrigger", "dependencies.*",
						"zabbix_proto_trigger.testprotodep2", "id"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_proto_trigger.testtrigger", "dependencies.*",
						"zabbix_trigger.testtriggerdep", "id"),
					testAccCheckProtoTriggerDependencyCount("zabbix_proto_trigger.testtrigger", 3),
				),
			},
			{ // C4: same elements, different order -- both are sets, so this
				// must plan clean
				Config: hcl(t, protoTriggerHCL(`
	tag {
		key = "scope"
		value = "availability"
	}
	tag {
		key = "protokey"
		value = "othervalue"
	}
	tag {
		key = "protokey"
		value = "protovalue"
	}

	dependencies = [
		zabbix_trigger.testtriggerdep.id,
		zabbix_proto_trigger.testprotodep2.id,
		zabbix_proto_trigger.testprotodep.id,
	]
`)),
				PlanOnly: true,
			},
			{ // C7: import with both collections at full size
				ResourceName:      "zabbix_proto_trigger.testtrigger",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{ // C5: edit one tag and swap one dependency; the rest untouched
				Config: hcl(t, protoTriggerHCL(`
	tag {
		key = "protokey"
		value = "protovalue"
	}
	tag {
		key = "protokey"
		value = "othervalue"
	}
	tag {
		key = "scope"
		value = "performance"
	}

	dependencies = [
		zabbix_proto_trigger.testprotodep.id,
		zabbix_proto_trigger.testprotodep2.id,
		zabbix_trigger.testtriggerdep.id,
	]
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "tag.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_proto_trigger.testtrigger", "tag.*", map[string]string{
						"key":   "scope",
						"value": "performance",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_proto_trigger.testtrigger", "tag.*", map[string]string{
						"key":   "protokey",
						"value": "protovalue",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_proto_trigger.testtrigger", "tag.*", map[string]string{
						"key":   "protokey",
						"value": "othervalue",
					}),
					testAccCheckProtoTriggerDependencyCount("zabbix_proto_trigger.testtrigger", 3),
				),
			},
			{ // C6: three dependencies down to one, one tag removed
				Config: hcl(t, protoTriggerHCL(`
	tag {
		key = "protokey"
		value = "protovalue"
	}
	tag {
		key = "scope"
		value = "performance"
	}

	dependencies = [ zabbix_trigger.testtriggerdep.id ]
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "tag.#", "2"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "dependencies.#", "1"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_proto_trigger.testtrigger", "dependencies.*",
						"zabbix_trigger.testtriggerdep", "id"),
					testAccCheckProtoTriggerDependencyCount("zabbix_proto_trigger.testtrigger", 1),
					testAccCheckProtoTriggerTagCount("zabbix_proto_trigger.testtrigger", 2),
				),
			},
			{ // C6 to zero, both collections, confirmed on the server:
				// triggerprototype.update replaces each wholesale, so an
				// omitted element is a deletion -- provided the provider sends
				// the property at all.
				Config: hcl(t, protoTriggerHCL(``)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "tag.#", "0"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrigger", "dependencies.#", "0"),
					testAccCheckProtoTriggerDependencyCount("zabbix_proto_trigger.testtrigger", 0),
					testAccCheckProtoTriggerTagCount("zabbix_proto_trigger.testtrigger", 0),
				),
			},
			{ // C1: and empty is stable
				Config:   hcl(t, protoTriggerHCL(``)),
				PlanOnly: true,
			},
		},
	})
}
