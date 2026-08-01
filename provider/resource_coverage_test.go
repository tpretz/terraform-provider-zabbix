package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

// TestAccTemplategroup covers the zabbix_templategroup resource, which backs
// zabbix_template on Zabbix >= 6.2 where template groups were split out of
// host groups.
func TestAccTemplategroup(t *testing.T) {
	id := resource.UniqueId()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "zabbix_templategroup" "tg" {
  name = "acc-tg-%s"
}
`, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_templategroup.tg", "name", "acc-tg-"+id),
					resource.TestCheckResourceAttrSet("zabbix_templategroup.tg", "id"),
				),
			},
			{
				// rename, exercising update
				Config: fmt.Sprintf(`
resource "zabbix_templategroup" "tg" {
  name = "acc-tg-renamed-%s"
}
`, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_templategroup.tg", "name", "acc-tg-renamed-"+id),
				),
			},
		},
	})
}

// TestAccTemplate covers zabbix_template create/update against template groups.
func TestAccTemplate(t *testing.T) {
	id := resource.UniqueId()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccTemplateBase(id, "desc A"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.t", "host", "acc-tmpl-"+id),
					// name defaults to host, must be computed to stay idempotent
					resource.TestCheckResourceAttr("zabbix_template.t", "name", "acc-tmpl-"+id),
					resource.TestCheckResourceAttr("zabbix_template.t", "description", "desc A"),
					resource.TestCheckResourceAttr("zabbix_template.t", "groups.#", "1"),
				),
			},
			{
				Config: testAccTemplateBase(id, "desc B"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_template.t", "description", "desc B"),
				),
			},
		},
	})
}

func testAccTemplateBase(id, desc string) string {
	return fmt.Sprintf(`
resource "zabbix_templategroup" "g" {
  name = "acc-tmplgrp-%s"
}
resource "zabbix_template" "t" {
  host        = "acc-tmpl-%s"
  groups      = [zabbix_templategroup.g.id]
  description = "%s"
}
`, id, id, desc)
}

// TestAccItemFamilies exercises several item types in one config: agent,
// trapper, calculated and dependent. These share common_item.go helpers, so a
// serialization regression in one usually affects all.
func TestAccItemFamilies(t *testing.T) {
	id := resource.UniqueId()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccItemFamiliesConfig(id, "1m"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("zabbix_item_agent.agent", "id"),
					resource.TestCheckResourceAttr("zabbix_item_agent.agent", "delay", "1m"),
					resource.TestCheckResourceAttrSet("zabbix_item_trapper.trap", "id"),
					resource.TestCheckResourceAttrSet("zabbix_item_calculated.calc", "id"),
					resource.TestCheckResourceAttrSet("zabbix_item_dependent.dep", "id"),
				),
			},
			{
				// change delay to exercise item update
				Config: testAccItemFamiliesConfig(id, "2m"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_agent.agent", "delay", "2m"),
				),
			},
		},
	})
}

func testAccItemFamiliesConfig(id, delay string) string {
	return fmt.Sprintf(`
resource "zabbix_templategroup" "g" {
  name = "acc-items-grp-%s"
}
resource "zabbix_template" "t" {
  host   = "acc-items-tmpl-%s"
  groups = [zabbix_templategroup.g.id]
}

resource "zabbix_item_agent" "agent" {
  hostid    = zabbix_template.t.id
  key       = "acc.agent[%s]"
  name      = "acc agent %s"
  valuetype = "unsigned"
  delay     = "%s"
}

resource "zabbix_item_trapper" "trap" {
  hostid    = zabbix_template.t.id
  key       = "acc.trap[%s]"
  name      = "acc trap %s"
  valuetype = "float"
}

resource "zabbix_item_calculated" "calc" {
  hostid    = zabbix_template.t.id
  key       = "acc.calc[%s]"
  name      = "acc calc %s"
  valuetype = "float"
  formula   = "last(//acc.trap[%s])"
}

resource "zabbix_item_dependent" "dep" {
  hostid        = zabbix_template.t.id
  key           = "acc.dep[%s]"
  name          = "acc dep %s"
  valuetype     = "text"
  master_itemid = zabbix_item_trapper.trap.id
}
`, id, id, id, id, delay, id, id, id, id, id, id, id)
}

// TestAccItemPreprocessorAndTags covers the preprocessing and tag blocks on
// items, which are serialized through separate code paths.
func TestAccItemPreprocessorAndTags(t *testing.T) {
	id := resource.UniqueId()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "zabbix_templategroup" "g" {
  name = "acc-pre-grp-%s"
}
resource "zabbix_template" "t" {
  host   = "acc-pre-tmpl-%s"
  groups = [zabbix_templategroup.g.id]
}
resource "zabbix_item_trapper" "i" {
  hostid    = zabbix_template.t.id
  key       = "acc.pre[%s]"
  name      = "acc pre %s"
  valuetype = "float"

  preprocessor {
    type   = "1"
    params = ["8"]
  }

  tag {
    key   = "component"
    value = "network"
  }
}
`, id, id, id, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_trapper.i", "preprocessor.#", "1"),
					resource.TestCheckResourceAttr("zabbix_item_trapper.i", "preprocessor.0.type", "1"),
					resource.TestCheckResourceAttr("zabbix_item_trapper.i", "tag.#", "1"),
				),
			},
		},
	})
}

