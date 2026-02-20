package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccResourceTrigger(t *testing.T) {
	// Zabbix trigger expression syntax differs across major versions.
	// - v6+:   last(/host/key)=0
	// - v4/v5: {host:key.last()}=0
	// Avoid item keys with quoted parameters inside expressions (e.g. script["abc"]) as Zabbix may reject them.
	id := resource.UniqueId()
	groupName := "test-group-" + id
	hostName := "test-host-" + id

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Zabbix >= 5.4 expression syntax
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version < 50400, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}

resource "zabbix_host" "testhost" {
  host = %q
  groups = [zabbix_hostgroup.testgrp.id]

  interface {
    type = "agent"
    dns  = "localhost"
    port = 10050
  }
}

resource "zabbix_item_trapper" "testitem" {
  hostid = zabbix_host.testhost.id
  key = "trapper.ping"

  name = "Trapper Item"
  valuetype = "unsigned"
}

resource "zabbix_trigger" "testtrg" {
  name = "test-trigger"
  expression = "last(/%s/trapper.ping)=0"
  priority = "warn"
  enabled = true

  depends_on = [zabbix_item_trapper.testitem]
}
`, groupName, hostName, hostName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.testtrg", "name", "test-trigger"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrg", "priority", "warn"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrg", "enabled", "true"),
				),
			},
			{
				// Zabbix < 5.4 legacy expression syntax
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version >= 50400, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}

resource "zabbix_host" "testhost" {
  host = %q
  groups = [zabbix_hostgroup.testgrp.id]

  interface {
    type = "agent"
    dns  = "localhost"
    port = 10050
  }
}

resource "zabbix_item_trapper" "testitem" {
  hostid = zabbix_host.testhost.id
  key = "trapper.ping"

  name = "Trapper Item"
  valuetype = "unsigned"
}

resource "zabbix_trigger" "testtrg" {
  name = "test-trigger"
  expression = "{%s:trapper.ping.last()}=0"
  priority = "warn"
  enabled = true

  depends_on = [zabbix_item_trapper.testitem]
}
`, groupName, hostName, hostName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.testtrg", "name", "test-trigger"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrg", "priority", "warn"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrg", "enabled", "true"),
				),
			},
			{
				// Zabbix >= 5.4 expression syntax (update)
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version < 50400, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}

resource "zabbix_host" "testhost" {
  host = %q
  groups = [zabbix_hostgroup.testgrp.id]

  interface {
    type = "agent"
    dns  = "localhost"
    port = 10050
  }
}

resource "zabbix_item_trapper" "testitem" {
  hostid = zabbix_host.testhost.id
  key = "trapper.ping"

  name = "Trapper Item"
  valuetype = "unsigned"
}

resource "zabbix_trigger" "testtrg" {
  name = "test-trigger-a"
  expression = "last(/%s/trapper.ping)=1"
  priority = "high"
  enabled = false

  depends_on = [zabbix_item_trapper.testitem]
}
`, groupName, hostName, hostName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.testtrg", "name", "test-trigger-a"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrg", "priority", "high"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrg", "enabled", "false"),
				),
			},
			{
				// Zabbix < 5.4 legacy expression syntax (update)
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version >= 50400, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}

resource "zabbix_host" "testhost" {
  host = %q
  groups = [zabbix_hostgroup.testgrp.id]

  interface {
    type = "agent"
    dns  = "localhost"
    port = 10050
  }
}

resource "zabbix_item_trapper" "testitem" {
  hostid = zabbix_host.testhost.id
  key = "trapper.ping"

  name = "Trapper Item"
  valuetype = "unsigned"
}

resource "zabbix_trigger" "testtrg" {
  name = "test-trigger-a"
  expression = "{%s:trapper.ping.last()}=1"
  priority = "high"
  enabled = false

  depends_on = [zabbix_item_trapper.testitem]
}
`, groupName, hostName, hostName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.testtrg", "name", "test-trigger-a"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrg", "priority", "high"),
					resource.TestCheckResourceAttr("zabbix_trigger.testtrg", "enabled", "false"),
				),
			},
		},
	})
}
