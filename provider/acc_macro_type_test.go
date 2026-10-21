package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// Macro `type` and `description` (common_macro.go), owned here for U1/U2 and
// R1. macroSetSchema's Elem is shared by zabbix_host and zabbix_template, so
// one resource covers both; the host is the one tested because it is where a
// secret macro is most often wanted.
//
// What makes `type` more than another enum is `secret`. Probed on 6.0.48,
// 7.0.29, 7.4.13 and 8.0-trunk with host.create and then host.get
// selectMacros=extend:
//
//	text     value returned
//	secret   "value" omitted entirely; "type" is "1"
//	vault    value (the vault path) returned; "path:key" syntax enforced
//	         by the server, "incorrect syntax near ..." otherwise
//
// and on all four, host.update changes a macro's type in place, keeping its
// hostmacroid, in both directions. So the read path has to carry a secret's
// value over from what is already in state, or every plan after the first
// would propose to set it again. The empty plan Terraform's test harness runs
// after every step is the check that it does.

// testAccMacroTypeHCL is a host carrying the given macro blocks.
func testAccMacroTypeHCL(macros string) string {
	return `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-macro-type-group"
}
resource "zabbix_host" "testhost" {
	host   = "test-macro-type-host"
	groups = [ zabbix_hostgroup.testgrp.id ]
` + macros + `
}
`
}

// testAccCheckMacroServer asserts one macro, found by name, as the server
// holds it. A secret's value is not in the want map because the server never
// returns it -- which testAccCheckMacroSecretHidden asserts separately.
func testAccCheckMacroServer(addr, name string, want map[string]string) resource.TestCheckFunc {
	return testAccCheckServerElem(addr, serverHost, "macros", "macro", name, want)
}

// testAccCheckMacroSecretHidden pins the premise the read path is built on:
// the server does not hand a secret macro's value back. If a future version
// starts returning it, this fails and the carry-over in flattenMacros can be
// revisited rather than silently masking a value the server now reports.
func testAccCheckMacroSecretHidden(addr, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		api, ok := testAccProvider.Meta().(*zabbix.API)
		if !ok || api == nil {
			return fmt.Errorf("testAccCheckMacroSecretHidden: provider not configured")
		}
		id, err := testAccStateID(s, addr)
		if err != nil {
			return err
		}
		obj, err := serverHost(api, id)
		if err != nil {
			return err
		}
		macros, _ := obj["macros"].([]interface{})
		for _, raw := range macros {
			m, _ := raw.(map[string]interface{})
			if m["macro"] != name {
				continue
			}
			if v, present := m["value"]; present {
				return fmt.Errorf("%s: server returned value %q for secret macro %s", addr, v, name)
			}
			return nil
		}
		return fmt.Errorf("%s: server has no macro %s", addr, name)
	}
}

// testAccMacroImportCheck asserts what an import of a host with a secret
// macro produces: the secret arrives with its type and an empty value, since
// the server has nothing else to give, and the other macros arrive whole.
// ImportStateVerify cannot express that -- the secret element's value, and so
// its set hash, legitimately differs from the applied state.
func testAccMacroImportCheck(want map[string]map[string]string) resource.ImportStateCheckFunc {
	return func(states []*terraform.InstanceState) error {
		if len(states) != 1 {
			return fmt.Errorf("imported %d states, want 1", len(states))
		}
		attrs := states[0].Attributes

		got := map[string]map[string]string{}
		for k, v := range attrs {
			rest, ok := strings.CutPrefix(k, "macro.")
			if !ok || rest == "#" {
				continue
			}
			idx, field, ok := strings.Cut(rest, ".")
			if !ok {
				continue
			}
			if got[idx] == nil {
				got[idx] = map[string]string{}
			}
			got[idx][field] = v
		}

		byName := map[string]map[string]string{}
		for _, m := range got {
			byName[m["name"]] = m
		}
		if len(byName) != len(want) {
			return fmt.Errorf("imported %d macros, want %d: %v", len(byName), len(want), byName)
		}
		for name, fields := range want {
			m, ok := byName[name]
			if !ok {
				return fmt.Errorf("imported state has no macro %s: %v", name, byName)
			}
			for f, v := range fields {
				if m[f] != v {
					return fmt.Errorf("imported macro %s: %s is %q, want %q", name, f, m[f], v)
				}
			}
		}
		return nil
	}
}

