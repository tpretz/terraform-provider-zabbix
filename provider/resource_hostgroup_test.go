package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceHostgroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{ // simple create
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_hostgroup.testgrp", "name", "test-group"),
				),
			},
			{ // rename
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group-renamed" 
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_hostgroup.testgrp", "name", "test-group-renamed"),
				),
			},
		},
	})
}
