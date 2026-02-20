package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceLLDExternal(t *testing.T) {
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

resource "zabbix_lld_external" "testrule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.external.discovery"
  name   = "LLD External Rule"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_external.testrule", "key", "lld.external.discovery"),
					resource.TestCheckResourceAttr("zabbix_lld_external.testrule", "name", "LLD External Rule"),
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

resource "zabbix_lld_external" "testrule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.external.discovery2"
  name   = "LLD External Rule A"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_external.testrule", "key", "lld.external.discovery2"),
					resource.TestCheckResourceAttr("zabbix_lld_external.testrule", "name", "LLD External Rule A"),
				),
			},
		},
	})
}