// TestAccMacroType owns macro `type` and `description`: create with all three
// types, change both attributes in life (U1, U2), import, catch a type changed
// outside Terraform, then delete the lines again (R1 for `type`, the S9d clear
// for `description`).
func TestAccMacroType(t *testing.T) {
	const addr = "zabbix_host.testhost"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // all three types, each with a description
				Config: testAccMacroTypeHCL(`
	macro {
		name        = "{$MT_TEXT}"
		value       = "text-a"
		description = "text macro"
	}
	macro {
		name        = "{$MT_SECRET}"
		value       = "secret-a"
		type        = "secret"
		description = "secret macro"
	}
	macro {
		name        = "{$MT_VAULT}"
		value       = "secret/zabbix:password"
		type        = "vault"
		description = "vault macro"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "macro.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs(addr, "macro.*", map[string]string{
						"name": "{$MT_SECRET}", "value": "secret-a", "type": "secret", "description": "secret macro",
					}),
					testAccCheckMacroServer(addr, "{$MT_TEXT}", map[string]string{
						"type": "0", "value": "text-a", "description": "text macro",
					}),
					testAccCheckMacroServer(addr, "{$MT_SECRET}", map[string]string{
						"type": "1", "description": "secret macro",
					}),
					testAccCheckMacroSecretHidden(addr, "{$MT_SECRET}"),
					testAccCheckMacroServer(addr, "{$MT_VAULT}", map[string]string{
						"type": "2", "value": "secret/zabbix:password", "description": "vault macro",
					}),
				),
			},
			{ // U1/U2: text becomes secret, secret becomes text, a secret's
				// value and every description change -- all in place
				Config: testAccMacroTypeHCL(`
	macro {
		name        = "{$MT_TEXT}"
		value       = "text-b"
		type        = "secret"
		description = "now secret"
	}
	macro {
		name        = "{$MT_SECRET}"
		value       = "secret-b"
		type        = "text"
		description = "now text"
	}
	macro {
		name        = "{$MT_VAULT}"
		value       = "secret/zabbix:token"
		type        = "vault"
		description = "vault changed"
	}
`),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMacroServer(addr, "{$MT_TEXT}", map[string]string{
						"type": "1", "description": "now secret",
					}),
					testAccCheckMacroSecretHidden(addr, "{$MT_TEXT}"),
					testAccCheckMacroServer(addr, "{$MT_SECRET}", map[string]string{
						"type": "0", "value": "secret-b", "description": "now text",
					}),
					testAccCheckMacroServer(addr, "{$MT_VAULT}", map[string]string{
						"type": "2", "value": "secret/zabbix:token", "description": "vault changed",
					}),
				),
			},
			{ // an import knows the secret is a secret, and nothing more
				ResourceName: addr,
				ImportState:  true,
				ImportStateCheck: testAccMacroImportCheck(map[string]map[string]string{
					"{$MT_TEXT}":   {"type": "secret", "value": "", "description": "now secret"},
					"{$MT_SECRET}": {"type": "text", "value": "secret-b", "description": "now text"},
					"{$MT_VAULT}":  {"type": "vault", "value": "secret/zabbix:token", "description": "vault changed"},
				}),
			},
			{ // a type changed outside Terraform is drift: the text macro
				// turned secret has no secret in state to inherit a value
				// from, so it reads back empty and the plan corrects it
				PreConfig: func() {
					api := testAccProvider.Meta().(*zabbix.API)
					hosts, err := api.HostsGet(zabbix.Params{
						"filter":       map[string]interface{}{"host": "test-macro-type-host"},
						"selectMacros": "extend",
					})
					if err != nil || len(hosts) != 1 {
						t.Fatalf("finding test-macro-type-host: %v (%d hosts)", err, len(hosts))
					}
					for _, m := range hosts[0].UserMacros {
						if m.MacroName == "{$MT_SECRET}" {
							if _, err := api.CallWithError("usermacro.update", zabbix.Params{
								"hostmacroid": m.MacroID, "type": "1",
							}); err != nil {
								t.Fatalf("usermacro.update: %s", err)
							}
							return
						}
					}
					t.Fatal("test-macro-type-host has no {$MT_SECRET}")
				},
				Config: testAccMacroTypeHCL(`
	macro {
		name        = "{$MT_TEXT}"
		value       = "text-b"
		type        = "secret"
		description = "now secret"
	}
	macro {
		name        = "{$MT_SECRET}"
		value       = "secret-b"
		type        = "text"
		description = "now text"
	}
	macro {
		name        = "{$MT_VAULT}"
		value       = "secret/zabbix:token"
		type        = "vault"
		description = "vault changed"
	}
`),
				ConfigPlanChecks: expectUpdate(addr),
				Check: testAccCheckMacroServer(addr, "{$MT_SECRET}", map[string]string{
					"type": "0", "value": "secret-b",
				}),
			},
			{ // R1 and S9d: deleting `type` returns the secret to text, and
				// deleting `description` clears it -- both reaching the server
				Config: testAccMacroTypeHCL(`
	macro {
		name  = "{$MT_TEXT}"
		value = "text-c"
	}
	macro {
		name  = "{$MT_SECRET}"
		value = "secret-b"
	}
	macro {
		name  = "{$MT_VAULT}"
		value = "secret/zabbix:token"
		type  = "vault"
	}
`),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs(addr, "macro.*", map[string]string{
						"name": "{$MT_TEXT}", "value": "text-c", "type": "text", "description": "",
					}),
					testAccCheckMacroServer(addr, "{$MT_TEXT}", map[string]string{
						"type": "0", "value": "text-c", "description": "",
					}),
					testAccCheckMacroServer(addr, "{$MT_SECRET}", map[string]string{
						"type": "0", "value": "secret-b", "description": "",
					}),
					testAccCheckMacroServer(addr, "{$MT_VAULT}", map[string]string{
						"type": "2", "description": "",
					}),
				),
			},
		},
	})
}
