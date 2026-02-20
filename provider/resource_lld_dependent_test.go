package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccResourceLLDDependent(t *testing.T) {
	id := resource.UniqueId()
	groupName := "test-group-" + id
	tmplHost := "test-template-" + id

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					// Zabbix < 5.4 rejects dependent LLD rules (type=18).
					return api.Config.Version < 50400, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [zabbix_hostgroup.testgrp.id]
  host   = %q
}

resource "zabbix_item_simple" "parent" {
  hostid = zabbix_template.testtmpl.id
  key = "script[\"abc\"]"

  name = "Parent Item"
  valuetype = "text"
}

resource "zabbix_lld_dependent" "testrule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.dependent.discovery"
  name   = "LLD Dependent Rule"
  master_itemid = zabbix_item_simple.parent.id
  delay = 0
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_dependent.testrule", "key", "lld.dependent.discovery"),
					resource.TestCheckResourceAttr("zabbix_lld_dependent.testrule", "name", "LLD Dependent Rule"),
					resource.TestCheckResourceAttrSet("zabbix_lld_dependent.testrule", "master_itemid"),
				),
			},
			{
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version < 50400, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [zabbix_hostgroup.testgrp.id]
  host   = %q
}

resource "zabbix_item_simple" "parent" {
  hostid = zabbix_template.testtmpl.id
  key = "script[\"abc\"]"

  name = "Parent Item"
  valuetype = "text"
}

resource "zabbix_lld_dependent" "testrule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.dependent.discovery2"
  name   = "LLD Dependent Rule A"
  master_itemid = zabbix_item_simple.parent.id
  delay = 0
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_dependent.testrule", "key", "lld.dependent.discovery2"),
					resource.TestCheckResourceAttr("zabbix_lld_dependent.testrule", "name", "LLD Dependent Rule A"),
				),
			},
		},
	})
}
