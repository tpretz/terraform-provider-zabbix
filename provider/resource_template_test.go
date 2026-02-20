package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceTemplate(t *testing.T) {
	id := resource.UniqueId()
	groupName := "test-group-" + id
	groupName2 := groupName + "-extra"
	tmplHost := "test-template-" + id
	tmplHost2 := "test-template-extra-" + id

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{ // simple create
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [ zabbix_hostgroup.testgrp.id ]
  host = %q
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "host", tmplHost),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "name", tmplHost),
				),
			},
			{ // rename
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [ zabbix_hostgroup.testgrp.id ]
  host = %q
}
`, groupName, tmplHost2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "host", tmplHost2),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "name", tmplHost),
				),
			},
			{ // friendly name, description and a macro
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [ zabbix_hostgroup.testgrp.id ]
  host = %q
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
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "host", tmplHost),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "name", "bob"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "description", "test description"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "macro.0.value", "fish"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "macro.1.value", "fish"),
				),
			},
			{ // remove all macros
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [ zabbix_hostgroup.testgrp.id ]
  host = %q
  name = "bob"
  description = "test description"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "host", tmplHost),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "name", "bob"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "description", "test description"),
				),
			},
			{ // add a second group, add a linked template
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_hostgroup" "testgrp2" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [ zabbix_hostgroup.testgrp.id, zabbix_hostgroup.testgrp2.id ]
  host = %q
  name = "bob"
  description = "test description"
}
resource "zabbix_template" "testtmpl2" {
  groups = [ zabbix_hostgroup.testgrp.id, zabbix_hostgroup.testgrp2.id ]
  host = %q

  templates = [ zabbix_template.testtmpl.id ]
}
`, groupName, groupName2, tmplHost, tmplHost2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl2", "templates.#", "1"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "groups.#", "2"),
					resource.TestCheckResourceAttr("zabbix_template.testtmpl2", "groups.#", "2"),
				),
			},
		},
	})
}
