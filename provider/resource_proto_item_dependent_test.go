package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccResourceProtoItemDependent also covers the two shapes of master item a
// dependent prototype can have: a plain item on the same template, and another
// item prototype under the same discovery rule.
func TestAccResourceProtoItemDependent(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // create, mastered by a plain item
				Config: protoItemConfig(t, `
resource "zabbix_item_trapper" "masteritem" {
	hostid = zabbix_template.testtmpl.id
	key = "test.master"

	name = "Master Item"
	valuetype = "text"
}
resource "zabbix_proto_item_dependent" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "proto.dependent[{#FSNAME}]"

	name = "Proto Dependent {#FSNAME}"
	valuetype = "text"

	master_itemid = zabbix_item_trapper.masteritem.id
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_dependent.testproto", "key", "proto.dependent[{#FSNAME}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_dependent.testproto", "name", "Proto Dependent {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_dependent.testproto", "valuetype", "text"),
					resource.TestCheckResourceAttrPair(
						"zabbix_proto_item_dependent.testproto", "master_itemid",
						"zabbix_item_trapper.masteritem", "id"),
					resource.TestCheckResourceAttrPair(
						"zabbix_proto_item_dependent.testproto", "ruleid",
						"zabbix_lld_trapper.testlld", "id"),
				),
			},
			{ // update: repoint the master at another prototype under the same rule
				Config: protoItemConfig(t, `
resource "zabbix_item_trapper" "masteritem" {
	hostid = zabbix_template.testtmpl.id
	key = "test.master"

	name = "Master Item"
	valuetype = "text"
}
resource "zabbix_proto_item_trapper" "masterproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "proto.master[{#FSNAME}]"

	name = "Master Proto {#FSNAME}"
	valuetype = "text"
}
resource "zabbix_proto_item_dependent" "testproto" {
	hostid = zabbix_template.testtmpl.id
	ruleid = zabbix_lld_trapper.testlld.id
	key = "proto.dependent.changed[{#FSNAME}]"

	name = "Proto Dependent Changed {#FSNAME}"
	valuetype = "character"

	master_itemid = zabbix_proto_item_trapper.masterproto.id
}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_dependent.testproto", "key", "proto.dependent.changed[{#FSNAME}]"),
					resource.TestCheckResourceAttr("zabbix_proto_item_dependent.testproto", "name", "Proto Dependent Changed {#FSNAME}"),
					resource.TestCheckResourceAttr("zabbix_proto_item_dependent.testproto", "valuetype", "character"),
					resource.TestCheckResourceAttrPair(
						"zabbix_proto_item_dependent.testproto", "master_itemid",
						"zabbix_proto_item_trapper.masterproto", "id"),
				),
			},
			{ // import
				ResourceName:      "zabbix_proto_item_dependent.testproto",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
