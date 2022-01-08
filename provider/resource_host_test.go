package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestAccResourceHost(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceHostBasic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "host", "test-host"),
				),
			},
			{
				Config: testAccResourceHostWithInventory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost2", "inventory_location", "test location A"),
				),
			},
			{
				Config: testAccResourceHostWithInventoryUpdate(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost2", "inventory_location", "test location B"),
				),
			},
		},
	})
}

func testAccResourceHostBasic() string {
	return `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
	}
}
`
}

func testAccResourceHostWithInventory() string {
	return `
resource "zabbix_hostgroup" "testgrp2" {
	name = "test-group2" 
}
resource "zabbix_host" "testhost2" {
	host   = "test-host2"
	groups = [zabbix_hostgroup.testgrp2.id]
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
	}
    inventory_location = "test location A"
}
`
}

func testAccResourceHostWithInventoryUpdate() string {
	return `
resource "zabbix_hostgroup" "testgrp2" {
	name = "test-group2" 
}
resource "zabbix_host" "testhost2" {
	host   = "test-host2"
	groups = [zabbix_hostgroup.testgrp2.id]
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
	}
    inventory_location = "test location B"
}
`
}
