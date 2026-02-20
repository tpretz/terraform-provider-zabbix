package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccResourceProtoTrigger(t *testing.T) {
	// Proto trigger expressions follow the same version-specific rules as regular triggers.
	id := resource.UniqueId()
	groupName := "test-group-" + id
	tmplHost := "test-template-" + id
	// Note: avoid embedding quotes in item keys; legacy trigger syntax uses {host:key.func()} which becomes
	// tricky to represent safely inside HCL strings.
	itemKey := "trapper.ping"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Zabbix >= 5.4 uses the "new" expression syntax.
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version < 50400, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}

resource "zabbix_template" "testtmpl" {
  groups = [zabbix_hostgroup.testgrp.id]
  host   = %q
}

resource "zabbix_lld_simple" "rule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.simple.discovery"
  name   = "LLD Simple Rule"
}

resource "zabbix_proto_item_trapper" "item" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = %q

  name      = "Proto Trapper Item"
  valuetype = "unsigned"
}

resource "zabbix_proto_trigger" "testtrg" {
  name       = "proto-trigger"
  expression = "last(/%s/%s)=0"
  priority   = "warn"
  enabled    = true

  depends_on = [zabbix_proto_item_trapper.item]
}
`, groupName, tmplHost, itemKey, tmplHost, itemKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrg", "name", "proto-trigger"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrg", "priority", "warn"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrg", "enabled", "true"),
				),
			},
			{
				// Zabbix < 5.4 legacy expression syntax.
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version >= 50400, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}

resource "zabbix_template" "testtmpl" {
  groups = [zabbix_hostgroup.testgrp.id]
  host   = %q
}

resource "zabbix_lld_simple" "rule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.simple.discovery"
  name   = "LLD Simple Rule"
}

resource "zabbix_proto_item_trapper" "item" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = %q

  name      = "Proto Trapper Item"
  valuetype = "unsigned"
}

resource "zabbix_proto_trigger" "testtrg" {
  name       = "proto-trigger"
  expression = "{%s:%s.last()}=0"
  priority   = "warn"
  enabled    = true

  depends_on = [zabbix_proto_item_trapper.item]
}
`, groupName, tmplHost, itemKey, tmplHost, itemKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrg", "name", "proto-trigger"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrg", "priority", "warn"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrg", "enabled", "true"),
				),
			},
			{
				// Zabbix >= 5.4 update
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version < 50400, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}

resource "zabbix_template" "testtmpl" {
  groups = [zabbix_hostgroup.testgrp.id]
  host   = %q
}

resource "zabbix_lld_simple" "rule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.simple.discovery"
  name   = "LLD Simple Rule"
}

resource "zabbix_proto_item_trapper" "item" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = %q

  name      = "Proto Trapper Item"
  valuetype = "unsigned"
}

resource "zabbix_proto_trigger" "testtrg" {
  name       = "proto-trigger-a"
  expression = "last(/%s/%s)=1"
  priority   = "high"
  enabled    = false

  depends_on = [zabbix_proto_item_trapper.item]
}
`, groupName, tmplHost, itemKey, tmplHost, itemKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrg", "name", "proto-trigger-a"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrg", "priority", "high"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrg", "enabled", "false"),
				),
			},
			{
				// Zabbix < 5.4 legacy update
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version >= 50400, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}

resource "zabbix_template" "testtmpl" {
  groups = [zabbix_hostgroup.testgrp.id]
  host   = %q
}

resource "zabbix_lld_simple" "rule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.simple.discovery"
  name   = "LLD Simple Rule"
}

resource "zabbix_proto_item_trapper" "item" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = %q

  name      = "Proto Trapper Item"
  valuetype = "unsigned"
}

resource "zabbix_proto_trigger" "testtrg" {
  name       = "proto-trigger-a"
  expression = "{%s:%s.last()}=1"
  priority   = "high"
  enabled    = false

  depends_on = [zabbix_proto_item_trapper.item]
}
`, groupName, tmplHost, itemKey, tmplHost, itemKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrg", "name", "proto-trigger-a"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrg", "priority", "high"),
					resource.TestCheckResourceAttr("zabbix_proto_trigger.testtrg", "enabled", "false"),
				),
			},
		},
	})
}