// TestAccTrigger covers zabbix_trigger including tags and priority.
func TestAccTrigger(t *testing.T) {
	id := resource.UniqueId()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccTriggerConfig(id, "warn"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("zabbix_trigger.tr", "id"),
					resource.TestCheckResourceAttr("zabbix_trigger.tr", "priority", "warn"),
					resource.TestCheckResourceAttr("zabbix_trigger.tr", "tag.#", "1"),
				),
			},
			{
				Config: testAccTriggerConfig(id, "high"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_trigger.tr", "priority", "high"),
				),
			},
		},
	})
}

func testAccTriggerConfig(id, priority string) string {
	return fmt.Sprintf(`
resource "zabbix_templategroup" "g" {
  name = "acc-trig-grp-%s"
}
resource "zabbix_template" "t" {
  host   = "acc-trig-tmpl-%s"
  groups = [zabbix_templategroup.g.id]
}
resource "zabbix_item_trapper" "i" {
  hostid    = zabbix_template.t.id
  key       = "acc.trig[%s]"
  name      = "acc trig item %s"
  valuetype = "unsigned"
}
resource "zabbix_trigger" "tr" {
  name       = "acc trigger %s"
  expression = "last(/${zabbix_template.t.host}/${zabbix_item_trapper.i.key})>10"
  priority   = "%s"
  comments   = "acceptance test trigger"

  tag {
    key   = "scope"
    value = "availability"
  }
}
`, id, id, id, id, id, priority)
}

// TestAccGraph covers zabbix_graph and its nested item blocks.
func TestAccGraph(t *testing.T) {
	id := resource.UniqueId()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccGraphConfig(id, "200"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("zabbix_graph.g", "id"),
					resource.TestCheckResourceAttr("zabbix_graph.g", "height", "200"),
					resource.TestCheckResourceAttr("zabbix_graph.g", "item.#", "1"),
					resource.TestCheckResourceAttr("zabbix_graph.g", "item.0.color", "FF0000"),
				),
			},
			{
				Config: testAccGraphConfig(id, "300"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_graph.g", "height", "300"),
				),
			},
		},
	})
}

func testAccGraphConfig(id, height string) string {
	return fmt.Sprintf(`
resource "zabbix_templategroup" "g" {
  name = "acc-graph-grp-%s"
}
resource "zabbix_template" "t" {
  host   = "acc-graph-tmpl-%s"
  groups = [zabbix_templategroup.g.id]
}
resource "zabbix_item_trapper" "i" {
  hostid    = zabbix_template.t.id
  key       = "acc.graph[%s]"
  name      = "acc graph item %s"
  valuetype = "unsigned"
}
resource "zabbix_graph" "g" {
  name   = "acc graph %s"
  height = "%s"
  width  = "900"

  item {
    itemid = zabbix_item_trapper.i.id
    color  = "FF0000"
  }
}
`, id, id, id, id, id, height)
}

