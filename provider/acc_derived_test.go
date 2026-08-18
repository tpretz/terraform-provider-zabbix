package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// Derived defaults -- the other half of R2 (acc_removal_test.go).
//
// R2 asks what deleting the line does to an `Optional + Computed` attribute,
// and the answer for all five is "nothing, deliberately". This file asks the
// question that comes before it: **when the configuration never mentioned the
// attribute at all, where does its value come from, and when can the user see
// it?**
//
// Left to the flag alone the answer is "from the server, after apply". The
// plan says `(known after apply)` for values that are a pure function of
// something written three lines up, and the generated documentation can only
// say the server decides. Deriving them in `CustomizeDiff` fixes both -- and
// is exactly the change R2 refused for `zabbix_host.name`, because a
// derivation that runs on every plan overwrites the display name of every host
// imported into a configuration that does not manage it.
//
// Both halves are true, so the rule is the firing condition rather than the
// derivation:
//
//	attribute                    derived when                       clobber risk
//	---------------------------  ---------------------------------  ------------
//	zabbix_host.name             create; a `host` rename while the  none: the only
//	zabbix_template.name         stored name *is* the old `host`    value overwritten
//	                                                                is one the
//	                                                                derivation put there
//	trigger correlation_mode     create only                        none: no prior state
//	item trends                  create; a `valuetype` change       none: derived from
//	                             across the numeric boundary        config, not from
//	                                                                the server
//	host interface port          never -- see the note at the end
//
// Every test here therefore has three parts: the derived value is asserted
// **in the plan** (plancheck.ExpectKnownValue, which fails if it is unknown),
// against the server afterwards, and then a value the user owns is put in its
// way to prove the derivation steps aside.

// TestAccDerivedHostName -- `name` defaults to `host`, and the plan says so.
//
// The third and fourth steps are the ones that matter. A rename is followed
// only while the stored display name is exactly the old technical name; once
// it says anything else it is the user's, and renaming `host` underneath it
// leaves it alone. That is the case R2 recorded as the reason not to derive
// this attribute at all, asserted here rather than argued.
func TestAccDerivedHostName(t *testing.T) {
	const addr = "zabbix_host.testderivnamehost"

	host := func(technical, body string) string {
		return `
resource "zabbix_hostgroup" "testderivnamegrp" {
	name = "test-derived-name-group"
}
resource "zabbix_host" "testderivnamehost" {
	host   = "` + technical + `"
	groups = [ zabbix_hostgroup.testderivnamegrp.id ]
	interface {
		ip = "127.0.0.1"
	}
` + body + `
}
`
	}

	planned := func(want string) resource.ConfigPlanChecks {
		return resource.ConfigPlanChecks{
			PreApply: []plancheck.PlanCheck{
				plancheck.ExpectKnownValue(addr, tfjsonpath.New("name"), knownvalue.StringExact(want)),
			},
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // create: the derived name is in the plan, not "(known after apply)"
				Config:           host("test-derived-name-host", ``),
				ConfigPlanChecks: planned("test-derived-name-host"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", "test-derived-name-host"),
					testAccCheckServerAttrs(addr, serverHost, map[string]string{
						"host": "test-derived-name-host",
						"name": "test-derived-name-host",
					}),
				),
			},
			{
				Config:   host("test-derived-name-host", ``),
				PlanOnly: true,
			},
			{ // renaming `host` takes the derived name with it, rather than
				// leaving the frontend showing a name the host no longer has
				Config: host("test-derived-name-host-two", ``),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: append(
						planned("test-derived-name-host-two").PreApply,
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", "test-derived-name-host-two"),
					testAccCheckServerAttrs(addr, serverHost, map[string]string{
						"host": "test-derived-name-host-two",
						"name": "test-derived-name-host-two",
					}),
				),
			},
			{
				Config:   host("test-derived-name-host-two", ``),
				PlanOnly: true,
			},
			{ // a display name of the user's own, then the line deleted: R2's
				// stickiness, and the state the no-clobber case starts from
				Config:           host("test-derived-name-host-two", `	name = "Kept By Hand"`),
				ConfigPlanChecks: expectUpdate(addr),
				Check: testAccCheckServerAttrs(addr, serverHost, map[string]string{
					"name": "Kept By Hand",
				}),
			},
			{
				Config:   host("test-derived-name-host-two", ``),
				PlanOnly: true,
			},
			{ // and now the rename that must NOT move it: this is the imported
				// host with a display name somebody chose, in a configuration
				// that does not manage it
				Config:           host("test-derived-name-host-three", ``),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", "Kept By Hand"),
					testAccCheckServerAttrs(addr, serverHost, map[string]string{
						"host": "test-derived-name-host-three",
						"name": "Kept By Hand",
					}),
				),
			},
			{
				Config:   host("test-derived-name-host-three", ``),
				PlanOnly: true,
			},
		},
	})
}

