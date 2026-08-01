package provider

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/tpretz/go-zabbix-api"
)

// globalMacroSafeID generates a unique ID safe for Zabbix macro names
// (only uppercase A-Z, 0-9, and underscores are allowed after {$...})
func globalMacroSafeID() string {
	return strings.ToUpper(fmt.Sprintf("%d", time.Now().UnixNano()))
}

func TestAccGlobalMacro(t *testing.T) {
	id := globalMacroSafeID()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGlobalMacroDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGlobalMacroConfig(id, "first_value", "text", "first description"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_global_macro.test", "macro", fmt.Sprintf("{$ACC_TEST_%s}", id)),
					resource.TestCheckResourceAttr("zabbix_global_macro.test", "value", "first_value"),
					resource.TestCheckResourceAttr("zabbix_global_macro.test", "type", "text"),
					resource.TestCheckResourceAttr("zabbix_global_macro.test", "description", "first description"),
				),
			},
			{
				// Update: change value and description
				Config: testAccGlobalMacroConfig(id, "updated_value", "text", "updated description"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_global_macro.test", "macro", fmt.Sprintf("{$ACC_TEST_%s}", id)),
					resource.TestCheckResourceAttr("zabbix_global_macro.test", "value", "updated_value"),
					resource.TestCheckResourceAttr("zabbix_global_macro.test", "description", "updated description"),
				),
			},
		},
	})
}

func testAccGlobalMacroConfig(id, value, macroType, description string) string {
	return fmt.Sprintf(`
resource "zabbix_global_macro" "test" {
  macro       = "{$ACC_TEST_%s}"
  value       = "%s"
  type        = "%s"
  description = "%s"
}
`, id, value, macroType, description)
}

func TestAccGlobalMacroSecret(t *testing.T) {
	id := globalMacroSafeID()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGlobalMacroDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGlobalMacroSecretConfig(id, "secret_value", "first secret desc"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_global_macro.secret", "macro", fmt.Sprintf("{$ACC_SECRET_%s}", id)),
					resource.TestCheckResourceAttr("zabbix_global_macro.secret", "type", "secret"),
					resource.TestCheckResourceAttr("zabbix_global_macro.secret", "value", "secret_value"),
					resource.TestCheckResourceAttr("zabbix_global_macro.secret", "description", "first secret desc"),
				),
			},
			{
				// Update: change description (value stays hidden)
				Config: testAccGlobalMacroSecretConfig(id, "secret_value", "updated secret desc"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_global_macro.secret", "type", "secret"),
					resource.TestCheckResourceAttr("zabbix_global_macro.secret", "description", "updated secret desc"),
				),
			},
		},
	})
}

func testAccGlobalMacroSecretConfig(id, value, description string) string {
	return fmt.Sprintf(`
resource "zabbix_global_macro" "secret" {
  macro       = "{$ACC_SECRET_%s}"
  value       = "%s"
  type        = "secret"
  description = "%s"
}
`, id, value, description)
}

func testAccCheckGlobalMacroDestroy(s *terraform.State) error {
	api := testAccProvider.Meta().(*zabbix.API)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "zabbix_global_macro" {
			continue
		}

		macro, err := api.GlobalMacroGetByID(rs.Primary.ID)
		if err == nil && macro != nil {
			return fmt.Errorf("global macro %s still exists", rs.Primary.ID)
		}
	}
	return nil
}