// TestAccLLDAndPrototypes covers an LLD rule plus an item prototype hanging off
// it, exercising common_lld.go and the prototype code paths.
func TestAccLLDAndPrototypes(t *testing.T) {
	id := resource.UniqueId()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLLDConfig(id, "30d"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("zabbix_lld_trapper.lld", "id"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.lld", "lifetime", "30d"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.lld", "condition.#", "1"),
					resource.TestCheckResourceAttrSet("zabbix_proto_item_trapper.proto", "id"),
				),
			},
			{
				Config: testAccLLDConfig(id, "60d"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.lld", "lifetime", "60d"),
				),
			},
		},
	})
}

func testAccLLDConfig(id, lifetime string) string {
	return fmt.Sprintf(`
resource "zabbix_templategroup" "g" {
  name = "acc-lld-grp-%s"
}
resource "zabbix_template" "t" {
  host   = "acc-lld-tmpl-%s"
  groups = [zabbix_templategroup.g.id]
}
resource "zabbix_lld_trapper" "lld" {
  hostid   = zabbix_template.t.id
  key      = "acc.lld[%s]"
  name     = "acc lld %s"
  lifetime = "%s"

  condition {
    macro = "{#NAME}"
    value = "^acc"
  }
}
resource "zabbix_proto_item_trapper" "proto" {
  hostid    = zabbix_template.t.id
  ruleid    = zabbix_lld_trapper.lld.id
  key       = "acc.proto[%s,{#NAME}]"
  name      = "acc proto {#NAME}"
  valuetype = "unsigned"
}
`, id, id, id, id, lifetime, id)
}

// TestAccHostgroupAndDataSources covers zabbix_hostgroup plus the hostgroup and
// template data sources resolving by name.
func TestAccHostgroupAndDataSources(t *testing.T) {
	id := resource.UniqueId()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "hg" {
  name = "acc-hg-%s"
}
resource "zabbix_templategroup" "tg" {
  name = "acc-ds-tg-%s"
}
resource "zabbix_template" "t" {
  host   = "acc-ds-tmpl-%s"
  groups = [zabbix_templategroup.tg.id]
}

data "zabbix_hostgroup" "found" {
  name = zabbix_hostgroup.hg.name
}
data "zabbix_templategroup" "found" {
  name = zabbix_templategroup.tg.name
}
data "zabbix_template" "found" {
  host = zabbix_template.t.host
}
`, id, id, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.zabbix_hostgroup.found", "id", "zabbix_hostgroup.hg", "id"),
					resource.TestCheckResourceAttrPair(
						"data.zabbix_templategroup.found", "id", "zabbix_templategroup.tg", "id"),
					resource.TestCheckResourceAttrPair(
						"data.zabbix_template.found", "id", "zabbix_template.t", "id"),
				),
			},
		},
	})
}

// TestAccHostWithMacrosAndTemplates covers host macros and template linkage,
// which go through separate serialization paths from the basic host fields.
func TestAccHostWithMacrosAndTemplates(t *testing.T) {
	id := resource.UniqueId()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "g" {
  name = "acc-hm-grp-%s"
}
resource "zabbix_templategroup" "tg" {
  name = "acc-hm-tmplgrp-%s"
}
resource "zabbix_template" "t" {
  host   = "acc-hm-tmpl-%s"
  groups = [zabbix_templategroup.tg.id]
}
resource "zabbix_host" "h" {
  host      = "acc-hm-host-%s"
  groups    = [zabbix_hostgroup.g.id]
  templates = [zabbix_template.t.id]

  interface {
    type = "agent"
    ip   = "127.0.0.1"
    port = 10050
    main = true
  }

  macro {
    name  = "{$ACC_MACRO}"
    value = "acc-value"
  }
}
`, id, id, id, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.h", "macro.#", "1"),
					resource.TestCheckResourceAttr("zabbix_host.h", "templates.#", "1"),
				),
			},
		},
	})
}