// TestAccDerivedTemplateName -- the same rule on the same underlying object; a
// Zabbix template is a host with status 3. Covered separately because the two
// resources declare `name` separately, and because the servers do not agree
// with each other here: host.update overwrites the display name when `host`
// changes and `name` is omitted, template.update does not. The provider sends
// `name` on every write, so neither behaviour ever showed, and it applies its
// own conservative rule to both.
func TestAccDerivedTemplateName(t *testing.T) {
	const addr = "zabbix_template.testderivnametmpl"

	tmpl := func(technical, body string) string {
		return hcl(t, `
resource "zabbix_templategroup" "testderivnametmplgrp" {
	name = "test-derived-name-template-group"
}
resource "zabbix_template" "testderivnametmpl" {
	groups = [ zabbix_templategroup.testderivnametmplgrp.id ]
	host   = "`+technical+`"
`+body+`
}
`)
	}

	planned := func(want string) []plancheck.PlanCheck {
		return []plancheck.PlanCheck{
			plancheck.ExpectKnownValue(addr, tfjsonpath.New("name"), knownvalue.StringExact(want)),
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config:           tmpl("test-derived-name-template", ``),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: planned("test-derived-name-template")},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", "test-derived-name-template"),
					testAccCheckServerAttrs(addr, serverTemplate, map[string]string{
						"host": "test-derived-name-template",
						"name": "test-derived-name-template",
					}),
				),
			},
			{
				Config:   tmpl("test-derived-name-template", ``),
				PlanOnly: true,
			},
			{
				Config: tmpl("test-derived-name-template-two", ``),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: append(planned("test-derived-name-template-two"),
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate)),
				},
				Check: testAccCheckServerAttrs(addr, serverTemplate, map[string]string{
					"host": "test-derived-name-template-two",
					"name": "test-derived-name-template-two",
				}),
			},
			{
				Config:           tmpl("test-derived-name-template-two", `	name = "Kept By Hand"`),
				ConfigPlanChecks: expectUpdate(addr),
				Check: testAccCheckServerAttrs(addr, serverTemplate, map[string]string{
					"name": "Kept By Hand",
				}),
			},
			{ // the line deleted, then the technical name changed underneath it
				Config:   tmpl("test-derived-name-template-two", ``),
				PlanOnly: true,
			},
			{
				Config:           tmpl("test-derived-name-template-three", ``),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", "Kept By Hand"),
					testAccCheckServerAttrs(addr, serverTemplate, map[string]string{
						"host": "test-derived-name-template-three",
						"name": "Kept By Hand",
					}),
				),
			},
			{
				Config:   tmpl("test-derived-name-template-three", ``),
				PlanOnly: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// zabbix_host.interface.port -- the one that is NOT derived, and why
// ---------------------------------------------------------------------------
//
// The interface block's `port` looks like the best candidate of the five: the
// default is a pure function of `type` (agent 10050, snmp 161, ipmi 623, jmx
// 8686), hostInterfacePort already resolves it before the provider sends
// anything, and nothing about it depends on server state. A plan saying
// `port = (known after apply)` for a value the provider itself is about to
// compute is exactly what CustomizeDiff is for.
//
// It cannot be done, and the reason is mechanical rather than a judgement.
// `port` lives inside a TypeSet element, and ResourceDiff.SetNew is a
// **top-level** operation: it calls checkKey(key, "SetNew", false), which
// looks the name up in d.schema directly rather than walking the address
// (helper/schema/resource_diff.go), so "interface.0.port" comes back as
// `SetNew: invalid key`. The only settable name is the whole `interface`
// attribute -- and checkKey's second rule is that the key must be Computed,
// which `interface` is not, so that fails too:
//
//	SetNew only operates on computed keys - interface is not one
//
// Making the set Computed to get around that would be a much worse trade: an
// Optional+Computed collection can never be emptied (C1/C6 --
// acc_collection_test.go), so a host could no longer have its last interface
// removed. Rebuilding the set element by element would also mean inventing the
// server-assigned `id` of each element, unknown at plan time on create, and
// reimplementing hostInterfaceHash's normalisation; a mistake there does not
// misreport a port, it proposes replacing every interface on every plan, and
// Zabbix refuses to delete an interface that items are bound to.
//
// Nothing is lost by leaving it. The port the provider sends is deterministic,
// documented per type in the attribute's own description, and *removal already
// reverts* -- hostInterfaceHash normalises an absent port to the type default
// before hashing, so deleting the line behaves like an R1 attribute, which is
// TestAccRemoveHostInterfacePort. What remains unshown is one line of plan
// output on create.
