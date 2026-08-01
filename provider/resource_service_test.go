package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccService(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceConfig(id, "first"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_service.test", "name", "acc-svc-"+id+"-first"),
					resource.TestCheckResourceAttr("zabbix_service.test", "algorithm", "most_critical_all"),
					resource.TestCheckResourceAttr("zabbix_service.test", "sortorder", "0"),
					resource.TestCheckResourceAttr("zabbix_service.test", "description", "first description"),
					resource.TestCheckResourceAttr("zabbix_service.test", "problem_tags.#", "1"),
					resource.TestCheckResourceAttr("zabbix_service.test", "problem_tags.0.tag", "scope"),
					resource.TestCheckResourceAttr("zabbix_service.test", "tags.#", "1"),
					resource.TestCheckResourceAttr("zabbix_service.test", "tags.0.tag", "env"),
				),
			},
			{
				// Update: change name, algorithm, add status_rules, change tags
				Config: testAccServiceConfig(id, "second"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_service.test", "name", "acc-svc-"+id+"-second"),
					resource.TestCheckResourceAttr("zabbix_service.test", "algorithm", "most_critical_child"),
					resource.TestCheckResourceAttr("zabbix_service.test", "sortorder", "1"),
					resource.TestCheckResourceAttr("zabbix_service.test", "description", "second description"),
					resource.TestCheckResourceAttr("zabbix_service.test", "problem_tags.#", "1"),
					resource.TestCheckResourceAttr("zabbix_service.test", "problem_tags.0.tag", "env"),
					resource.TestCheckResourceAttr("zabbix_service.test", "tags.#", "1"),
					resource.TestCheckResourceAttr("zabbix_service.test", "tags.0.tag", "team"),
					resource.TestCheckResourceAttr("zabbix_service.test", "status_rules.#", "1"),
				),
			},
		},
	})
}

func testAccServiceConfig(id, variant string) string {
	if variant == "first" {
		return fmt.Sprintf(`
resource "zabbix_service" "test" {
  name        = "acc-svc-%s-first"
  algorithm   = "most_critical_all"
  sortorder   = "0"
  description = "first description"

  problem_tags {
    tag      = "scope"
    operator = "0"
    value    = "prod"
  }

  tags {
    tag   = "env"
    value = "test"
  }
}
`, id)
	}
	return fmt.Sprintf(`
resource "zabbix_service" "test" {
  name        = "acc-svc-%s-second"
  algorithm   = "most_critical_child"
  sortorder   = "1"
  description = "second description"

  problem_tags {
    tag      = "env"
    operator = "0"
    value    = "staging"
  }

  tags {
    tag   = "team"
    value = "sre"
  }

  status_rules {
    type         = "0"
    limit_value  = "1"
    limit_status = "3"
    new_status   = "4"
  }
}
`, id)
}

func testAccCheckServiceDestroy(s *terraform.State) error {
	api := testAccProvider.Meta().(*zabbix.API)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "zabbix_service" {
			continue
		}
		svc, err := api.ServiceGetByID(rs.Primary.ID)
		if err == nil && svc != nil {
			return fmt.Errorf("service %s still exists", rs.Primary.ID)
		}
	}
	return nil
}
