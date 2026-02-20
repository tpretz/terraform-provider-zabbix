package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcesTemplateHostgroupHost(t *testing.T) {
	id := resource.UniqueId()
	groupName := "test-group-" + id
	tmplHost := "test-template-" + id
	hostName := "test-host-" + id

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}

resource "zabbix_template" "testtmpl" {
  groups = [zabbix_hostgroup.testgrp.id]
  host   = %q
}

resource "zabbix_host" "testhost" {
  host = %q
  interface {
    type = "agent"
    main = true
    ip   = "127.0.0.1"
    dns  = ""
    port = 10050
  }
  groups    = [zabbix_hostgroup.testgrp.id]
  templates = [zabbix_template.testtmpl.id]
}

# Data sources

data "zabbix_hostgroup" "by_name" {
  name = zabbix_hostgroup.testgrp.name
}

data "zabbix_template" "by_host" {
  host = zabbix_template.testtmpl.host
}

data "zabbix_host" "by_host" {
  host = zabbix_host.testhost.host
}
`, groupName, tmplHost, hostName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.zabbix_hostgroup.by_name", "id"),
					resource.TestCheckResourceAttr("data.zabbix_hostgroup.by_name", "name", groupName),

					resource.TestCheckResourceAttrSet("data.zabbix_template.by_host", "id"),
					resource.TestCheckResourceAttr("data.zabbix_template.by_host", "host", tmplHost),

					resource.TestCheckResourceAttrSet("data.zabbix_host.by_host", "id"),
					resource.TestCheckResourceAttr("data.zabbix_host.by_host", "host", hostName),
				),
			},
		},
	})
}
