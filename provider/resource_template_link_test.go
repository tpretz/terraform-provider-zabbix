package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccTemplateLink(t *testing.T) {
	id := resource.UniqueId()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccTemplateLinkConfig(id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("zabbix_template_link.test", "template_id"),
					resource.TestCheckResourceAttr("zabbix_template_link.test", "item.#", "1"),
				),
			},
		},
	})
}

func testAccTemplateLinkConfig(id string) string {
	return fmt.Sprintf(`
resource "zabbix_templategroup" "tl_grp" {
  name = "tl-group-%s"
}

resource "zabbix_template" "tl_tmpl" {
  host   = "tl-tmpl-%s"
  groups = [zabbix_templategroup.tl_grp.id]
}

resource "zabbix_item_trapper" "tl_item" {
  hostid    = zabbix_template.tl_tmpl.id
  key       = "tl.test.item[%s]"
  name      = "TL Test Item %s"
  valuetype = "unsigned"
}

resource "zabbix_template_link" "test" {
  template_id = zabbix_template.tl_tmpl.id
  item {
    item_id = zabbix_item_trapper.tl_item.id
  }
}
`, id, id, id, id)
}
