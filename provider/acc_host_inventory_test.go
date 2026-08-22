package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// inventory_mode = "automatic" and the fields Zabbix populates itself.
//
// Under "automatic" Zabbix fills inventory fields in from any item carrying an
// `inventory_link`. flattenInventory used to copy every non-empty field the
// server returned into state, and hostGenerateInventory then sent "" for
// anything the prior state held that the configuration no longer set -- which
// is exactly that auto-populated set. Verified end to end on 8.0-trunk before
// the fix: a host with an inventory block naming `name`, the server having set
// `os` itself, planned `- os = "Linux 6.1 (auto-discovered)" -> null` and the
// apply wiped it on the server. With a live inventory_link item Zabbix
// repopulates and the fight is permanent.
//
// The existing automatic-mode coverage in acc_update_host_test.go could not
// have caught it: the provider exposes no `inventory_link`, so those fixtures
// run against an inventory nothing ever populates. Writing the field straight
// through the API is the same thing from the provider's point of view and
// needs no item, no data collection and no waiting.
//
// The step that matters is the second one, and it is an ordinary apply rather
// than a PlanOnly, because that catches both halves of the defect at once:
// wiping the field shows up in the server re-read, and a plan that proposes
// the deletion without the apply carrying it out shows up in the harness's own
// post-apply idempotency plan.

// testAccSetHostInventory writes inventory fields straight through the API,
// which is what an item with an inventory_link does under automatic mode --
// and what somebody editing the frontend does under either.
func testAccSetHostInventory(t *testing.T, id *string, fields map[string]string) func() {
	return func() {
		api, ok := testAccProvider.Meta().(*zabbix.API)
		if !ok || api == nil {
			t.Fatal("testAccSetHostInventory: provider not configured")
		}
		if _, err := api.CallWithError("host.update", zabbix.Params{
			"hostid":    *id,
			"inventory": fields,
		}); err != nil {
			t.Fatalf("testAccSetHostInventory(%s, %v): %s", *id, fields, err)
		}
	}
}

// testAccCheckServerInventory re-reads the host from Zabbix and compares the
// named inventory fields. Against the server, not against state: state is
// written by the provider's own read, so a clear the provider sent and should
// not have still looks right there (S9e).
func testAccCheckServerInventory(addr string, want map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		api, ok := testAccProvider.Meta().(*zabbix.API)
		if !ok || api == nil {
			return fmt.Errorf("testAccCheckServerInventory: provider not configured")
		}
		id, err := testAccStateID(s, addr)
		if err != nil {
			return err
		}
		obj, err := serverHost(api, id)
		if err != nil {
			return err
		}
		inv, _ := obj["inventory"].(map[string]interface{})
		for k, v := range want {
			got, _ := inv[k].(string)
			if got != v {
				return fmt.Errorf("%s inventory %q on the server is %q, want %q", addr, k, got, v)
			}
		}
		return nil
	}
}

func testAccHostInventoryHCL(body string) string {
	return `
resource "zabbix_hostgroup" "testinvgrp" {
	name = "test-inventory-host-group"
}
resource "zabbix_host" "testinv" {
	host   = "test-inventory-host"
	groups = [ zabbix_hostgroup.testinvgrp.id ]
	interface {
		ip = "127.0.0.1"
	}
` + body + `
}
`
}

