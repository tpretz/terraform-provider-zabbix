package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform/helper/acctest"
)

func TestAccResourceHost(t *testing.T) {
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceHostBasic(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "host", fmt.Sprintf("test-host-%s", rName)),
				),
			},
			{
				Config: testAccResourceHostWithInventory(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost2", "inventory.0.location", "test location A"),
				),
			},
			{
				Config: testAccResourceHostWithInventoryUpdate(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost2", "inventory.0.location", "test location B"),
				),
			},
			{
				Config: testAccResourceHostMultiIface1(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost3", "macro.0.value", "fish"),
					resource.TestCheckResourceAttr("zabbix_host.testhost3", "interface.1.type", "jmx"),
				),
			},
		},
	})
}

func testAccResourceHostBasic(rHost string) string {
	return fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group-%s" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host-%s"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
	}
}
`, rHost, rHost)
}

func testAccResourceHostWithInventory(rHost string) string {
	return fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp2" {
	name = "test-group2-%s" 
}
resource "zabbix_host" "testhost2" {
	host   = "test-host2-%s"
	groups = [zabbix_hostgroup.testgrp2.id]
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
	}
	inventory_mode = "manual"
    inventory {
		location = "test location A"
	}
}
`, rHost, rHost)
}

func testAccResourceHostWithInventoryUpdate(rHost string) string {
	return fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp2" {
	name = "test-group2-%s" 
}
resource "zabbix_host" "testhost2" {
	host   = "test-host2-%s"
	groups = [zabbix_hostgroup.testgrp2.id]
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
	}
	inventory_mode = "manual"
    inventory {
		location = "test location B"
	}
}
`, rHost, rHost)
}

func testAccResourceHostMultiIface1(rHost string) string {
	return fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group3-%s" 
}
resource "zabbix_host" "testhost3" {
	host   = "test-host3-%s"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		dns = "localhost"
		port = 1234
	}

	interface {
		dns = "bob"
		type = "jmx"
	}

	macro {
		value = "fish"
		name = "{$BOB}"
	}
}
`, rHost, rHost)
}
