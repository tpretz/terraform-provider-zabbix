package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccResourceProtoItemSnmp(t *testing.T) {
	id := resource.UniqueId()
	groupName := "test-group-" + id
	tmplHost := "test-template-" + id

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// Zabbix < 5.0 supports explicit SNMP version/security fields on items.
			{
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version >= 50000, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [zabbix_hostgroup.testgrp.id]
  host   = %q
}

resource "zabbix_lld_simple" "rule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.simple.discovery"
  name   = "LLD Simple Rule"
}

resource "zabbix_proto_item_snmp" "testitem" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "snmp.proto.key"

  name      = "Proto SNMP Item"
  valuetype = "text"

  interfaceid = "0"
  delay       = "1m"

  snmp_oid      = "1.2.3.4"
  snmp_version  = "2"
  snmp_community = "{$SNMP_COMMUNITY}"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("zabbix_proto_item_snmp.testitem", "ruleid"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmp.testitem", "snmp_oid", "1.2.3.4"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmp.testitem", "snmp_version", "2"),
				),
			},
			{
				SkipFunc: func() (bool, error) {
					api := testAccProvider.Meta().(*zabbix.API)
					return api.Config.Version >= 50000, nil
				},
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp" {
  name = %q
}
resource "zabbix_template" "testtmpl" {
  groups = [zabbix_hostgroup.testgrp.id]
  host   = %q
}

resource "zabbix_lld_simple" "rule" {
  hostid = zabbix_template.testtmpl.id
  key    = "lld.simple.discovery"
  name   = "LLD Simple Rule"
}

resource "zabbix_proto_item_snmp" "testitem" {
  ruleid = zabbix_lld_simple.rule.id
  hostid = zabbix_template.testtmpl.id
  key    = "snmp.proto.key2"

  name      = "Proto SNMP Item A"
  valuetype = "unsigned"

  interfaceid = "0"
  delay       = "30s"

  snmp_oid      = "1.2.3.5"
  snmp_version  = "3"
  snmp3_authpassphrase = "{$SNMP3_AUTHPASSPHRASE}"
  snmp3_authprotocol   = "sha"
  snmp3_contextname    = "{$SNMP3_CONTEXTNAME}"
  snmp3_privpassphrase = "{$SNMP3_PRIVPASSPHRASE}"
  snmp3_privprotocol   = "aes"
  snmp3_securitylevel  = "authpriv"
  snmp3_securityname   = "{$SNMP3_SECURITYNAME}"
}
`, groupName, tmplHost),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_snmp.testitem", "snmp_oid", "1.2.3.5"),
					resource.TestCheckResourceAttr("zabbix_proto_item_snmp.testitem", "snmp_version", "3"),
				),
			},
		},
	})
}
