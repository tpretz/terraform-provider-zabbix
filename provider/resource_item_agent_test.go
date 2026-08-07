package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceItemAgent(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // simple create
				Config: hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template"
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitem"

	name = "Test Item"
	valuetype = "text"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "key", "testitem"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "name", "Test Item"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "valuetype", "text"),
				),
			},
			{ // change values
				Config: hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template"
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitemchanged"

	name = "Test Item Changed"
	valuetype = "float"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "key", "testitemchanged"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "name", "Test Item Changed"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "valuetype", "float"),
				),
			},
			{ // optionals
				Config: hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template"
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitemchanged"

	name = "Test Item Changed"
	valuetype = "float"

	active = true
	delay = "2m"
	history = "1h"
	trends = "7d"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "active", "true"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "delay", "2m"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "history", "1h"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "trends", "7d"),
				),
			},
			{ // optionals, with tags
				Config: hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template"
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitemchanged"

	name = "Test Item Changed"
	valuetype = "float"

	active = true
	delay = "2m"
	history = "1h"
	trends = "7d"

	tag {
		key = "action"
		value = "test"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "active", "true"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "delay", "2m"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "history", "1h"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "trends", "7d"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "tag.0.key", "action"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "tag.0.value", "test"),
				),
			},
			{ // attached to interface id
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_host.testhost.id
	key = "testitemchanged"
	interfaceid = one(zabbix_host.testhost.interface).id

	name = "Test Item Changed"
	valuetype = "float"

	active = false
	delay = "2m"
	history = "1h"
	trends = "7d"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "active", "false"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "delay", "2m"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "history", "1h"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "trends", "7d"),
					resource.TestCheckResourceAttrSet("zabbix_item_agent.testitem", "interfaceid"),
				),
			},
			{ // preprocessor, javascript
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
locals {
	script = <<-EOT
	  var fish = false;
	  return fish;
	EOT
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_host.testhost.id
	key = "testitemchanged"
	interfaceid = one(zabbix_host.testhost.interface).id

	name = "Test Item Changed"
	valuetype = "float"

	active = false
	delay = "2m"
	history = "1h"
	trends = "7d"

	preprocessor {
		type = "21"
		params = [ "var bob = true;", "return bob;" ]
		error_handler = "0"
	}
	
	preprocessor {
		type = "21"
		params = split("\n", "var cheese = true;\nreturn cheese;")
		error_handler = "0"
	}
	
	preprocessor {
		type = "21"
		params = split("\n", trimspace(local.script))
		# note: change schema to allow blank lines
		error_handler = "0"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "active", "false"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "delay", "2m"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "history", "1h"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "trends", "7d"),
					resource.TestCheckResourceAttrSet("zabbix_item_agent.testitem", "interfaceid"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.0.type", "21"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.0.params.0", "var bob = true;"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.0.params.1", "return bob;"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.1.type", "21"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.1.params.0", "var cheese = true;"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.1.params.1", "return cheese;"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.2.type", "21"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.2.params.0", "var fish = false;"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.2.params.1", "return fish;"),
				),
			},
			{ // preprocessor
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_host.testhost.id
	key = "testitemchanged"
	interfaceid = one(zabbix_host.testhost.interface).id

	name = "Test Item Changed"
	valuetype = "float"

	active = false
	delay = "2m"
	history = "1h"
	trends = "7d"

	preprocessor {
		type = "1"
		params = [ "55" ]
		error_handler = "0" # issue for version 4
	}
	
	preprocessor {
		type = "10"
		error_handler = "0"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "active", "false"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "delay", "2m"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "history", "1h"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "trends", "7d"),
					resource.TestCheckResourceAttrSet("zabbix_item_agent.testitem", "interfaceid"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.0.type", "1"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.0.params.0", "55"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.1.type", "10"),
				),
			},
			{ // C4 for a list: preprocessing order IS semantic, so a reorder must
				// produce a diff and the new order must survive the round trip.
				// Verified against live 6.0/7.0/7.4/8.0: item.get returns
				// preprocessing in submission order, and the objects carry no
				// sortorder field — position is the only thing conveying sequence.
				// Nothing else in the suite asserts this, so a future Zabbix that
				// reordered these would silently change what the steps compute.
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_host.testhost.id
	key = "testitemchanged"
	interfaceid = one(zabbix_host.testhost.interface).id

	name = "Test Item Changed"
	valuetype = "float"

	active = false
	delay = "2m"
	history = "1h"
	trends = "7d"

	preprocessor {
		type = "10"
		error_handler = "0"
	}

	preprocessor {
		type = "1"
		params = [ "55" ]
		error_handler = "0"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.0.type", "10"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.1.type", "1"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.1.params.0", "55"),
				),
			},
			{ // and the reordered state is stable, not flapping back
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_host.testhost.id
	key = "testitemchanged"
	interfaceid = one(zabbix_host.testhost.interface).id

	name = "Test Item Changed"
	valuetype = "float"

	active = false
	delay = "2m"
	history = "1h"
	trends = "7d"

	preprocessor {
		type = "10"
		error_handler = "0"
	}

	preprocessor {
		type = "1"
		params = [ "55" ]
		error_handler = "0"
	}
}
`),
				PlanOnly: true,
			},
			{ // C6 for `preprocessor`, first half: two steps down to one.
				// Verified against the server, since item.update replaces the
				// preprocessing array wholesale and an omitted step is a
				// deletion.
				Config: hcl(t, itemAgentPreprocessorHCL(`
	preprocessor {
		type = "1"
		params = [ "55" ]
		error_handler = "0"
	}
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.#", "1"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.0.type", "1"),
					testAccCheckItemPreprocessorCount("zabbix_item_agent.testitem", 1),
				),
			},
			{ // C6 second half: no preprocessing at all
				Config: hcl(t, itemAgentPreprocessorHCL(``)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.#", "0"),
					testAccCheckItemPreprocessorCount("zabbix_item_agent.testitem", 0),
				),
			},
			{ // C1: and the empty state is stable
				Config:   hcl(t, itemAgentPreprocessorHCL(``)),
				PlanOnly: true,
			},
		},
	})
}

