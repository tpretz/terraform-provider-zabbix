package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceProtoItemHttp(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // create
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_http" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "proto.http[{#PATH}]"

	name = "Proto HTTP {#PATH}"
	valuetype = "text"

	url = "http://localhost/{#PATH}"
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "key", "proto.http[{#PATH}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "name", "Proto HTTP {#PATH}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "url", "http://localhost/{#PATH}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "request_method", "get"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "retrieve_mode", "body"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "auth_type", "none"),
					resource.TestCheckResourceAttrPair(
						"zabbix_proto_item_http.testproto", "ruleid",
						"zabbix_lld_trapper.testlld", "id"),
				),
			},
			{ // update: post with headers and basic auth
				Config: protoItemConfig(t, `
resource "zabbix_proto_item_http" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "proto.http.changed[{#PATH}]"

	name = "Proto HTTP Changed {#PATH}"
	valuetype = "text"

	url = "http://localhost/changed/{#PATH}"
	request_method = "post"
	post_type = "json"
	posts = "{\"probe\": \"{#PATH}\"}"
	retrieve_mode = "both"
	auth_type = "basic"
	username = "probe"
	password = "secret"
	status_codes = "200,201"
	timeout = "10s"
	follow_redirects = false
	verify_host = false
	verify_peer = false

	headers = {
		"X-Probe" = "zabbix"
	}
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "key", "proto.http.changed[{#PATH}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "url", "http://localhost/changed/{#PATH}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "request_method", "post"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "post_type", "json"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "retrieve_mode", "both"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "auth_type", "basic"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "username", "probe"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "status_codes", "200,201"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "timeout", "10s"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "follow_redirects", "false"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "verify_host", "false"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "verify_peer", "false"),
					resource.TestCheckResourceAttr("zabbix_proto_item_http.testproto", "headers.X-Probe", "zabbix"),
				),
			},
			{ // import
				ResourceName:      "zabbix_proto_item_http.testproto",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
