package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// skipWithoutTemplateGroups aborts the calling test on servers older than 6.2,
// which have no templategroup API at all.
func skipWithoutTemplateGroups(t *testing.T) {
	if !testAccTemplateGroups(t) {
		t.Skipf("template groups require Zabbix 6.2 or later (server reports %d)", testAccVersion(t))
	}
}

func TestAccResourceTemplategroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipWithoutTemplateGroups(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{ // simple create
				Config: `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-templategroup"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_templategroup.testtmplgrp", "name", "test-templategroup"),
				),
			},
			{ // rename
				Config: `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-templategroup-renamed"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_templategroup.testtmplgrp", "name", "test-templategroup-renamed"),
				),
			},
			{ // import
				ResourceName:      "zabbix_templategroup.testtmplgrp",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{ // usable as a zabbix_template group
				Config: `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-templategroup-renamed"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-templategroup-template"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.testtmpl", "groups.#", "1"),
					resource.TestCheckResourceAttrPair(
						"zabbix_template.testtmpl", "groups.0",
						"zabbix_templategroup.testtmplgrp", "id"),
				),
			},
		},
	})
}

func TestAccDataSourceTemplategroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipWithoutTemplateGroups(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-templategroup-data"
}
data "zabbix_templategroup" "lookup" {
	name = zabbix_templategroup.testtmplgrp.name
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.zabbix_templategroup.lookup", "name", "test-templategroup-data"),
					resource.TestCheckResourceAttrPair(
						"data.zabbix_templategroup.lookup", "id",
						"zabbix_templategroup.testtmplgrp", "id"),
				),
			},
		},
	})
}

// TestAccResourceTemplategroupUnsupported checks the other side of the gate:
// on 6.0/6.1 the resource must refuse with a message that names the
// alternative, rather than failing deep inside an API call.
func TestAccResourceTemplategroupUnsupported(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if testAccTemplateGroups(t) {
				t.Skip("server has template groups; nothing to reject")
			}
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-templategroup"
}
`,
				ExpectError: regexp.MustCompile("requires Zabbix 6.2 or later"),
			},
		},
	})
}

// TestAccTemplateStateUpgradeV0 exercises the zabbix_template v0->v1 state
// upgrader against a live server: a template group id passes through untouched,
// a host group id (what v0.17.0 state actually contains) is rejected with an
// actionable error rather than silently rewritten.
func TestAccTemplateStateUpgradeV0(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipWithoutTemplateGroups(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-upgrade-hostgroup"
}
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-upgrade-templategroup"
}
`,
				Check: func(s *terraform.State) error {
					meta := testAccProvider.Meta()
					hostGroupID := s.RootModule().Resources["zabbix_hostgroup.testgrp"].Primary.ID
					tmplGroupID := s.RootModule().Resources["zabbix_templategroup.testtmplgrp"].Primary.ID

					// already-correct ids survive untouched
					good := map[string]interface{}{
						"host":   "test-template",
						"groups": []interface{}{tmplGroupID},
					}
					out, err := resourceTemplateStateUpgradeV0(context.Background(), good, meta)
					if err != nil {
						return err
					}
					if len(stateStringList(out["groups"])) != 1 || stateStringList(out["groups"])[0] != tmplGroupID {
						return fmt.Errorf("template group id should have passed through unchanged, got %v", out["groups"])
					}

					// host group ids must fail loudly
					bad := map[string]interface{}{
						"host":   "test-template",
						"groups": []interface{}{hostGroupID},
					}
					_, err = resourceTemplateStateUpgradeV0(context.Background(), bad, meta)
					if err == nil {
						return fmt.Errorf("host group id %s should have been rejected", hostGroupID)
					}
					for _, want := range []string{"test-template", hostGroupID, "test-upgrade-hostgroup", "zabbix_templategroup"} {
						if !strings.Contains(err.Error(), want) {
							return fmt.Errorf("upgrade error should mention %q, got: %s", want, err)
						}
					}
					return nil
				},
			},
		},
	})
}

// TestTemplateStateUpgradeV0Offline covers the branches that need no server.
func TestTemplateStateUpgradeV0Offline(t *testing.T) {
	raw := map[string]interface{}{
		"host":   "test-template",
		"groups": []interface{}{"13"},
	}

	// provider not configured: leave state alone rather than guess
	out, err := resourceTemplateStateUpgradeV0(context.Background(), raw, nil)
	if err != nil {
		t.Fatalf("nil meta should not error: %s", err)
	}
	if len(stateStringList(out["groups"])) != 1 {
		t.Fatalf("nil meta should pass state through, got %v", out["groups"])
	}

	// pre-6.2 server: host group ids are still correct, nothing to do
	api := &zabbix.API{Config: zabbix.Config{Version: 60000}}
	out, err = resourceTemplateStateUpgradeV0(context.Background(), raw, api)
	if err != nil {
		t.Fatalf("pre-6.2 should not error: %s", err)
	}
	if got := stateStringList(out["groups"]); len(got) != 1 || got[0] != "13" {
		t.Fatalf("pre-6.2 should pass state through, got %v", got)
	}

	if got := templateStateName(map[string]interface{}{"host": "bob"}); got != `"bob"` {
		t.Fatalf("templateStateName host: %s", got)
	}
	if got := templateStateName(map[string]interface{}{"id": "42"}); got != "id 42" {
		t.Fatalf("templateStateName id: %s", got)
	}
	if got := templateStateName(map[string]interface{}{}); got != "(unnamed)" {
		t.Fatalf("templateStateName empty: %s", got)
	}
}
