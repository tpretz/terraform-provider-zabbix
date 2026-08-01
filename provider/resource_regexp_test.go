package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccRegexp(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRegexpDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRegexpConfig(id, "test_regexp_"+id, "hello123", "^hello", "regexp", true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_regexp.test", "name", "test_regexp_"+id),
					resource.TestCheckResourceAttr("zabbix_regexp.test", "test_string", "hello123"),
					resource.TestCheckResourceAttr("zabbix_regexp.test", "expressions.#", "1"),
					resource.TestCheckResourceAttr("zabbix_regexp.test", "expressions.0.expression", "^hello"),
					resource.TestCheckResourceAttr("zabbix_regexp.test", "expressions.0.expression_type", "regexp"),
					resource.TestCheckResourceAttr("zabbix_regexp.test", "expressions.0.exp_delimiter", ""),
					resource.TestCheckResourceAttr("zabbix_regexp.test", "expressions.0.case_sensitive", "true"),
				),
			},
			{
				// Update: change name, test_string, expression, and case_sensitive
				Config: testAccRegexpConfig(id, "test_regexp_updated_"+id, "world456", "^world", "not_regexp", false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_regexp.test", "name", "test_regexp_updated_"+id),
					resource.TestCheckResourceAttr("zabbix_regexp.test", "test_string", "world456"),
					resource.TestCheckResourceAttr("zabbix_regexp.test", "expressions.0.expression", "^world"),
					resource.TestCheckResourceAttr("zabbix_regexp.test", "expressions.0.expression_type", "not_regexp"),
					resource.TestCheckResourceAttr("zabbix_regexp.test", "expressions.0.case_sensitive", "false"),
				),
			},
		},
	})
}

func testAccRegexpConfig(id, name, testString, expression, exprType string, caseSensitive bool) string {
	return fmt.Sprintf(`
resource "zabbix_regexp" "test" {
  name        = "%s"
  test_string = "%s"

  expressions {
    expression      = "%s"
    expression_type = "%s"
    exp_delimiter   = ""
    case_sensitive  = %t
  }
}
`, name, testString, expression, exprType, caseSensitive)
}

func testAccCheckRegexpDestroy(s *terraform.State) error {
	api := testAccProvider.Meta().(*zabbix.API)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "zabbix_regexp" {
			continue
		}

		r, err := api.RegexpGetByID(rs.Primary.ID)
		if err == nil && r != nil {
			return fmt.Errorf("regexp %s still exists", rs.Primary.ID)
		}
	}
	return nil
}
