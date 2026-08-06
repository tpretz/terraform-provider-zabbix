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
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "macro.0.value", "fish"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "macro.1.value", "fish"),
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
