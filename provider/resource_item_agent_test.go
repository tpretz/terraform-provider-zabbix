package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceItemAgent(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // simple create
				Config: hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template"
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitem"

	name = "Test Item"
	valuetype = "text"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "key", "testitem"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "name", "Test Item"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "valuetype", "text"),
				),
			},
			{ // change values
				Config: hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template"
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitemchanged"

	name = "Test Item Changed"
	valuetype = "float"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "key", "testitemchanged"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "name", "Test Item Changed"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "valuetype", "float"),
				),
			},
			{ // optionals
				Config: hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template"
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitemchanged"

	name = "Test Item Changed"
	valuetype = "float"

	active = true
	delay = "2m"
	history = "1h"
	trends = "7d"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "active", "true"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "delay", "2m"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "history", "1h"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "trends", "7d"),
				),
			},
			{ // optionals, with tags
				Config: hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template"
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitemchanged"

	name = "Test Item Changed"
	valuetype = "float"

	active = true
	delay = "2m"
	history = "1h"
	trends = "7d"

	tag {
		key = "action"
		value = "test"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "active", "true"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "delay", "2m"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "history", "1h"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "trends", "7d"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "tag.0.key", "action"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "tag.0.value", "test"),
				),
			},
			{ // attached to interface id
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_host.testhost.id
	key = "testitemchanged"
	interfaceid = one(zabbix_host.testhost.interface).id

	name = "Test Item Changed"
	valuetype = "float"

	active = false
	delay = "2m"
	history = "1h"
	trends = "7d"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "active", "false"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "delay", "2m"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "history", "1h"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "trends", "7d"),
					resource.TestCheckResourceAttrSet("zabbix_item_agent.testitem", "interfaceid"),
				),
			},
			{ // preprocessor, javascript
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
locals {
	script = <<-EOT
	  var fish = false;
	  return fish;
	EOT
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_host.testhost.id
	key = "testitemchanged"
	interfaceid = one(zabbix_host.testhost.interface).id

	name = "Test Item Changed"
	valuetype = "float"

	active = false
	delay = "2m"
	history = "1h"
	trends = "7d"

	preprocessor {
		type = "21"
		params = [ "var bob = true;", "return bob;" ]
		error_handler = "0"
	}
	
	preprocessor {
		type = "21"
		params = split("\n", "var cheese = true;\nreturn cheese;")
		error_handler = "0"
	}
	
	preprocessor {
		type = "21"
		params = split("\n", trimspace(local.script))
		# note: change schema to allow blank lines
		error_handler = "0"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "active", "false"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "delay", "2m"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "history", "1h"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "trends", "7d"),
					resource.TestCheckResourceAttrSet("zabbix_item_agent.testitem", "interfaceid"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.0.type", "21"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.0.params.0", "var bob = true;"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.0.params.1", "return bob;"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.1.type", "21"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.1.params.0", "var cheese = true;"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.1.params.1", "return cheese;"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.2.type", "21"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.2.params.0", "var fish = false;"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.2.params.1", "return fish;"),
				),
			},
			{ // preprocessor
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_host.testhost.id
	key = "testitemchanged"
	interfaceid = one(zabbix_host.testhost.interface).id

	name = "Test Item Changed"
	valuetype = "float"

	active = false
	delay = "2m"
	history = "1h"
	trends = "7d"

	preprocessor {
		type = "1"
		params = [ "55" ]
		error_handler = "0" # issue for version 4
	}
	
	preprocessor {
		type = "10"
		error_handler = "0"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "active", "false"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "delay", "2m"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "history", "1h"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "trends", "7d"),
					resource.TestCheckResourceAttrSet("zabbix_item_agent.testitem", "interfaceid"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.0.type", "1"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.0.params.0", "55"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.1.type", "10"),
				),
			},
			{ // C4 for a list: preprocessing order IS semantic, so a reorder must
				// produce a diff and the new order must survive the round trip.
				// Verified against live 6.0/7.0/7.4/8.0: item.get returns
				// preprocessing in submission order, and the objects carry no
				// sortorder field — position is the only thing conveying sequence.
				// Nothing else in the suite asserts this, so a future Zabbix that
				// reordered these would silently change what the steps compute.
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_host.testhost.id
	key = "testitemchanged"
	interfaceid = one(zabbix_host.testhost.interface).id

	name = "Test Item Changed"
	valuetype = "float"

	active = false
	delay = "2m"
	history = "1h"
	trends = "7d"

	preprocessor {
		type = "10"
		error_handler = "0"
	}

	preprocessor {
		type = "1"
		params = [ "55" ]
		error_handler = "0"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.0.type", "10"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.1.type", "1"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.1.params.0", "55"),
				),
			},
			{ // and the reordered state is stable, not flapping back
				Config: hcl(t, `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_host" "testhost" {
	host   = "test-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_host.testhost.id
	key = "testitemchanged"
	interfaceid = one(zabbix_host.testhost.interface).id

	name = "Test Item Changed"
	valuetype = "float"

	active = false
	delay = "2m"
	history = "1h"
	trends = "7d"

	preprocessor {
		type = "10"
		error_handler = "0"
	}

	preprocessor {
		type = "1"
		params = [ "55" ]
		error_handler = "0"
	}
}
`),
				PlanOnly: true,
			},
		},
	})
}
