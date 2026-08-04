package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceLLDAgent(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // lazy init
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
`),
			},
			{ // simple create
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_lld_agent" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.agent[\"abc\"]"

	name = "LLD Agent Rule"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_agent.testlld", "key", "lld.agent[\"abc\"]"),
					resource.TestCheckResourceAttr("zabbix_lld_agent.testlld", "name", "LLD Agent Rule"),
					resource.TestCheckResourceAttr("zabbix_lld_agent.testlld", "active", "false"),
					resource.TestCheckResourceAttr("zabbix_lld_agent.testlld", "delay", "3600"),
					resource.TestCheckResourceAttr("zabbix_lld_agent.testlld", "lifetime", "30d"),
				),
			},
			{ // modify: rename, switch to active agent, tweak delay/lifetime
				Config: hcl(t, `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_lld_agent" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.agent[\"abc\"]"

	name = "LLD Agent Rule Renamed"
	active = true
	delay = "60s"
	lifetime = "7d"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_agent.testlld", "name", "LLD Agent Rule Renamed"),
					resource.TestCheckResourceAttr("zabbix_lld_agent.testlld", "active", "true"),
					resource.TestCheckResourceAttr("zabbix_lld_agent.testlld", "delay", "60s"),
					resource.TestCheckResourceAttr("zabbix_lld_agent.testlld", "lifetime", "7d"),
				),
			},
			{ // import
				ResourceName:      "zabbix_lld_agent.testlld",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
