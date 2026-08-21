package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// Acceptance coverage for the backend-type check (item_backend.go).
//
// The unit guard in item_backend_test.go proves the accepted set of every
// resource in the triad against a stub server. This proves the thing the stub
// cannot: that a real `terraform import` of a real item id into the wrong
// resource is refused by a real server, on every supported version, and that
// the message a user sees names both what the object is and which resource
// takes it.
//
// It is worth an acceptance test rather than only a unit one because the
// failure it replaces was *silent success*. Importing an SNMP item as a
// zabbix_item_agent populated state, planned empty, and rewrote the item's
// type -- stopping collection -- on the first unrelated edit. Nothing about
// that shows up in a unit test of a read function; the whole point is that
// the read function was perfectly happy.

const itemBackendMismatchHCL = `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host   = "test-template"
}

// one real SNMP object per family: these are the ids that get imported into
// the wrong resource below
resource "zabbix_lld_snmp" "probe" {
	hostid   = zabbix_template.testtmpl.id
	key      = "probe.rule"
	name     = "Probe Rule"
	snmp_oid = "1.3.6.1.2.1.2.2"
}
resource "zabbix_item_snmp" "probe" {
	hostid    = zabbix_template.testtmpl.id
	key       = "probe.item"
	name      = "Probe Item"
	valuetype = "unsigned"
	snmp_oid  = "1.3.6.1.2.1.1.3.0"
}
resource "zabbix_proto_item_snmp" "probe" {
	hostid    = zabbix_template.testtmpl.id
	ruleid    = zabbix_lld_snmp.probe.id
	key       = "probe.proto[{#SNMPINDEX}]"
	name      = "Probe Proto"
	valuetype = "unsigned"
	snmp_oid  = "1.3.6.1.2.1.2.2.1.10[{#SNMPINDEX}]"
}

// the agent resources a user reaches for by mistake. They are applied as well
// as imported into, so that the import step is exercised against a
// configuration that is otherwise entirely valid
resource "zabbix_lld_agent" "mismatch" {
	hostid = zabbix_template.testtmpl.id
	key    = "mismatch.rule"
	name   = "Mismatch Rule"
}
resource "zabbix_item_agent" "mismatch" {
	hostid    = zabbix_template.testtmpl.id
	key       = "mismatch.item"
	name      = "Mismatch Item"
	valuetype = "unsigned"
}
resource "zabbix_proto_item_agent" "mismatch" {
	hostid    = zabbix_template.testtmpl.id
	ruleid    = zabbix_lld_agent.mismatch.id
	key       = "mismatch.proto[{#SNMPINDEX}]"
	name      = "Mismatch Proto"
	valuetype = "unsigned"
}
`

// testAccImportIDOf hands the import step the id of a *different* resource in
// state, which is what makes this an import of the wrong object rather than a
// round trip.
func testAccImportIDOf(addr string) func(*terraform.State) (string, error) {
	return func(s *terraform.State) (string, error) {
		return testAccStateID(s, addr)
	}
}

func TestAccItemBackendTypeMismatchImport(t *testing.T) {
	config := hcl(t, itemBackendMismatchHCL)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{Config: config},
			{ // an SNMP item imported as a Zabbix agent item
				Config:            config,
				ResourceName:      "zabbix_item_agent.mismatch",
				ImportState:       true,
				ImportStateIdFunc: testAccImportIDOf("zabbix_item_snmp.probe"),
				ExpectError: regexp.MustCompile(
					`item [0-9]+ is a SNMP agent item, not a Zabbix agent or Zabbix agent \(active\) item; ` +
						`import it as zabbix_item_snmp`),
			},
			{ // an SNMP item prototype imported as a Zabbix agent one
				Config:            config,
				ResourceName:      "zabbix_proto_item_agent.mismatch",
				ImportState:       true,
				ImportStateIdFunc: testAccImportIDOf("zabbix_proto_item_snmp.probe"),
				ExpectError: regexp.MustCompile(
					`item prototype [0-9]+ is a SNMP agent item prototype, not a Zabbix agent or ` +
						`Zabbix agent \(active\) item prototype; import it as zabbix_proto_item_snmp`),
			},
			{ // an SNMP discovery rule imported as a Zabbix agent one
				Config:            config,
				ResourceName:      "zabbix_lld_agent.mismatch",
				ImportState:       true,
				ImportStateIdFunc: testAccImportIDOf("zabbix_lld_snmp.probe"),
				ExpectError: regexp.MustCompile(
					`discovery rule [0-9]+ is a SNMP agent discovery rule, not a Zabbix agent or ` +
						`Zabbix agent \(active\) discovery rule; import it as zabbix_lld_snmp`),
			},
			{ // and the right resource still imports the same ids cleanly, so
				// the check is proven to reject on type rather than on
				// everything
				Config:            config,
				ResourceName:      "zabbix_item_snmp.probe",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:            config,
				ResourceName:      "zabbix_lld_snmp.probe",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// testAccCaptureID copies an id out of state so that a later step's PreConfig,
// which is handed no state at all, can reach the object behind Terraform's
// back.
func testAccCaptureID(addr string, into *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccStateID(s, addr)
		if err != nil {
			return err
		}
		*into = id
		return nil
	}
}

// testAccSetItemType changes an item's backend type through the API, which is
// what the Zabbix frontend does when somebody edits the "Type" dropdown.
func testAccSetItemType(t *testing.T, id *string, typ string, extra zabbix.Params) func() {
	return func() {
		api, ok := testAccProvider.Meta().(*zabbix.API)
		if !ok || api == nil {
			t.Fatal("testAccSetItemType: provider not configured")
		}
		params := zabbix.Params{"itemid": *id, "type": typ}
		for k, v := range extra {
			params[k] = v
		}
		if _, err := api.CallWithError("item.update", params); err != nil {
			t.Fatalf("testAccSetItemType(%s -> %s): %s", *id, typ, err)
		}
	}
}

// TestAccItemBackendTypeDrift is the other half of the same defect, and the
// half that has no import in it: somebody changes the type in the frontend.
// Before the check that was invisible -- the read copied the shared properties
// into state, the type was never compared against anything, and every plan
// from then on was empty. The item had silently stopped being what the
// configuration said it was, and the next apply would have quietly changed it
// back without ever saying so.
func TestAccItemBackendTypeDrift(t *testing.T) {
	config := hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host   = "test-template"
}
resource "zabbix_item_agent" "drift" {
	hostid    = zabbix_template.testtmpl.id
	key       = "drift.item"
	name      = "Drift Item"
	valuetype = "unsigned"
	delay     = "1m"
}
`)

	var id string

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCaptureID("zabbix_item_agent.drift", &id),
			},
			{ // retyped to a trapper item outside Terraform: the refresh at
				// the head of this step is where it has to be caught
				PreConfig: testAccSetItemType(t, &id, "2", nil),
				Config:    config,
				ExpectError: regexp.MustCompile(
					`item [0-9]+ is a Zabbix trapper item, not a Zabbix agent or ` +
						`Zabbix agent \(active\) item; import it as zabbix_item_trapper`),
			},
			{ // put it back, and the configuration converges again. Zabbix
				// forces delay to 0 on a trapper item, so the restore names it
				PreConfig: testAccSetItemType(t, &id, "0", zabbix.Params{"delay": "1m"}),
				Config:    config,
				Check: resource.TestCheckResourceAttr(
					"zabbix_item_agent.drift", "delay", "1m"),
			},
		},
	})
}
