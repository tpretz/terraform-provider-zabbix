package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

func TestAccResourceTemplate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // simple create
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "host", "test-template"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "name", "test-template"),
					// Zabbix generates the uuid; it must land in state on
					// every supported version
					resource.TestCheckResourceAttrSet("zabbix_template.testtmpl", "uuid"),
					resource.TestMatchResourceAttr("zabbix_template.testtmpl", "uuid", regexp.MustCompile(`^[0-9a-f]{32}$`)),
					// the version-gated attributes read back at their zero
					// value on every version, including the ones that do not
					// have the underlying field at all
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "vendor_name", ""),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "vendor_version", ""),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "readme", ""),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "wizard_ready", "false"),
				),
			},
			{ // rename
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template-renamed"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "host", "test-template-renamed"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "name", "test-template"),
				),
			},
			{ // friendly name, description and a macro
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template-renamed"
	name = "bob"
	description = "test description"

	macro {
		name = "{$TEST}"
		value = "fish"
	}
	
	macro {
		name = "{$TESTA}"
		value = "fish"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "host", "test-template-renamed"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "name", "bob"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "description", "test description"),
					// `macro` is a set: by content, never by index. Plural
					// coverage for it is in TestAccResourceTemplateCollections.
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "macro.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_template.testtmpl", "macro.*", map[string]string{
						"name":  "{$TEST}",
						"value": "fish",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_template.testtmpl", "macro.*", map[string]string{
						"name":  "{$TESTA}",
						"value": "fish",
					}),
				),
			},
			{ // remove all macros
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template-renamed"
	name = "bob"
	description = "test description"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "host", "test-template-renamed"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "name", "bob"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "description", "test description"),
				),
			},
			{ // add a second group, add a linked template
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_templategroup" "testgrp2" {
	name = "test-group-2" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id, zabbix_templategroup.testgrp2.id ]
	host = "test-template-renamed"
	name = "bob"
	description = "test description"
}
resource "zabbix_template" "testtmpl2" {
	groups = [ zabbix_templategroup.testgrp.id, zabbix_templategroup.testgrp2.id ]
	host = "test-template-2"

	templates = [ zabbix_template.testtmpl.id ]
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl2", "templates.#", "1"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "groups.#", "2"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl2", "groups.#", "2"),
				),
			},
			{ // vendor_name / vendor_version, 6.4+ only
				SkipFunc: skipBelow(t, zabbix.V64),
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template-renamed"
	name = "bob"

	vendor_name    = "test vendor"
	vendor_version = "1.0-0"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "vendor_name", "test vendor"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "vendor_version", "1.0-0"),
				),
			},
			{ // readme / wizard_ready, 7.4+ only
				SkipFunc: skipBelow(t, zabbix.V74),
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template-renamed"
	name = "bob"

	vendor_name    = "test vendor"
	vendor_version = "1.0-0"

	readme       = "test readme text"
	wizard_ready = true
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "readme", "test readme text"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "wizard_ready", "true"),
				),
			},
			{ // clearing the gated attributes again, 6.4+ only (below that
				// they were never set in the first place)
				SkipFunc: skipBelow(t, zabbix.V64),
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template-renamed"
	name = "bob"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "vendor_name", ""),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "vendor_version", ""),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "readme", ""),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "wizard_ready", "false"),
				),
			},
			{ // import: everything round-trips, including the generated uuid
				ResourceName:      "zabbix_template.testtmpl",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// templateCollectionsHCL wraps a zabbix_template body in the groups, linkable
// templates and nothing else the collection steps need.
func templateCollectionsHCL(body string) string {
	return `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_templategroup" "testgrp2" {
	name = "test-group-2"
}
resource "zabbix_templategroup" "testgrp3" {
	name = "test-group-3"
}
resource "zabbix_template" "testlink1" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template-link-1"
}
resource "zabbix_template" "testlink2" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template-link-2"
}
resource "zabbix_template" "testlink3" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template-link-3"
}
resource "zabbix_template" "testtmpl" {
	host = "test-template"
` + body + `}
`
}

// TestAccResourceTemplateCollections is C1-C7 for the three collections a
// template carries: `groups`, `templates` and `macro`.
//
// `macro` comes from the shared common_macro.go and is identical on
// zabbix_host, so it is exercised plural here, once, rather than twice;
// zabbix_host keeps its single-macro smoke check. `groups` and `templates` are
// tested on both, because the two resources have separate update paths -- and
// only one of them filters templates_clear through existingTemplateIds.
func TestAccResourceTemplateCollections(t *testing.T) {
	// C5 needs values carried between steps: the macros that were *not*
	// edited must keep their server-assigned hostmacroids, proving the update
	// touched one element rather than churning the whole collection.
	var macroIDB, macroIDG string

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // C2 for `groups`, C1 for the two optional collections.
				// `groups` is Required and Zabbix rejects a template with no
				// groups, so C1 and C6-to-zero do not apply to it.
				Config: hcl(t, templateCollectionsHCL(`
	groups = [ zabbix_templategroup.testgrp.id ]
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "groups.#", "1"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "templates.#", "0"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "macro.#", "0"),
					testAccCheckTemplateGroupCount("zabbix_template.testtmpl", 1),
				),
			},
			{ // C3 for all three: three groups, three linked templates, three
				// macros
				Config: hcl(t, templateCollectionsHCL(`
	groups = [
		zabbix_templategroup.testgrp.id,
		zabbix_templategroup.testgrp2.id,
		zabbix_templategroup.testgrp3.id,
	]

	templates = [
		zabbix_template.testlink1.id,
		zabbix_template.testlink2.id,
		zabbix_template.testlink3.id,
	]

	macro {
		name = "{$ALPHA}"
		value = "one"
	}
	macro {
		name = "{$BETA}"
		value = "two"
	}
	macro {
		name = "{$GAMMA}"
		value = "three"
	}
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "groups.#", "3"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_template.testtmpl", "groups.*",
						tmplGroupAddr(t, "testgrp"), "id"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_template.testtmpl", "groups.*",
						tmplGroupAddr(t, "testgrp2"), "id"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_template.testtmpl", "groups.*",
						tmplGroupAddr(t, "testgrp3"), "id"),
					testAccCheckTemplateGroupCount("zabbix_template.testtmpl", 3),

					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "templates.#", "3"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_template.testtmpl", "templates.*",
						"zabbix_template.testlink1", "id"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_template.testtmpl", "templates.*",
						"zabbix_template.testlink2", "id"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_template.testtmpl", "templates.*",
						"zabbix_template.testlink3", "id"),
					testAccCheckTemplateLinkedCount("zabbix_template.testtmpl", 3),

					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "macro.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_template.testtmpl", "macro.*", map[string]string{
						"name":  "{$ALPHA}",
						"value": "one",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_template.testtmpl", "macro.*", map[string]string{
						"name":  "{$BETA}",
						"value": "two",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_template.testtmpl", "macro.*", map[string]string{
						"name":  "{$GAMMA}",
						"value": "three",
					}),
					testAccCheckTemplateMacroCount("zabbix_template.testtmpl", 3),

					// remember the hostmacroids of the two macros C5 leaves
					// alone. They are populated at all only since the client's
					// Macro.MacroID json tag was corrected from "hostmacroids"
					// (the usermacro.get filter name) to "hostmacroid" (the
					// object property).
					testAccRecordSetElemAttr("zabbix_template.testtmpl", "macro", "name", "{$BETA}", "id", &macroIDB),
					testAccRecordSetElemAttr("zabbix_template.testtmpl", "macro", "name", "{$GAMMA}", "id", &macroIDG),
				),
			},
			{ // C4: every one of the three rewritten in a different order.
				// All three are sets, so this must plan clean.
				Config: hcl(t, templateCollectionsHCL(`
	groups = [
		zabbix_templategroup.testgrp3.id,
		zabbix_templategroup.testgrp.id,
		zabbix_templategroup.testgrp2.id,
	]

	templates = [
		zabbix_template.testlink2.id,
		zabbix_template.testlink3.id,
		zabbix_template.testlink1.id,
	]

	macro {
		name = "{$GAMMA}"
		value = "three"
	}
	macro {
		name = "{$ALPHA}"
		value = "one"
	}
	macro {
		name = "{$BETA}"
		value = "two"
	}
`)),
				PlanOnly: true,
			},
			{ // C7: import with all three at full size
				ResourceName:      "zabbix_template.testtmpl",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{ // C5: change one macro's value. The other two must be untouched
				// -- same value, same hostmacroid -- so the update edited one
				// element rather than replacing the collection.
				//
				// The *edited* macro does get a new hostmacroid. An element
				// whose value changed hashes differently, so it arrives at the
				// provider with no id and template.update is handed a macro
				// list in which {$ALPHA} carries no hostmacroid; Zabbix then
				// drops the old row and inserts a new one. Nothing references
				// a hostmacroid, so this is cosmetic -- but it is the reason
				// this step asserts identity on the untouched elements rather
				// than on the edited one. Reusing the prior id the way
				// hostReuseInterfaceIDs does for interfaces would remove it.
				Config: hcl(t, templateCollectionsHCL(`
	groups = [
		zabbix_templategroup.testgrp.id,
		zabbix_templategroup.testgrp2.id,
		zabbix_templategroup.testgrp3.id,
	]

	templates = [
		zabbix_template.testlink1.id,
		zabbix_template.testlink2.id,
		zabbix_template.testlink3.id,
	]

	macro {
		name = "{$ALPHA}"
		value = "one-changed"
	}
	macro {
		name = "{$BETA}"
		value = "two"
	}
	macro {
		name = "{$GAMMA}"
		value = "three"
	}
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "macro.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_template.testtmpl", "macro.*", map[string]string{
						"name":  "{$ALPHA}",
						"value": "one-changed",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_template.testtmpl", "macro.*", map[string]string{
						"name":  "{$BETA}",
						"value": "two",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_template.testtmpl", "macro.*", map[string]string{
						"name":  "{$GAMMA}",
						"value": "three"}),
					testAccCheckSetElemAttr("zabbix_template.testtmpl", "macro", "name", "{$BETA}", "id", &macroIDB),
					testAccCheckSetElemAttr("zabbix_template.testtmpl", "macro", "name", "{$GAMMA}", "id", &macroIDG),
					// the other two collections did not move
					testAccCheckTemplateGroupCount("zabbix_template.testtmpl", 3),
					testAccCheckTemplateLinkedCount("zabbix_template.testtmpl", 3),
				),
			},
			{ // C6, first half: one out of each of the three
				Config: hcl(t, templateCollectionsHCL(`
	groups = [
		zabbix_templategroup.testgrp.id,
		zabbix_templategroup.testgrp3.id,
	]

	templates = [
		zabbix_template.testlink1.id,
		zabbix_template.testlink3.id,
	]

	macro {
		name = "{$ALPHA}"
		value = "one-changed"
	}
	macro {
		name = "{$GAMMA}"
		value = "three"
	}
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "groups.#", "2"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "templates.#", "2"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "macro.#", "2"),
					// survivors, by content
					resource.TestCheckTypeSetElemAttrPair("zabbix_template.testtmpl", "templates.*",
						"zabbix_template.testlink1", "id"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_template.testtmpl", "templates.*",
						"zabbix_template.testlink3", "id"),
					// and the server agrees the third of each is gone. This is
					// the step that matters for `templates`: unlinking is not
					// a matter of omitting the id, it needs templates_clear.
					testAccCheckTemplateGroupCount("zabbix_template.testtmpl", 2),
					testAccCheckTemplateLinkedCount("zabbix_template.testtmpl", 2),
					testAccCheckTemplateMacroCount("zabbix_template.testtmpl", 2),
				),
			},
			{ // C6, second half: `templates` and `macro` back to zero,
				// `groups` back to the one Zabbix insists on
				Config: hcl(t, templateCollectionsHCL(`
	groups = [ zabbix_templategroup.testgrp.id ]
`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "groups.#", "1"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "templates.#", "0"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "macro.#", "0"),
					testAccCheckTemplateGroupCount("zabbix_template.testtmpl", 1),
					testAccCheckTemplateLinkedCount("zabbix_template.testtmpl", 0),
					testAccCheckTemplateMacroCount("zabbix_template.testtmpl", 0),
				),
			},
			{ // C1: and empty is stable
				Config: hcl(t, templateCollectionsHCL(`
	groups = [ zabbix_templategroup.testgrp.id ]
`)),
				PlanOnly: true,
			},
		},
	})
}

