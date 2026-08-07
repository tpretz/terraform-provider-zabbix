package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// E3 -- ForceNew (PLAN.md Phase 8).
//
// The provider declares exactly three ForceNew attributes and no test proved
// any of them. The attribute is a claim about the *server*: that Zabbix
// cannot move the object, so Terraform must destroy and recreate it. Getting
// it wrong is quiet in both directions. Marked ForceNew when the API would
// have accepted an update, the user loses history for no reason. Not marked
// when the API rejects it, the update fails -- or worse, is accepted and
// ignored, leaving state claiming a move that never happened.
//
// Each case here asserts three things, because any one alone can pass for the
// wrong reason:
//
//  1. the plan is a destroy-and-create, not an in-place update
//     (an attribute silently dropped from the update call plans as no diff at
//     all, and an attribute that is diffed but not ForceNew plans as Update)
//  2. the object that comes out has a different id
//     (a plan can say replace and still be a no-op if the provider's Create
//     happens to find and adopt the existing object)
//  3. the old id is gone from Zabbix
//     (Terraform is happy either way -- it stops tracking the old object -- so
//     only the server can say whether it was actually destroyed or leaked)
//
// A control step at the end of each test changes a *non*-ForceNew attribute
// and requires an in-place update, so "everything is replaced" cannot pass as
// "this attribute is ForceNew".

// testAccCheckAttrChanged requires addr's attribute to differ from a value
// recorded by testAccRecordAttr in an earlier step.
func testAccCheckAttrChanged(addr, attr string, prev *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[addr]
		if !ok {
			return fmt.Errorf("%s not found in state", addr)
		}
		v := rs.Primary.Attributes[attr]
		if attr == "id" {
			v = rs.Primary.ID
		}
		if v == "" {
			return fmt.Errorf("%s has no %s", addr, attr)
		}
		if v == *prev {
			return fmt.Errorf("%s: %s is still %q -- the resource was not recreated", addr, attr, v)
		}
		return nil
	}
}

// testAccCheckGone requires an id recorded in an earlier step to no longer
// exist in Zabbix. Terraform cannot tell a destroyed object from an
// abandoned one, so this is the only check that distinguishes them.
func testAccCheckGone(prev *string, exists func(*zabbix.API, string) (bool, error)) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		api, ok := testAccProvider.Meta().(*zabbix.API)
		if !ok || api == nil {
			return fmt.Errorf("testAccCheckGone: provider not configured")
		}
		if *prev == "" {
			return fmt.Errorf("testAccCheckGone: nothing was recorded to look for")
		}
		found, err := exists(api, *prev)
		if err != nil {
			return fmt.Errorf("looking up the replaced object %s: %s", *prev, err)
		}
		if found {
			return fmt.Errorf("the replaced object %s still exists in Zabbix -- it was abandoned, not destroyed", *prev)
		}
		return nil
	}
}

// forceNewTemplatesHCL gives the hostid cases two owners to move between.
const forceNewTemplatesHCL = `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-forcenew-template-group"
}
resource "zabbix_template" "testtmpla" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host   = "test-forcenew-template-a"
}
resource "zabbix_template" "testtmplb" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host   = "test-forcenew-template-b"
}
`

