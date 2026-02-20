package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceLLDSimple(t *testing.T) {
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
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [zabbix_hostgroup.testgrp.id]
  host   = %q
}

resource "zabbix_lld_simple" "testrule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.simple.discovery"
  name   = "LLD Simple Rule"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_simple.testrule", "key", "lld.simple.discovery"),
					resource.TestCheckResourceAttr("zabbix_lld_simple.testrule", "name", "LLD Simple Rule"),
					resource.TestCheckResourceAttrSet("zabbix_lld_simple.testrule", "hostid"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [zabbix_hostgroup.testgrp.id]
  host   = %q
}

resource "zabbix_lld_simple" "testrule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.simple.discovery2"
  name   = "LLD Simple Rule A"
  delay  = "600"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_simple.testrule", "key", "lld.simple.discovery2"),
					resource.TestCheckResourceAttr("zabbix_lld_simple.testrule", "name", "LLD Simple Rule A"),
					resource.TestCheckResourceAttr("zabbix_lld_simple.testrule", "delay", "600"),
				),
			},
		},
	})
}
