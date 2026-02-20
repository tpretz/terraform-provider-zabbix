package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceItemHttp(t *testing.T) {
	id := resource.UniqueId()
	groupName := "test-group-" + id
	tmplHost := "test-template-" + id

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
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

resource "zabbix_item_http" "testitem" {
  hostid   = zabbix_template.testtmpl.id
  key      = "web.page.get[http://example.com]"
  name     = "HTTP Item"
  valuetype = "text"

  url            = "http://example.com"
  request_method = "get"
  retrieve_mode  = "body"
  status_codes   = "200"
  timeout        = "3s"
  verify_host    = false
  verify_peer    = false
  follow_redirects = true

  headers = {
    "User-Agent" = "terraform-provider-zabbix"
  }
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "key", "web.page.get[http://example.com]"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "name", "HTTP Item"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "url", "http://example.com"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "request_method", "get"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "retrieve_mode", "body"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "status_codes", "200"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "timeout", "3s"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "verify_host", "false"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "verify_peer", "false"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "follow_redirects", "true"),
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

resource "zabbix_item_http" "testitem" {
  hostid   = zabbix_template.testtmpl.id
  key      = "web.page.get2[http://example.com]"
  name     = "HTTP Item A"
  valuetype = "text"

  url            = "http://example.com/abc"
  request_method = "head"
  retrieve_mode  = "headers"
  status_codes   = "200,301"
  timeout        = "5s"
  verify_host    = true
  verify_peer    = true
  follow_redirects = false
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "key", "web.page.get2[http://example.com]"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "name", "HTTP Item A"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "url", "http://example.com/abc"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "request_method", "head"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "retrieve_mode", "headers"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "status_codes", "200,301"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "timeout", "5s"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "verify_host", "true"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "verify_peer", "true"),
					resource.TestCheckResourceAttr("zabbix_item_http.testitem", "follow_redirects", "false"),
				),
			},
		},
	})
}
