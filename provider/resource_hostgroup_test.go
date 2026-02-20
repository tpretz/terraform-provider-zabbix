package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceHostgroup(t *testing.T) {
	id := resource.UniqueId()
	groupName := "test-group-" + id
	groupNameRenamed := groupName + "-renamed"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{ // simple create
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
`, groupName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_hostgroup.testgrp", "name", groupName),
				),
			},
			{ // rename
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
`, groupNameRenamed),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_hostgroup.testgrp", "name", groupNameRenamed),
				),
			},
		},
	})
}
