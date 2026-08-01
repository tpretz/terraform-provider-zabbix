package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccSLA(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSLADestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSLAConfig(id, "first"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_sla.test", "name", "acc-sla-"+id+"-first"),
					resource.TestCheckResourceAttr("zabbix_sla.test", "period", "daily"),
					resource.TestCheckResourceAttr("zabbix_sla.test", "slo", "99.9"),
					resource.TestCheckResourceAttr("zabbix_sla.test", "status", "enabled"),
					resource.TestCheckResourceAttr("zabbix_sla.test", "service_tags.#", "1"),
					resource.TestCheckResourceAttr("zabbix_sla.test", "service_tags.0.tag", "scope"),
				),
			},
			{
				// Update: change name, slo, add schedule and excluded_downtimes
				Config: testAccSLAConfig(id, "second"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_sla.test", "name", "acc-sla-"+id+"-second"),
					resource.TestCheckResourceAttr("zabbix_sla.test", "slo", "99.95"),
					resource.TestCheckResourceAttr("zabbix_sla.test", "description", "updated"),
					resource.TestCheckResourceAttr("zabbix_sla.test", "service_tags.#", "1"),
					resource.TestCheckResourceAttr("zabbix_sla.test", "service_tags.0.tag", "env"),
					resource.TestCheckResourceAttr("zabbix_sla.test", "schedule.#", "1"),
					resource.TestCheckResourceAttr("zabbix_sla.test", "excluded_downtimes.#", "1"),
				),
			},
		},
	})
}

func testAccSLAConfig(id, variant string) string {
	if variant == "first" {
		return fmt.Sprintf(`
resource "zabbix_sla" "test" {
  name           = "acc-sla-%s-first"
  period         = "daily"
  slo            = "99.9"
  effective_date = "1672531200"
  timezone       = "UTC"
  status         = "enabled"
  description    = "first"

  service_tags {
    tag      = "scope"
    operator = "0"
    value    = "prod"
  }
}
`, id)
	}
	return fmt.Sprintf(`
resource "zabbix_sla" "test" {
  name           = "acc-sla-%s-second"
  period         = "daily"
  slo            = "99.95"
  effective_date = "1672531200"
  timezone       = "UTC"
  status         = "enabled"
  description    = "updated"

  service_tags {
    tag      = "env"
    operator = "0"
    value    = "staging"
  }

  schedule {
    period_from = "0"
    period_to   = "86400"
  }

  excluded_downtimes {
    name        = "maintenance"
    period_from = "1672617600"
    period_to   = "1672704000"
  }
}
`, id)
}

func testAccCheckSLADestroy(s *terraform.State) error {
	api := testAccProvider.Meta().(*zabbix.API)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "zabbix_sla" {
			continue
		}
		sla, err := api.SLAGetByID(rs.Primary.ID)
		if err == nil && sla != nil {
			return fmt.Errorf("SLA %s still exists", rs.Primary.ID)
		}
	}
	return nil
}
