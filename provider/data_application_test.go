package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccDataApplication(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{ // lazy load config, needed for skipfunc that look at meta
				Config: `
resource "zabbix_hostgroup" "lazyconfigload" {
  name = "lazyload"
}
`,
			},
			{
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version >= 50400, nil
				},
				Config: `
resource "zabbix_hostgroup" "testgrp" {
  name = "test-group"
}
resource "zabbix_template" "testtmpl" {
  groups = [zabbix_hostgroup.testgrp.id]
  host   = "test-template"
}
resource "zabbix_application" "testapp" {
  name   = "test-app"
  hostid = zabbix_template.testtmpl.id
}

data "zabbix_application" "by_name" {
  name = zabbix_application.testapp.name
  hostid = zabbix_template.testtmpl.id
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.zabbix_application.by_name", "id"),
					resource.TestCheckResourceAttr("data.zabbix_application.by_name", "name", "test-app"),
				),
			},
		},
	})
}
