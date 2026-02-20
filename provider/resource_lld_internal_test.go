package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceLLDInternal(t *testing.T) {
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

resource "zabbix_lld_internal" "testrule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.internal.discovery"
  name   = "LLD Internal Rule"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_internal.testrule", "key", "lld.internal.discovery"),
					resource.TestCheckResourceAttr("zabbix_lld_internal.testrule", "name", "LLD Internal Rule"),
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

resource "zabbix_lld_internal" "testrule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.internal.discovery2"
  name   = "LLD Internal Rule A"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_internal.testrule", "key", "lld.internal.discovery2"),
					resource.TestCheckResourceAttr("zabbix_lld_internal.testrule", "name", "LLD Internal Rule A"),
				),
			},
		},
	})
}