// TestAccResourceTemplateClearDestroyed is the case existingTemplateIds exists
// for, at more than one template.
//
// A linked template is unlinked by naming it in `templates_clear`, and Zabbix
// 7.0 made an unknown object id a hard error. So when a template is destroyed
// in the same apply that unlinks it, whether the update succeeds depends on
// which of the two Terraform does first -- and if the destroy wins, the id in
// templates_clear no longer resolves and the whole update fails.
//
// resourceHostUpdate filters the clear list through existingTemplateIds for
// exactly this reason; resourceTemplateUpdate now does the same.
func TestAccResourceTemplateClearDestroyed(t *testing.T) {
	const bothLinks = `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testlink1" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template-link-1"
}
resource "zabbix_template" "testlink2" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template-link-2"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"

	templates = [
		zabbix_template.testlink1.id,
		zabbix_template.testlink2.id,
	]
}
`

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // two linked templates
				Config: hcl(t, bothLinks),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "templates.#", "2"),
					testAccCheckTemplateLinkedCount("zabbix_template.testtmpl", 2),
				),
			},
			{ // one of them removed from `templates` *and* destroyed in the
				// same apply
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testlink1" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template-link-1"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"

	templates = [
		zabbix_template.testlink1.id,
	]
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "templates.#", "1"),
					resource.TestCheckTypeSetElemAttrPair("zabbix_template.testtmpl", "templates.*",
						"zabbix_template.testlink1", "id"),
					testAccCheckTemplateLinkedCount("zabbix_template.testtmpl", 1),
				),
			},
		},
	})
}