// TestAccForceNewItemHostid covers itemCommonSchema's "hostid", shared by all
// ten zabbix_item_* and all ten zabbix_proto_item_* resources.
func TestAccForceNewItemHostid(t *testing.T) {
	var itemID string

	itemOn := func(tmpl, name string) string {
		return hcl(t, forceNewTemplatesHCL+`
resource "zabbix_item_agent" "testitem" {
	hostid    = `+tmpl+`.id
	key       = "forcenew.item"
	name      = "`+name+`"
	valuetype = "unsigned"
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: itemOn("zabbix_template.testtmpla", "ForceNew Item"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("zabbix_item_agent.testitem", "hostid", "zabbix_template.testtmpla", "id"),
					testAccRecordAttr("zabbix_item_agent.testitem", "id", &itemID),
				),
			},
			{ // move it to the other template: an item cannot change host
				Config: itemOn("zabbix_template.testtmplb", "ForceNew Item"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("zabbix_item_agent.testitem", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("zabbix_item_agent.testitem", "hostid", "zabbix_template.testtmplb", "id"),
					testAccCheckAttrChanged("zabbix_item_agent.testitem", "id", &itemID),
					testAccCheckGone(&itemID, testAccItemExists),
					testAccRecordAttr("zabbix_item_agent.testitem", "id", &itemID),
				),
			},
			{ // control: renaming is not a replacement
				Config: itemOn("zabbix_template.testtmplb", "ForceNew Item Renamed"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("zabbix_item_agent.testitem", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.testitem", "name", "ForceNew Item Renamed"),
					resource.TestCheckResourceAttrPtr("zabbix_item_agent.testitem", "id", &itemID),
				),
			},
		},
	})
}

// TestAccForceNewLLDHostid covers lldCommonSchema's "hostid", shared by all
// eight zabbix_lld_* resources. It is a separate declaration from the item
// one -- common_lld.go, not common_item.go -- so the item test says nothing
// about it.
func TestAccForceNewLLDHostid(t *testing.T) {
	var lldID string

	lldOn := func(tmpl, name string) string {
		return hcl(t, forceNewTemplatesHCL+`
resource "zabbix_lld_trapper" "testlld" {
	hostid = `+tmpl+`.id
	key    = "forcenew.lld"
	name   = "`+name+`"
	delay  = "0"
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: lldOn("zabbix_template.testtmpla", "ForceNew LLD"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("zabbix_lld_trapper.testlld", "hostid", "zabbix_template.testtmpla", "id"),
					testAccRecordAttr("zabbix_lld_trapper.testlld", "id", &lldID),
				),
			},
			{
				Config: lldOn("zabbix_template.testtmplb", "ForceNew LLD"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("zabbix_lld_trapper.testlld", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("zabbix_lld_trapper.testlld", "hostid", "zabbix_template.testtmplb", "id"),
					testAccCheckAttrChanged("zabbix_lld_trapper.testlld", "id", &lldID),
					testAccCheckGone(&lldID, testAccLLDExists),
					testAccRecordAttr("zabbix_lld_trapper.testlld", "id", &lldID),
				),
			},
			{ // control
				Config: lldOn("zabbix_template.testtmplb", "ForceNew LLD Renamed"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("zabbix_lld_trapper.testlld", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "name", "ForceNew LLD Renamed"),
					resource.TestCheckResourceAttrPtr("zabbix_lld_trapper.testlld", "id", &lldID),
				),
			},
		},
	})
}

// TestAccForceNewProtoItemRuleid covers itemPrototypeSchema's "ruleid", the
// one ForceNew attribute unique to the prototype resources. Unlike hostid it
// stays within one template, so this is specifically about the prototype's
// membership of a discovery rule rather than about moving hosts.
func TestAccForceNewProtoItemRuleid(t *testing.T) {
	var protoID string

	protoOn := func(rule, name string) string {
		return hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-forcenew-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host   = "test-forcenew-template"
}
resource "zabbix_lld_trapper" "testllda" {
	hostid = zabbix_template.testtmpl.id
	key    = "forcenew.lld.a"
	name   = "ForceNew LLD A"
	delay  = "0"
}
resource "zabbix_lld_trapper" "testlldb" {
	hostid = zabbix_template.testtmpl.id
	key    = "forcenew.lld.b"
	name   = "ForceNew LLD B"
	delay  = "0"
}
resource "zabbix_proto_item_trapper" "testproto" {
	hostid    = zabbix_template.testtmpl.id
	ruleid    = `+rule+`.id
	key       = "forcenew.proto[{#FSNAME}]"
	name      = "`+name+`"
	valuetype = "unsigned"
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: protoOn("zabbix_lld_trapper.testllda", "ForceNew Proto {#FSNAME}"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("zabbix_proto_item_trapper.testproto", "ruleid", "zabbix_lld_trapper.testllda", "id"),
					testAccRecordAttr("zabbix_proto_item_trapper.testproto", "id", &protoID),
				),
			},
			{ // re-parent onto the other discovery rule
				Config: protoOn("zabbix_lld_trapper.testlldb", "ForceNew Proto {#FSNAME}"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("zabbix_proto_item_trapper.testproto", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("zabbix_proto_item_trapper.testproto", "ruleid", "zabbix_lld_trapper.testlldb", "id"),
					testAccCheckAttrChanged("zabbix_proto_item_trapper.testproto", "id", &protoID),
					testAccCheckGone(&protoID, testAccProtoItemExists),
					testAccRecordAttr("zabbix_proto_item_trapper.testproto", "id", &protoID),
				),
			},
			{ // control
				Config: protoOn("zabbix_lld_trapper.testlldb", "ForceNew Proto Renamed {#FSNAME}"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("zabbix_proto_item_trapper.testproto", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proto_item_trapper.testproto", "name", "ForceNew Proto Renamed {#FSNAME}"),
					resource.TestCheckResourceAttrPtr("zabbix_proto_item_trapper.testproto", "id", &protoID),
				),
			},
		},
	})
}
