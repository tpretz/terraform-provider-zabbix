package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccHTTPTest(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHTTPTestDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHTTPTestConfig(id, "first"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_httptest.test", "name", "acc-web-"+id+"-first"),
					resource.TestCheckResourceAttr("zabbix_httptest.test", "hostid", "10084"),
					resource.TestCheckResourceAttr("zabbix_httptest.test", "delay", "60"),
					resource.TestCheckResourceAttr("zabbix_httptest.test", "retries", "1"),
					resource.TestCheckResourceAttr("zabbix_httptest.test", "status", "enabled"),
					resource.TestCheckResourceAttr("zabbix_httptest.test", "steps.#", "1"),
					resource.TestCheckResourceAttr("zabbix_httptest.test", "steps.0.name", "Homepage"),
					resource.TestCheckResourceAttr("zabbix_httptest.test", "steps.0.url", "http://example.com"),
					resource.TestCheckResourceAttr("zabbix_httptest.test", "steps.0.no", "1"),
				),
			},
			{
				// Update: change name, delay, add second step, headers
				Config: testAccHTTPTestConfig(id, "second"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_httptest.test", "name", "acc-web-"+id+"-second"),
					resource.TestCheckResourceAttr("zabbix_httptest.test", "delay", "120"),
					resource.TestCheckResourceAttr("zabbix_httptest.test", "retries", "2"),
					resource.TestCheckResourceAttr("zabbix_httptest.test", "agent", "CustomAgent/1.0"),
					resource.TestCheckResourceAttr("zabbix_httptest.test", "status", "enabled"),
					resource.TestCheckResourceAttr("zabbix_httptest.test", "steps.#", "2"),
					resource.TestCheckResourceAttr("zabbix_httptest.test", "steps.0.name", "Home"),
					resource.TestCheckResourceAttr("zabbix_httptest.test", "steps.1.name", "About"),
					resource.TestCheckResourceAttr("zabbix_httptest.test", "headers.#", "1"),
				),
			},
		},
	})
}

func testAccHTTPTestConfig(id, variant string) string {
	if variant == "first" {
		return fmt.Sprintf(`
resource "zabbix_httptest" "test" {
  name   = "acc-web-%s-first"
  hostid = "10084"
  delay  = "60"
  retries = "1"
  status = "enabled"

  steps {
    name         = "Homepage"
    url          = "http://example.com"
    no           = "1"
    timeout      = "15"
    status_codes = "200"
  }
}
`, id)
	}
	return fmt.Sprintf(`
resource "zabbix_httptest" "test" {
  name    = "acc-web-%s-second"
  hostid  = "10084"
  delay   = "120"
  retries = "2"
  agent   = "CustomAgent/1.0"
  status  = "enabled"

  headers {
    name  = "X-Custom"
    value = "headerval"
  }

  steps {
    name         = "Home"
    url          = "http://example.com"
    no           = "1"
    timeout      = "15"
    status_codes = "200"
  }

  steps {
    name         = "About"
    url          = "http://example.com/about"
    no           = "2"
    timeout      = "15"
    status_codes = "200"
  }
}
`, id)
}

func testAccCheckHTTPTestDestroy(s *terraform.State) error {
	api := testAccProvider.Meta().(*zabbix.API)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "zabbix_httptest" {
			continue
		}
		ht, err := api.HTTPTestGetByID(rs.Primary.ID)
		if err == nil && ht != nil {
			return fmt.Errorf("httptest %s still exists", rs.Primary.ID)
		}
	}
	return nil
}
