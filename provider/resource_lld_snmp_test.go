package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceLLDSnmp(t *testing.T) {
	id := resource.UniqueId()
	groupName := "test-group-" + id
	tmplHost := "test-template-" + id

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

resource "zabbix_lld_snmp" "testrule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.snmp.discovery"
  name   = "LLD SNMP Rule"

  snmp_version   = "2"
  snmp_oid       = ".1.3.6.1.2.1.1.3.0"
  snmp_community = "{$SNMP_COMMUNITY}"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_snmp.testrule", "key", "lld.snmp.discovery"),
					resource.TestCheckResourceAttr("zabbix_lld_snmp.testrule", "name", "LLD SNMP Rule"),
					resource.TestCheckResourceAttr("zabbix_lld_snmp.testrule", "snmp_version", "2"),
					resource.TestCheckResourceAttr("zabbix_lld_snmp.testrule", "snmp_oid", ".1.3.6.1.2.1.1.3.0"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [zabbix_hostgroup.testgrp.id]
  host   = %q
}

resource "zabbix_lld_snmp" "testrule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.snmp.discovery2"
  name   = "LLD SNMP Rule A"

  snmp_version   = "1"
  snmp_oid       = ".1.3.6.1.2.1.1.5.0"
  snmp_community = "{$SNMP_COMMUNITY}"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_snmp.testrule", "key", "lld.snmp.discovery2"),
					resource.TestCheckResourceAttr("zabbix_lld_snmp.testrule", "name", "LLD SNMP Rule A"),
					resource.TestCheckResourceAttr("zabbix_lld_snmp.testrule", "snmp_version", "1"),
					resource.TestCheckResourceAttr("zabbix_lld_snmp.testrule", "snmp_oid", ".1.3.6.1.2.1.1.5.0"),
				),
			},
		},
	})
}
