package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceItemHttp(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // simple create, everything on defaults
				Config: hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template"
}
resource "zabbix_item_http" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitem.http"

	name = "Test HTTP Item"
	valuetype = "text"

	url = "http://localhost/probe"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "key", "testitem.http"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "name", "Test HTTP Item"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "valuetype", "text"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "url", "http://localhost/probe"),
					// schema defaults, read back from the server
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "request_method", "get"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "post_type", "raw"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "retrieve_mode", "body"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "auth_type", "none"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "status_codes", "200"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "timeout", "3s"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "follow_redirects", "true"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "verify_host", "true"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "verify_peer", "true"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "delay", "1m"),
				),
			},
			{ // change values: POST with a JSON body, headers, auth, TLS off
				Config: hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template"
}
resource "zabbix_item_http" "testitem" {
	hostid = zabbix_template.testtmpl.id
	key = "testitem.http.changed"

	name = "Test HTTP Item Changed"
	valuetype = "log"

	url = "https://localhost/probe/changed"
	request_method = "post"
	post_type = "json"
	posts = "{\"hello\": \"world\"}"
	retrieve_mode = "headers"
	auth_type = "ntlm"
	username = "probeuser"
	password = "probepass"
	proxy = "http://proxy.example.com:3128"
	status_codes = "200-299"
	timeout = "15s"
	delay = "4m"
	follow_redirects = false
	verify_host = false
	verify_peer = false

	headers = {
		"Accept" = "application/json"
		"X-Probe" = "zabbix"
	}

	tag {
		key = "component"
		value = "http"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "key", "testitem.http.changed"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "name", "Test HTTP Item Changed"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "valuetype", "log"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "url", "https://localhost/probe/changed"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "request_method", "post"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "post_type", "json"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "posts", "{\"hello\": \"world\"}"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "retrieve_mode", "headers"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "auth_type", "ntlm"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "username", "probeuser"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "password", "probepass"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "proxy", "http://proxy.example.com:3128"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "status_codes", "200-299"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "timeout", "15s"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "delay", "4m"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "follow_redirects", "false"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "verify_host", "false"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "verify_peer", "false"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "headers.Accept", "application/json"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "headers.X-Probe", "zabbix"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "tag.0.key", "component"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "tag.0.value", "http"),
				),
			},
			{ // attached to a host interface rather than a template
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
resource "zabbix_item_http" "testitem" {
	hostid = zabbix_host.testhost.id
	interfaceid = one(zabbix_host.testhost.interface).id
	key = "testitem.http.changed"

	name = "Test HTTP Item Changed"
	valuetype = "text"

	url = "http://localhost/probe"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("zabbix_item_http.testitem", "interfaceid"),
					resource.TestCheckResourceAttrPair(
						"zabbix_item_http.testitem", "hostid",
						"zabbix_host.testhost", "id"),
				),
			},
			{ // import
				ResourceName:      "zabbix_item_http.testitem",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