// itemAgentPreprocessorHCL wraps a block of preprocessing steps in the rest of
// the fixture, so the C6 steps differ only by the collection under test.
func itemAgentPreprocessorHCL(preprocessors string) string {
	return `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_host.testhost.id
	key = "testitemchanged"
	interfaceid = one(zabbix_host.testhost.interface).id

	name = "Test Item Changed"
	valuetype = "float"

	active = false
	delay = "2m"
	history = "1h"
	trends = "7d"
` + preprocessors + `}
`
}

// itemAgentTagHCL is the same idea for `tag`.
func itemAgentTagHCL(tags string) string {
	return `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template"
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitem.tags"

	name = "Test Item Tags"
	valuetype = "float"
` + tags + `}
`
}

// TestAccResourceItemAgentTags covers C1-C7 for the shared `tag` collection.
//
// `tag` comes from common_tag.go and is merged into the schema of every
// zabbix_item_*, zabbix_proto_item_* and zabbix_lld_* resource by
// common_item.go / common_lld.go, plus zabbix_host and zabbix_trigger. It is
// one TypeSet, one tagGenerate, one flattenTags. So it is exercised plural
// *here*, once, rather than in eleven near-identical fixtures; the other item
// resources deliberately keep their single-tag smoke check. zabbix_host and
// zabbix_trigger have their own multi-tag steps because their build/update
// paths handle tags themselves rather than going through common_item.go.
func TestAccResourceItemAgentTags(t *testing.T) {
	// C5 needs a value carried across steps. Items have no per-tag id, so
	// what is asserted here is that editing one tag leaves the other two
	// exactly as they were -- including the item id, i.e. the item was
	// updated rather than replaced.
	var itemID string

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // C1: no tags at all
				Config: hcl(t, itemAgentTagHCL(``)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "tag.#", "0"),
				),
			},
			{ // C2: one
				Config: hcl(t, itemAgentTagHCL(`
	tag {
		key = "component"
		value = "cpu"
	}
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "tag.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_item_agent.testitem", "tag.*", map[string]string{
						"key":   "component",
						"value": "cpu",
					}),
				),
			},
			{ // C3: three, two of them sharing a key -- Zabbix allows repeated
				// tag keys with different values, so key alone cannot identify
				// an element and content has to
				Config: hcl(t, itemAgentTagHCL(`
	tag {
		key = "component"
		value = "cpu"
	}
	tag {
		key = "component"
		value = "memory"
	}
	tag {
		key = "scope"
		value = "performance"
	}
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "tag.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_item_agent.testitem", "tag.*", map[string]string{
						"key":   "component",
						"value": "cpu",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_item_agent.testitem", "tag.*", map[string]string{
						"key":   "component",
						"value": "memory",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_item_agent.testitem", "tag.*", map[string]string{
						"key":   "scope",
						"value": "performance",
					}),
					testAccRecordAttr("zabbix_item_agent.testitem", "id", &itemID),
				),
			},
			{ // C4: the same three in a different order -- a set, so this must
				// plan clean
				Config: hcl(t, itemAgentTagHCL(`
	tag {
		key = "scope"
		value = "performance"
	}
	tag {
		key = "component"
		value = "memory"
	}
	tag {
		key = "component"
		value = "cpu"
	}
`)),
				PlanOnly: true,
			},
			{ // C7: import at full size -- the only check that flattenTags and
				// the set hash agree
				ResourceName:      "zabbix_item_agent.testitem",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{ // C5: edit one of the three. The other two must be untouched,
				// and the item itself must have been updated in place rather
				// than replaced.
				Config: hcl(t, itemAgentTagHCL(`
	tag {
		key = "component"
		value = "cpu"
	}
	tag {
		key = "component"
		value = "memory"
	}
	tag {
		key = "scope"
		value = "availability"
	}
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "tag.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_item_agent.testitem", "tag.*", map[string]string{
						"key":   "scope",
						"value": "availability",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_item_agent.testitem", "tag.*", map[string]string{
						"key":   "component",
						"value": "cpu",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_item_agent.testitem", "tag.*", map[string]string{
						"key":   "component",
						"value": "memory",
					}),
					resource.TestCheckResourceAttrPtr("zabbix_item_agent.testitem", "id", &itemID),
				),
			},
			{ // C6: three -> two
				Config: hcl(t, itemAgentTagHCL(`
	tag {
		key = "component"
		value = "cpu"
	}
	tag {
		key = "scope"
		value = "availability"
	}
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "tag.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_item_agent.testitem", "tag.*", map[string]string{
						"key":   "component",
						"value": "cpu",
					}),
					testAccCheckItemTagCount("zabbix_item_agent.testitem", 2),
				),
			},
			{ // C6: and back to none. item.update replaces the tag array
				// wholesale, so this has to be confirmed on the server.
				Config: hcl(t, itemAgentTagHCL(``)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "tag.#", "0"),
					testAccCheckItemTagCount("zabbix_item_agent.testitem", 0),
				),
			},
		},
	})
}
