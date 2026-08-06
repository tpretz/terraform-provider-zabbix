package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceLLDHttp(t *testing.T) {
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
resource "zabbix_lld_http" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.http"

	name = "LLD HTTP Rule"
	url = "http://localhost/discovery"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "key", "lld.http"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "name", "LLD HTTP Rule"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "url", "http://localhost/discovery"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "request_method", "get"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "post_type", "raw"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "retrieve_mode", "body"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "auth_type", "none"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "delay", "3600"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "lifetime", "30d"),
				),
			},
			{ // modify: POST + headers + auth, and an LLD macro path / filter
				Config: hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template"
}
resource "zabbix_lld_http" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "lld.http.changed"

	name = "LLD HTTP Rule Renamed"
	url = "https://localhost/discovery/changed"
	request_method = "post"
	post_type = "json"
	posts = "{\"discover\": true}"
	auth_type = "basic"
	username = "discovery"
	password = "discoverypass"
	status_codes = "200,204"
	timeout = "20s"
	verify_peer = false
	follow_redirects = false

	headers = {
		"Accept" = "application/json"
	}

	delay = "10m"
	lifetime = "7d"

	macro {
		macro = "{#FSNAME}"
		path = "$.name"
	}

	condition {
		macro = "{#FSNAME}"
		value = "^/data"
		operator = "match"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "key", "lld.http.changed"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "name", "LLD HTTP Rule Renamed"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "url", "https://localhost/discovery/changed"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "request_method", "post"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "post_type", "json"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "auth_type", "basic"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "username", "discovery"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "status_codes", "200,204"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "timeout", "20s"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "verify_peer", "false"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "follow_redirects", "false"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "headers.Accept", "application/json"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "delay", "10m"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "lifetime", "7d"),
					resource.TestCheckResourceAttr("zabbix_lld_http.testlld", "macro.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_lld_http.testlld", "condition.*", map[string]string{
						"macro":    "{#FSNAME}",
						"value":    "^/data",
						"operator": "match",
					}),
				),
			},
			{ // import
				ResourceName:      "zabbix_lld_http.testlld",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
