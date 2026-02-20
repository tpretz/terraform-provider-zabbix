package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceProtoItemHttp(t *testing.T) {
	id := resource.UniqueId()
	groupName := "test-group-" + id
	tmplHost := "test-template-" + id

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
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

resource "zabbix_proto_item_http" "testitem" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id

  key       = "http.agent.key"
  name      = "Proto HTTP Item"
  valuetype = "text"

  url    = "http://example.com"
  delay  = "1m"
  timeout = "3s"

  request_method   = "get"
  retrieve_mode    = "body"
  follow_redirects = true
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("zabbix_proto_item_http.testitem", "ruleid"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testitem", "key", "http.agent.key"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testitem", "url", "http://example.com"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testitem", "request_method", "get"),
				),
			},
			{
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

resource "zabbix_proto_item_http" "testitem" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id

  key       = "http.agent.key2"
  name      = "Proto HTTP Item A"
  valuetype = "text"

  url    = "http://example.org"
  delay  = "30s"
  timeout = "5s"

  request_method   = "post"
  retrieve_mode    = "headers"
  follow_redirects = false
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testitem", "key", "http.agent.key2"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testitem", "url", "http://example.org"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testitem", "request_method", "post"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testitem", "retrieve_mode", "headers"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testitem", "follow_redirects", "false"),
				),
			},
		},
	})
}
