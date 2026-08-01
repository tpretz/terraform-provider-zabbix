package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccValueMap(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckValueMapDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccValueMapConfig(id, "test_vm_"+id, "Down", "Up"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_valuemap.test", "name", "test_vm_"+id),
					resource.TestCheckResourceAttr("zabbix_valuemap.test", "mappings.#", "3"),
					resource.TestCheckResourceAttr("zabbix_valuemap.test", "mappings.0.type", "equal"),
					resource.TestCheckResourceAttr("zabbix_valuemap.test", "mappings.0.value", "0"),
					resource.TestCheckResourceAttr("zabbix_valuemap.test", "mappings.0.newvalue", "Down"),
					resource.TestCheckResourceAttr("zabbix_valuemap.test", "mappings.1.type", "equal"),
					resource.TestCheckResourceAttr("zabbix_valuemap.test", "mappings.1.value", "1"),
					resource.TestCheckResourceAttr("zabbix_valuemap.test", "mappings.1.newvalue", "Up"),
					resource.TestCheckResourceAttr("zabbix_valuemap.test", "mappings.2.type", "default"),
					resource.TestCheckResourceAttr("zabbix_valuemap.test", "mappings.2.value", ""),
					resource.TestCheckResourceAttr("zabbix_valuemap.test", "mappings.2.newvalue", "Unknown"),
					resource.TestCheckResourceAttrSet("zabbix_valuemap.test", "hostid"),
				),
			},
			{
				// Update: change name and mapping values
				Config: testAccValueMapConfig(id, "test_vm_updated_"+id, "Offline", "Online"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_valuemap.test", "name", "test_vm_updated_"+id),
					resource.TestCheckResourceAttr("zabbix_valuemap.test", "mappings.0.newvalue", "Offline"),
					resource.TestCheckResourceAttr("zabbix_valuemap.test", "mappings.1.newvalue", "Online"),
				),
			},
		},
	})
}

func testAccValueMapConfig(id, name, val0, val1 string) string {
	return fmt.Sprintf(`
resource "zabbix_templategroup" "test" {
  name = "acc-vm-tg-%s"
}

resource "zabbix_template" "test" {
  host   = "acc-vm-tmpl-%s"
  groups = [zabbix_templategroup.test.id]
}

resource "zabbix_valuemap" "test" {
  hostid = zabbix_template.test.id
  name   = "%s"

  mappings {
    type     = "equal"
    value    = "0"
    newvalue = "%s"
  }

  mappings {
    type     = "equal"
    value    = "1"
    newvalue = "%s"
  }

  mappings {
    type     = "default"
    value    = ""
    newvalue = "Unknown"
  }
}
`, id, id, name, val0, val1)
}

func testAccCheckValueMapDestroy(s *terraform.State) error {
	api := testAccProvider.Meta().(*zabbix.API)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "zabbix_valuemap" {
			continue
		}

		vm, err := api.ValueMapGetByID(rs.Primary.ID)
		if err == nil && vm != nil {
			return fmt.Errorf("valuemap %s still exists", rs.Primary.ID)
		}
	}
	return nil
}

func TestAccDataValueMap(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataValueMapConfig(id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.zabbix_valuemap.test", "name", "data_vm_"+id),
					resource.TestCheckResourceAttr("data.zabbix_valuemap.test", "mappings.#", "1"),
					resource.TestCheckResourceAttr("data.zabbix_valuemap.test", "mappings.0.type", "equal"),
					resource.TestCheckResourceAttr("data.zabbix_valuemap.test", "mappings.0.value", "1"),
					resource.TestCheckResourceAttr("data.zabbix_valuemap.test", "mappings.0.newvalue", "Yes"),
				),
			},
		},
	})
}

func testAccDataValueMapConfig(id string) string {
	return fmt.Sprintf(`
resource "zabbix_templategroup" "test" {
  name = "acc-dvm-tg-%s"
}

resource "zabbix_template" "test" {
  host   = "acc-dvm-tmpl-%s"
  groups = [zabbix_templategroup.test.id]
}

resource "zabbix_valuemap" "test" {
  hostid = zabbix_template.test.id
  name   = "data_vm_%s"

  mappings {
    type     = "equal"
    value    = "1"
    newvalue = "Yes"
  }
}

data "zabbix_valuemap" "test" {
  hostid = zabbix_template.test.id
  name   = zabbix_valuemap.test.name
}
`, id, id, id)
}
