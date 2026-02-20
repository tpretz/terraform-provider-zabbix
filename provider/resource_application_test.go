package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccResourceApplication(t *testing.T) {
	id := resource.UniqueId()
	lazyGroup := "lazyload-" + id
	groupName := "test-group-" + id
	tmplHost := "test-template-" + id
	appName := "test-app-" + id

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{ // lazy load config, needed for SkipFunc that looks at meta
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "lazyconfigload" {
  name = %q
}
`, lazyGroup),
			},
			{ // simple create
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
resource "zabbix_application" "testapp" {
  name   = %q
  hostid = zabbix_template.testtmpl.id
}
`, groupName, tmplHost, appName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_application.testapp", "name", appName),
				),
			},
			{ // rename
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
resource "zabbix_application" "testapp" {
  name   = %q
  hostid = zabbix_template.testtmpl.id
}
`, groupName, tmplHost, appName+"-renamed"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_application.testapp", "name", appName+"-renamed"),
				),
			},
			// TODO: ref to an item
		},
	})
}