func TestAccHostInventoryAutomatic(t *testing.T) {
	var id string

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // one field, managed by the configuration
				Config: testAccHostInventoryHCL(`
	inventory_mode = "automatic"
	inventory {
		name = "tf name"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCaptureID("zabbix_host.testinv", &id),
					resource.TestCheckResourceAttr("zabbix_host.testinv", "inventory.0.name", "tf name"),
					testAccCheckServerInventory("zabbix_host.testinv", map[string]string{
						"name": "tf name",
					}),
				),
			},
			{ // Zabbix populates a field of its own. The configuration has not
				// changed, so this apply must leave it exactly where it is --
				// and, because the harness plans again afterwards and requires
				// that plan to be empty, must not have proposed deleting it
				PreConfig: testAccSetHostInventory(t, &id, map[string]string{
					"os": "Linux 6.1 (auto-discovered)",
				}),
				Config: testAccHostInventoryHCL(`
	inventory_mode = "automatic"
	inventory {
		name = "tf name"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerInventory("zabbix_host.testinv", map[string]string{
						"os":   "Linux 6.1 (auto-discovered)",
						"name": "tf name",
					}),
					// and the field the provider does not manage is not in
					// state either, which is what keeps the plan empty
					resource.TestCheckResourceAttr("zabbix_host.testinv", "inventory.0.os", ""),
				),
			},
			{ // editing the field the configuration does own still works, and
				// still leaves the server's own field alone
				Config: testAccHostInventoryHCL(`
	inventory_mode = "automatic"
	inventory {
		name = "tf name 2"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerInventory("zabbix_host.testinv", map[string]string{
						"os":   "Linux 6.1 (auto-discovered)",
						"name": "tf name 2",
					}),
				),
			},
			{ // clearing a field the configuration still names: "" is a value
				// it gives rather than a line it omits, so it goes on the wire
				// and the field empties. This is what replaces line deletion
				// under automatic mode, and it has to work while the field is
				// still managed -- see the last step for why that matters
				Config: testAccHostInventoryHCL(`
	inventory_mode = "automatic"
	inventory {
		name = ""
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerInventory("zabbix_host.testinv", map[string]string{
						"os":   "Linux 6.1 (auto-discovered)",
						"name": "",
					}),
				),
			},
			{ // set again, so the next step has something to leave behind
				Config: testAccHostInventoryHCL(`
	inventory_mode = "automatic"
	inventory {
		name = "tf name 3"
	}
`),
				Check: testAccCheckServerInventory("zabbix_host.testinv", map[string]string{
					"name": "tf name 3",
				}),
			},
			{ // R1's one exception, asserted rather than assumed: under
				// automatic mode deleting a line leaves the field as it was,
				// because the provider cannot tell a field it set from a field
				// an item populated. The field also leaves state, and the plan
				// after this apply is still required to be empty -- which is
				// what makes the exception liveable rather than a diff that
				// reapplies for ever.
				//
				// The corner this creates, and it is documented on the
				// attribute: state now says "" while the server says "tf name
				// 3", so putting the line back as `name = ""` is not a change
				// Terraform can see and clears nothing. Writing any other value
				// is a change, which is the next step.
				Config: testAccHostInventoryHCL(`
	inventory_mode = "automatic"
	inventory {
		location = "tf location"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerInventory("zabbix_host.testinv", map[string]string{
						"os":       "Linux 6.1 (auto-discovered)",
						"name":     "tf name 3",
						"location": "tf location",
					}),
					resource.TestCheckResourceAttr("zabbix_host.testinv", "inventory.0.name", ""),
				),
			},
			{ // taking the field back into the configuration works: a value
				// that differs from the "" state holds is an ordinary diff
				Config: testAccHostInventoryHCL(`
	inventory_mode = "automatic"
	inventory {
		location = "tf location"
		name     = "tf name 4"
	}
`),
				Check: testAccCheckServerInventory("zabbix_host.testinv", map[string]string{
					"os":       "Linux 6.1 (auto-discovered)",
					"name":     "tf name 4",
					"location": "tf location",
				}),
			},
		},
	})
}

// TestAccHostInventoryManual is the other half of the same decision. Manual
// mode is unchanged: the configuration owns the whole inventory there, so a
// deleted line clears the field and a value written behind Terraform's back is
// drift to be corrected. Both are asserted against a server re-read, because
// state is written by the provider's own read and would agree either way.
func TestAccHostInventoryManual(t *testing.T) {
	var id string

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccHostInventoryHCL(`
	inventory_mode = "manual"
	inventory {
		name     = "manual name"
		location = "manual location"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCaptureID("zabbix_host.testinv", &id),
					testAccCheckServerInventory("zabbix_host.testinv", map[string]string{
						"name":     "manual name",
						"location": "manual location",
					}),
				),
			},
			{ // the line deleted -- and under manual mode that still clears
				Config: testAccHostInventoryHCL(`
	inventory_mode = "manual"
	inventory {
		location = "manual location"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerInventory("zabbix_host.testinv", map[string]string{
						"name":     "",
						"location": "manual location",
					}),
				),
			},
			{ // a field set outside Terraform under manual mode is drift, and
				// the next apply takes it back
				PreConfig: testAccSetHostInventory(t, &id, map[string]string{
					"os": "set in the frontend",
				}),
				Config: testAccHostInventoryHCL(`
	inventory_mode = "manual"
	inventory {
		location = "manual location"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerInventory("zabbix_host.testinv", map[string]string{
						"os":       "",
						"location": "manual location",
					}),
				),
			},
		},
	})
}
