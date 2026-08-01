package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
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
					resource.TestCheckResourceAttr("zabbix_host.testhost2", "inventory_mode", "manual"),
					resource.TestCheckResourceAttr("zabbix_host.testhost2", "inventory.0.location", "test location A"),
				),
			},
			{
				Config: testAccResourceHostWithInventoryUpdate(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost2", "inventory.0.location", "test location B"),
				),
			},
		},
	})
}

func TestAccHostWithTags(t *testing.T) {
	id := resource.UniqueId()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceHostWithTags(id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.tagstest", "tag.#", "2"),
				),
			},
		},
	})
}

func testAccResourceHostWithTags(id string) string {
	return fmt.Sprintf(`
resource "zabbix_hostgroup" "tagsgrp" {
  name = "test-tags-%s"
}
resource "zabbix_host" "tagstest" {
  host   = "test-tags-%s"
  groups = [zabbix_hostgroup.tagsgrp.id]
  interface {
    type = "agent"
    ip   = "127.0.0.1"
    port = 10050
    main = true
  }
  tag {
    key   = "env"
    value = "test"
  }
  tag {
    key   = "team"
    value = "platform"
  }
}
`, id, id)
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
	inventory_mode = "manual"
	inventory {
		location = "test location A"
	}
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
	inventory_mode = "manual"
	inventory {
		location = "test location B"
	}
}
`
}
