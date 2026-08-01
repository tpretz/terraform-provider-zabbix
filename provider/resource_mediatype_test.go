package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccMediatype_Email(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMediatypeDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "zabbix_mediatype" "email" {
  name                = "acc-email-%s"
  type                = "email"
  smtp_server         = "mail.example.com"
  smtp_port           = "25"
  smtp_helo           = "example.com"
  smtp_email          = "zabbix@example.com"
  smtp_security       = "none"
  smtp_authentication = "none"
  status              = "enabled"
  description         = "created"
}
`, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_mediatype.email", "name", "acc-email-"+id),
					resource.TestCheckResourceAttr("zabbix_mediatype.email", "type", "email"),
					resource.TestCheckResourceAttr("zabbix_mediatype.email", "smtp_server", "mail.example.com"),
					resource.TestCheckResourceAttr("zabbix_mediatype.email", "status", "enabled"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "zabbix_mediatype" "email" {
  name                = "acc-email-updated-%s"
  type                = "email"
  smtp_server         = "mail2.example.com"
  smtp_port           = "587"
  smtp_helo           = "example.org"
  smtp_email          = "alerts@example.org"
  smtp_security       = "starttls"
  smtp_authentication = "password"
  username            = "user"
  passwd              = "secret"
  status              = "disabled"
  description         = "updated"
}
`, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_mediatype.email", "name", "acc-email-updated-"+id),
					resource.TestCheckResourceAttr("zabbix_mediatype.email", "smtp_server", "mail2.example.com"),
					resource.TestCheckResourceAttr("zabbix_mediatype.email", "smtp_port", "587"),
					resource.TestCheckResourceAttr("zabbix_mediatype.email", "smtp_security", "starttls"),
					resource.TestCheckResourceAttr("zabbix_mediatype.email", "smtp_authentication", "password"),
					resource.TestCheckResourceAttr("zabbix_mediatype.email", "status", "disabled"),
				),
			},
		},
	})
}

func TestAccMediatype_Webhook(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMediatypeDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "zabbix_mediatype" "webhook" {
  name             = "acc-webhook-%s"
  type             = "webhook"
  script           = "return 'OK';"
  timeout          = "30s"
  process_tags     = false
  show_event_menu  = false
  status           = "enabled"
  description      = "created"

  parameters {
    name  = "URL"
    value = "http://example.com"
  }
}
`, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_mediatype.webhook", "name", "acc-webhook-"+id),
					resource.TestCheckResourceAttr("zabbix_mediatype.webhook", "type", "webhook"),
					resource.TestCheckResourceAttr("zabbix_mediatype.webhook", "timeout", "30s"),
					resource.TestCheckResourceAttr("zabbix_mediatype.webhook", "parameters.0.name", "URL"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "zabbix_mediatype" "webhook" {
  name             = "acc-webhook-updated-%s"
  type             = "webhook"
  script           = "return 'UPDATED';"
  timeout          = "60s"
  process_tags     = true
  show_event_menu  = true
  event_menu_url   = "http://example.com/{EVENT.ID}"
  event_menu_name  = "View Event"
  status           = "disabled"
  description      = "updated"

  parameters {
    name  = "URL"
    value = "http://updated.example.com"
  }
  parameters {
    name  = "Token"
    value = "abc123"
  }
}
`, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_mediatype.webhook", "name", "acc-webhook-updated-"+id),
					resource.TestCheckResourceAttr("zabbix_mediatype.webhook", "timeout", "60s"),
					resource.TestCheckResourceAttr("zabbix_mediatype.webhook", "process_tags", "true"),
					resource.TestCheckResourceAttr("zabbix_mediatype.webhook", "show_event_menu", "true"),
					resource.TestCheckResourceAttr("zabbix_mediatype.webhook", "parameters.#", "2"),
				),
			},
		},
	})
}

func testAccCheckMediatypeDestroy(s *terraform.State) error {
	api := testAccProvider.Meta().(*zabbix.API)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "zabbix_mediatype" {
			continue
		}
		mt, err := api.MediaTypeGetByID(rs.Primary.ID)
		if err == nil && mt != nil {
			return fmt.Errorf("mediatype %s still exists", rs.Primary.ID)
		}
	}
	return nil
}
