package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccScript_CustomScript(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckScriptDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "zabbix_script" "test" {
  name        = "acc-script-%s"
  command     = "echo hello"
  type        = "script"
  scope       = "action_operation"
  execute_on  = "server"
  description = "created"
}
`, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_script.test", "name", "acc-script-"+id),
					resource.TestCheckResourceAttr("zabbix_script.test", "type", "script"),
					resource.TestCheckResourceAttr("zabbix_script.test", "scope", "action_operation"),
					resource.TestCheckResourceAttr("zabbix_script.test", "execute_on", "server"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "zabbix_script" "test" {
  name        = "acc-script-updated-%s"
  command     = "echo world"
  type        = "script"
  scope       = "action_operation"
  execute_on  = "server"
  description = "updated"
}
`, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_script.test", "name", "acc-script-updated-"+id),
					resource.TestCheckResourceAttr("zabbix_script.test", "command", "echo world"),
					resource.TestCheckResourceAttr("zabbix_script.test", "description", "updated"),
				),
			},
		},
	})
}

func TestAccScript_Webhook(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckScriptDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "zabbix_script" "webhook" {
  name        = "acc-webhook-script-%s"
  command     = "return 1;"
  type        = "webhook"
  scope       = "action_operation"
  timeout     = "30s"
  description = "created"

  parameters {
    name  = "url"
    value = "http://example.com"
  }
}
`, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_script.webhook", "name", "acc-webhook-script-"+id),
					resource.TestCheckResourceAttr("zabbix_script.webhook", "type", "webhook"),
					resource.TestCheckResourceAttr("zabbix_script.webhook", "timeout", "30s"),
					resource.TestCheckResourceAttr("zabbix_script.webhook", "parameters.#", "1"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "zabbix_script" "webhook" {
  name        = "acc-webhook-script-updated-%s"
  command     = "return 2;"
  type        = "webhook"
  scope       = "action_operation"
  timeout     = "60s"
  description = "updated"

  parameters {
    name  = "url"
    value = "http://updated.example.com"
  }
  parameters {
    name  = "token"
    value = "xyz"
  }
}
`, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_script.webhook", "name", "acc-webhook-script-updated-"+id),
					resource.TestCheckResourceAttr("zabbix_script.webhook", "command", "return 2;"),
					resource.TestCheckResourceAttr("zabbix_script.webhook", "timeout", "60s"),
					resource.TestCheckResourceAttr("zabbix_script.webhook", "parameters.#", "2"),
				),
			},
		},
	})
}

func testAccCheckScriptDestroy(s *terraform.State) error {
	api := testAccProvider.Meta().(*zabbix.API)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "zabbix_script" {
			continue
		}
		script, err := api.ScriptGetByID(rs.Primary.ID)
		if err == nil && script != nil {
			return fmt.Errorf("script %s still exists", rs.Primary.ID)
		}
	}
	return nil
}
