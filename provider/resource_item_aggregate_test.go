package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccResourceItemAggregate(t *testing.T) {
	id := resource.UniqueId()
	groupName := "test-group-" + id
	tmplHost := "test-template-" + id

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{ // simple create
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					fmt.Printf("got version %d\n", api.Config.Version)
					return api.Config.Version >= 50400, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [ zabbix_hostgroup.testgrp.id ]
  host   = %q
}
resource "zabbix_item_aggregate" "testitem" {
  hostid = zabbix_template.testtmpl.id
  key    = "grpavg[\"${zabbix_hostgroup.testgrp.name}\", \"not_real\", last]"

  name      = "Test Item"
  valuetype = "unsigned"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "key", fmt.Sprintf("grpavg[\"%s\", \"not_real\", last]", groupName)),
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "name", "Test Item"),
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "valuetype", "unsigned"),
				),
			},
			{ // change values
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version >= 50400, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [ zabbix_hostgroup.testgrp.id ]
  host   = %q
}
resource "zabbix_item_aggregate" "testitem" {
  hostid = zabbix_template.testtmpl.id
  key    = "grpsum[\"${zabbix_hostgroup.testgrp.name}\", \"not_real\", last]"

  name      = "Test Item Changed"
  valuetype = "float"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "key", fmt.Sprintf("grpsum[\"%s\", \"not_real\", last]", groupName)),
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "name", "Test Item Changed"),
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "valuetype", "float"),
				),
			},
			{ // preprocessor, <v5
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version >= 50000, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [ zabbix_hostgroup.testgrp.id ]
  host   = %q
}
resource "zabbix_item_aggregate" "testitem" {
  hostid = zabbix_template.testtmpl.id
  key    = "grpsum[\"${zabbix_hostgroup.testgrp.name}\", \"not_real\", last]"

  name      = "Test Item Changed"
  valuetype = "float"

  preprocessor {
    type   = "1"
    params = [ "55" ]
  }

  preprocessor {
    type = "10"
  }
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "preprocessor.0.type", "1"),
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "preprocessor.0.params.0", "55"),
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "preprocessor.1.type", "10"),
				),
			},
			{ // preprocessor, javascript, =v5
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version < 50000 || api.Config.Version >= 50400, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [ zabbix_hostgroup.testgrp.id ]
  host   = %q
}
locals {
  script = <<-EOT
    var fish = false;
    return fish;
  EOT
}
resource "zabbix_item_aggregate" "testitem" {
  hostid = zabbix_template.testtmpl.id
  key    = "grpsum[\"${zabbix_hostgroup.testgrp.name}\", \"not_real\", last]"

  name      = "Test Item Changed"
  valuetype = "float"

  preprocessor {
    type          = "21"
    params        = [ "var bob = true;", "return bob;" ]
    error_handler = "0"
  }

  preprocessor {
    type          = "21"
    params        = split("\n", "var cheese = true;\nreturn cheese;")
    error_handler = "0"
  }

  preprocessor {
    type          = "21"
    params        = split("\n", trimspace(local.script))
    error_handler = "0"
  }
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "preprocessor.0.type", "21"),
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "preprocessor.0.params.0", "var bob = true;"),
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "preprocessor.0.params.1", "return bob;"),
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "preprocessor.1.type", "21"),
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "preprocessor.1.params.0", "var cheese = true;"),
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "preprocessor.1.params.1", "return cheese;"),
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "preprocessor.2.type", "21"),
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "preprocessor.2.params.0", "var fish = false;"),
					resource.TestCheckResourceAttr("zabbix_item_aggregate.testitem", "preprocessor.2.params.1", "return fish;"),
				),
			},
			// application, conditional only works on < 5.4
		},
	})
}
