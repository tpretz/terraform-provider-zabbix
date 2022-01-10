package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceTemplate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{ // simple create
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_hostgroup.testgrp.id ]
	host = "test-template"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "host", "test-template"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "name", "test-template"),
				),
			},
			{ // rename
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_hostgroup.testgrp.id ]
	host = "test-template-renamed"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "host", "test-template-renamed"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "name", "test-template"),
				),
			},
			{ // friendly name, description and a macro
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_hostgroup.testgrp.id ]
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
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "host", "test-template-renamed"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "name", "bob"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "description", "test description"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "macro.0.value", "fish"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "macro.1.value", "fish"),
				),
			},
			{ // remove all macros
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_hostgroup.testgrp.id ]
	host = "test-template-renamed"
	name = "bob"
	description = "test description"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "host", "test-template-renamed"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "name", "bob"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "description", "test description"),
				),
			},
		},
	})
}
