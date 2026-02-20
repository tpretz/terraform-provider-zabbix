package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceLLDAgent(t *testing.T) {
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

resource "zabbix_lld_agent" "testrule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.agent.discovery"
  name   = "LLD Agent Rule"
  active = false
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_agent.testrule", "key", "lld.agent.discovery"),
					resource.TestCheckResourceAttr("zabbix_lld_agent.testrule", "name", "LLD Agent Rule"),
					resource.TestCheckResourceAttr("zabbix_lld_agent.testrule", "active", "false"),
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

resource "zabbix_lld_agent" "testrule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.agent.discovery2"
  name   = "LLD Agent Rule A"
  active = true
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_agent.testrule", "key", "lld.agent.discovery2"),
					resource.TestCheckResourceAttr("zabbix_lld_agent.testrule", "name", "LLD Agent Rule A"),
					resource.TestCheckResourceAttr("zabbix_lld_agent.testrule", "active", "true"),
				),
			},
		},
	})
}
