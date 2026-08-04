package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// The proxy object was rewritten in Zabbix 7.0 -- "host" became "name",
// "status" became "operating_mode", the nested interface became
// "address"/"port" and "proxy_address" became "allowed_addresses". The
// provider hides all of that behind one set of attribute names, so these
// tests are deliberately version-neutral: the same configuration and the same
// assertions must hold on 6.0 and on 7.4 alike. A SkipFunc here would mean the
// translation had leaked.

func TestAccResourceProxyActive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // minimal create -- an active proxy needs nothing but a name
				Config: `
resource "zabbix_proxy" "testproxy" {
	name = "test-proxy-active"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "name", "test-proxy-active"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "operating_mode", "active"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "allowed_addresses", ""),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "description", ""),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "tls_connect", "unencrypted"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "tls_accept", "unencrypted"),
					// an active proxy has no endpoint of its own: both
					// versions report the Zabbix defaults back
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "address", "127.0.0.1"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "port", "10051"),
				),
			},
			{ // rename, describe, restrict where it may connect from
				Config: `
resource "zabbix_proxy" "testproxy" {
	name              = "test-proxy-active-renamed"
	description       = "managed by terraform"
	allowed_addresses = "10.0.0.1,10.0.0.2"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "name", "test-proxy-active-renamed"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "description", "managed by terraform"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "allowed_addresses", "10.0.0.1,10.0.0.2"),
				),
			},
			{ // import round trip
				ResourceName:      "zabbix_proxy.testproxy",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{ // clear the description again, and require PSK encryption
				Config: `
resource "zabbix_proxy" "testproxy" {
	name              = "test-proxy-active-renamed"
	allowed_addresses = "10.0.0.1,10.0.0.2"
	tls_accept        = "psk"
	tls_psk_identity  = "test-proxy-psk"
	tls_psk           = "b7e3b0e9a94e4f2a8d1c6f5b3a2e9d8c7b6a5f4e3d2c1b0a9f8e7d6c5b4a3f2e"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "description", ""),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "tls_accept", "psk"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "tls_connect", "unencrypted"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "tls_psk_identity", "test-proxy-psk"),
				),
			},
			{ // the PSK survives an import only as far as the API allows:
				// tls_psk_identity/tls_psk are write-only and never returned
				ResourceName:            "zabbix_proxy.testproxy",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"tls_psk_identity", "tls_psk"},
			},
		},
	})
}

func TestAccResourceProxyPassive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // create passive, addressed by IP
				Config: `
resource "zabbix_proxy" "testproxy" {
	name           = "test-proxy-passive"
	operating_mode = "passive"
	address        = "10.20.30.40"
	port           = "10051"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "operating_mode", "passive"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "address", "10.20.30.40"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "port", "10051"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "allowed_addresses", ""),
				),
			},
			{ // move it to a DNS name and a non-default port. Before 7.0 this
				// flips useip on the nested interface object; from 7.0 it is
				// the same "address" property either way.
				Config: `
resource "zabbix_proxy" "testproxy" {
	name             = "test-proxy-passive"
	operating_mode   = "passive"
	address          = "proxy.test.invalid"
	port             = "10555"
	description      = "passive proxy"
	tls_connect      = "psk"
	tls_psk_identity = "test-proxy-psk"
	tls_psk          = "b7e3b0e9a94e4f2a8d1c6f5b3a2e9d8c7b6a5f4e3d2c1b0a9f8e7d6c5b4a3f2e"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "address", "proxy.test.invalid"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "port", "10555"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "description", "passive proxy"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "tls_connect", "psk"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "tls_accept", "unencrypted"),
				),
			},
			{ // import round trip
				ResourceName:            "zabbix_proxy.testproxy",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"tls_psk_identity", "tls_psk"},
			},
			{ // switch it to active: the endpoint has to disappear, on both
				// the pre-7.0 model (the interface object is deleted) and the
				// 7.0 one (address/port revert to their fixed defaults)
				Config: `
resource "zabbix_proxy" "testproxy" {
	name              = "test-proxy-passive"
	operating_mode    = "active"
	description       = "passive proxy"
	allowed_addresses = "192.0.2.1"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "operating_mode", "active"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "address", "127.0.0.1"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "port", "10051"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "allowed_addresses", "192.0.2.1"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "tls_connect", "unencrypted"),
				),
			},
			{ // and back to passive again, to prove the round trip both ways
				Config: `
resource "zabbix_proxy" "testproxy" {
	name           = "test-proxy-passive"
	operating_mode = "passive"
	address        = "10.20.30.40"
	description    = "passive proxy"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "operating_mode", "passive"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "address", "10.20.30.40"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "port", "10051"),
					resource.TestCheckResourceAttr("zabbix_proxy.testproxy", "allowed_addresses", ""),
				),
			},
		},
	})
}

// TestAccResourceProxyModeMismatch checks that the attributes which only apply
// to one operating mode are rejected on the other with a message naming the
// mode, rather than being silently dropped -- which would leave a permanent
// diff, since Zabbix would report the value it kept, not the one asked for.
func TestAccResourceProxyModeMismatch(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: `
resource "zabbix_proxy" "testproxy" {
	name    = "test-proxy-mismatch"
	address = "10.20.30.40"
}
`,
				ExpectError: regexp.MustCompile("address applies to passive proxies only"),
			},
			{
				Config: `
resource "zabbix_proxy" "testproxy" {
	name              = "test-proxy-mismatch"
	operating_mode    = "passive"
	address           = "10.20.30.40"
	allowed_addresses = "192.0.2.1"
}
`,
				ExpectError: regexp.MustCompile("allowed_addresses applies to active proxies only"),
			},
			{ // Zabbix itself rejects encryption configured on the side that
				// does not apply; the provider sends both values so the server
				// has the final word rather than the provider dropping one
				Config: `
resource "zabbix_proxy" "testproxy" {
	name             = "test-proxy-mismatch"
	tls_connect      = "psk"
	tls_psk_identity = "test-proxy-psk"
	tls_psk          = "b7e3b0e9a94e4f2a8d1c6f5b3a2e9d8c7b6a5f4e3d2c1b0a9f8e7d6c5b4a3f2e"
}
`,
				ExpectError: regexp.MustCompile(`tls_connect`),
			},
		},
	})
}

// TestAccResourceProxyHost covers the other half of the 7.0 proxy rework: a
// host pointing at a proxy is "proxy_hostid" before 7.0 and
// "proxyid" plus "monitored_by" from 7.0.
func TestAccResourceProxyHost(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: `
resource "zabbix_proxy" "testproxy" {
	name = "test-proxy-forhost"
}
resource "zabbix_hostgroup" "testgrp" {
	name = "test-proxy-hostgroup"
}
resource "zabbix_host" "testhost" {
	host    = "test-proxy-host"
	groups  = [zabbix_hostgroup.testgrp.id]
	proxyid = zabbix_proxy.testproxy.id
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"zabbix_host.testhost", "proxyid",
						"zabbix_proxy.testproxy", "id"),
				),
			},
			{ // hand the host back to the server
				Config: `
resource "zabbix_proxy" "testproxy" {
	name = "test-proxy-forhost"
}
resource "zabbix_hostgroup" "testgrp" {
	name = "test-proxy-hostgroup"
}
resource "zabbix_host" "testhost" {
	host   = "test-proxy-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testhost", "proxyid", "0"),
				),
			},
		},
	})
}

func TestAccDataSourceProxy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: `
resource "zabbix_proxy" "testproxy" {
	name              = "test-proxy-data"
	description       = "looked up by the data source"
	allowed_addresses = "192.0.2.1"
}
data "zabbix_proxy" "byname" {
	name = zabbix_proxy.testproxy.name
}
data "zabbix_proxy" "byhost" {
	host = zabbix_proxy.testproxy.name
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.zabbix_proxy.byname", "id",
						"zabbix_proxy.testproxy", "id"),
					resource.TestCheckResourceAttr("data.zabbix_proxy.byname", "name", "test-proxy-data"),
					resource.TestCheckResourceAttr("data.zabbix_proxy.byname", "host", "test-proxy-data"),
					resource.TestCheckResourceAttr("data.zabbix_proxy.byname", "operating_mode", "active"),
					resource.TestCheckResourceAttr("data.zabbix_proxy.byname", "description", "looked up by the data source"),
					resource.TestCheckResourceAttr("data.zabbix_proxy.byname", "allowed_addresses", "192.0.2.1"),
					resource.TestCheckResourceAttr("data.zabbix_proxy.byname", "tls_accept", "unencrypted"),
					// the deprecated pre-7.0 argument still resolves the same proxy
					resource.TestCheckResourceAttrPair(
						"data.zabbix_proxy.byhost", "id",
						"zabbix_proxy.testproxy", "id"),
					resource.TestCheckResourceAttr("data.zabbix_proxy.byhost", "name", "test-proxy-data"),
				),
			},
		},
	})
}
