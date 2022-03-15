package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccResourceItemAgent(t *testing.T) {
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
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_hostgroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitem"

	name = "Test Item"
	valuetype = "text"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "key", "testitem"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "name", "Test Item"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "valuetype", "text"),
				),
			},
			{ // change values
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_hostgroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitemchanged"

	name = "Test Item Changed"
	valuetype = "float"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "key", "testitemchanged"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "name", "Test Item Changed"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "valuetype", "float"),
				),
			},
			{ // optionals, >=v5
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version < 50000, nil
				},
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_hostgroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitemchanged"

	name = "Test Item Changed"
	valuetype = "float"

	active = true
	#applications = [zabbix_application.testapp.id]
	delay = "2m"
	history = "1h"
	trends = "7d"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "active", "true"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "delay", "2m"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "history", "1h"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "trends", "7d"),
				),
			},
			{ // optionals, with tags >=v5.4
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version < 50400, nil
				},
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-group" 
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_hostgroup.testgrp.id ]
	host = "test-template"
}
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitemchanged"

	name = "Test Item Changed"
	valuetype = "float"

	active = true
	#applications = [zabbix_application.testapp.id]
	delay = "2m"
	history = "1h"
	trends = "7d"

	tag {
		key = "action"
		value = "test"
	}
}
`,
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
				Config: `
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
	interfaceid = zabbix_host.testhost.interface.0.id

	name = "Test Item Changed"
	valuetype = "float"

	active = false
	#applications = [zabbix_application.testapp.id]
	delay = "2m"
	history = "1h"
	trends = "7d"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "active", "false"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "delay", "2m"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "history", "1h"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "trends", "7d"),
					resource.TestCheckResourceAttrSet("zabbix_item_agent.testitem", "interfaceid"),
				),
			},
			{ // preprocessor, <v5
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version >= 50000, nil
				},
				Config: `
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
	interfaceid = zabbix_host.testhost.interface.0.id

	name = "Test Item Changed"
	valuetype = "float"

	active = false
	#applications = [zabbix_application.testapp.id]
	delay = "2m"
	history = "1h"
	trends = "7d"

	preprocessor {
		type = "1"
		params = [ "55" ]
	}
	
	preprocessor {
		type = "10"
	}
}
`,
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
			{ // preprocessor, javascript, >=v5
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version < 50000, nil
				},
				Config: `
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
	interfaceid = zabbix_host.testhost.interface.0.id

	name = "Test Item Changed"
	valuetype = "float"

	active = false
	#applications = [zabbix_application.testapp.id]
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
`,
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
			{ // preprocessor, >=v5
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version < 50000, nil
				},
				Config: `
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
	interfaceid = zabbix_host.testhost.interface.0.id

	name = "Test Item Changed"
	valuetype = "float"

	active = false
	#applications = [zabbix_application.testapp.id]
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
`,
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
			{ // preprocessor, >=v5.4
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version < 50400, nil
				},
				Config: `
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
	interfaceid = zabbix_host.testhost.interface.0.id

	name = "Test Item Changed"
	valuetype = "float"

	active = false
	#applications = [zabbix_application.testapp.id]
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
	
	preprocessor {
		type = "26"
		error_handler = "1"
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "active", "false"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "delay", "2m"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "history", "1h"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "trends", "7d"),
					resource.TestCheckResourceAttrSet("zabbix_item_agent.testitem", "interfaceid"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.0.type", "1"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.0.params.0", "55"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.1.type", "10"),
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "preprocessor.2.type", "26"),
				),
			},
			// preprocessor
			// application, conditional only works on < 5.4
		},
	})
}
