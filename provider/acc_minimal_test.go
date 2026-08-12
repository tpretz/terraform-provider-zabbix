package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// Minimal configurations -- one per registered resource, setting *only* the
// attributes the generated documentation marks Required.
//
// This is the config a user writes on day one, and it is the one shape the
// rest of the suite never exercised: every other fixture was written by
// someone who already knew which optional attributes had to be present, so it
// set them, and the resource's behaviour without them went untested for
// years. Four defects came out of writing these:
//
//   - zabbix_lld_trapper could not be created at all unless the config said
//     delay = "0". Every LLD fixture in the suite hardcoded it.
//   - a preprocessor block that omitted error_handler failed on create on
//     every supported version: the schema default was "" and Zabbix rejects
//     that outright.
//   - zabbix_host rejected a host with no interface block, which every
//     supported server accepts.
//   - HTTP item posts/proxy and trigger url could be set but never cleared.
//
// Each test applies the minimum and then re-plans. The apply proves the
// create path does not need more than the docs promise; the empty plan proves
// every schema default -- which is what the omitted attributes become -- is a
// value the server round-trips unchanged. A default the server rewrites shows
// up here as a permanent diff and nowhere else.
//
// Where a resource needs a prerequisite object (an item to graph, a rule to
// hang a prototype from) that prerequisite is itself minimal, so the fixtures
// double as a check that the minimum composes.

// minimalTemplateHCL is the owner every item, rule, prototype, graph and
// trigger below hangs from. A template rather than a host, deliberately:
// template items need no interface, so nothing here smuggles in an attribute
// the resource under test would otherwise have to declare.
const minimalTemplateHCL = `
resource "zabbix_templategroup" "testminimalgrp" {
	name = "test-minimal-template-group"
}
resource "zabbix_template" "testminimaltmpl" {
	groups = [ zabbix_templategroup.testminimalgrp.id ]
	host   = "test-minimal-template"
}
`

// TestAccMinimalHostgroup -- name is the only required attribute.
func TestAccMinimalHostgroup(t *testing.T) {
	config := `
resource "zabbix_hostgroup" "testminimal" {
	name = "test-minimal-hostgroup"
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{Config: config},
			{Config: config, PlanOnly: true},
		},
	})
}

// TestAccMinimalTemplategroup -- 6.2+, where template groups became their own
// object. Not run through hcl(): the point is the templategroup resource
// itself, so rewriting it to a hostgroup below 6.2 would test nothing.
func TestAccMinimalTemplategroup(t *testing.T) {
	config := `
resource "zabbix_templategroup" "testminimal" {
	name = "test-minimal-templategroup"
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{Config: config, SkipFunc: skipBelow(t, zabbix.V62)},
			{Config: config, SkipFunc: skipBelow(t, zabbix.V62), PlanOnly: true},
		},
	})
}

// TestAccMinimalHost -- host and groups. No interface block: Zabbix has never
// required one, though this provider did until v2.
func TestAccMinimalHost(t *testing.T) {
	config := `
resource "zabbix_hostgroup" "testminimalhostgrp" {
	name = "test-minimal-host-group"
}
resource "zabbix_host" "testminimal" {
	host   = "test-minimal-host"
	groups = [ zabbix_hostgroup.testminimalhostgrp.id ]
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_host.testminimal", "interface.#", "0"),
				),
			},
			{Config: config, PlanOnly: true},
		},
	})
}

// TestAccMinimalTemplate -- host and groups.
func TestAccMinimalTemplate(t *testing.T) {
	config := hcl(t, minimalTemplateHCL)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{Config: config},
			{Config: config, PlanOnly: true},
		},
	})
}

// TestAccMinimalProxy -- name alone. Everything the proxy model needs beyond
// that (operating_mode, and the address/port pair a passive proxy requires)
// has to come from a default, or the resource is not usable as documented.
func TestAccMinimalProxy(t *testing.T) {
	config := `
resource "zabbix_proxy" "testminimal" {
	name = "test-minimal-proxy"
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{Config: config},
			{Config: config, PlanOnly: true},
		},
	})
}

// minimalItemsHCL is all ten item backend types at their documented minimum,
// in one configuration. One apply rather than ten because that is also how a
// user meets them -- and because a create that needs an attribute the docs
// call optional fails the same way either way.
const minimalItemsHCL = `
resource "zabbix_item_trapper" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	key       = "test.minimal.trapper"
	name      = "Test Minimal Trapper"
	valuetype = "unsigned"
}
resource "zabbix_item_agent" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	key       = "test.minimal.agent"
	name      = "Test Minimal Agent"
	valuetype = "unsigned"
}
resource "zabbix_item_simple" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	key       = "test.minimal.simple"
	name      = "Test Minimal Simple"
	valuetype = "unsigned"
}
resource "zabbix_item_external" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	key       = "test.minimal.external"
	name      = "Test Minimal External"
	valuetype = "unsigned"
}
resource "zabbix_item_internal" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	key       = "test.minimal.internal"
	name      = "Test Minimal Internal"
	valuetype = "unsigned"
}
resource "zabbix_item_snmptrap" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	key       = "snmptrap.fallback"
	name      = "Test Minimal Snmptrap"
	valuetype = "text"
}
resource "zabbix_item_snmp" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	key       = "test.minimal.snmp"
	name      = "Test Minimal Snmp"
	valuetype = "unsigned"
	snmp_oid  = "1.3.6.1.2.1.1.1.0"
}
resource "zabbix_item_http" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	key       = "test.minimal.http"
	name      = "Test Minimal Http"
	valuetype = "text"
	url       = "http://localhost/minimal"
}
resource "zabbix_item_calculated" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	key       = "test.minimal.calculated"
	name      = "Test Minimal Calculated"
	valuetype = "unsigned"
	formula   = "last(/test-minimal-template/test.minimal.trapper)"
}
resource "zabbix_item_dependent" "testminimal" {
	hostid        = zabbix_template.testminimaltmpl.id
	key           = "test.minimal.dependent"
	name          = "Test Minimal Dependent"
	valuetype     = "unsigned"
	master_itemid = zabbix_item_trapper.testminimal.id
}
`

func TestAccMinimalItems(t *testing.T) {
	config := hcl(t, minimalTemplateHCL+minimalItemsHCL)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{Config: config},
			{Config: config, PlanOnly: true},
		},
	})
}

// minimalLLDsHCL is all eight discovery-rule types at their minimum. Note
// that neither zabbix_lld_trapper nor zabbix_lld_dependent sets delay: those
// two are the rules Zabbix polls never, their delay is pinned to "0", and
// having to write it out was a real defect (the attribute exists only so the
// shared read path stays total).
const minimalLLDsHCL = `
resource "zabbix_lld_trapper" "testminimal" {
	hostid = zabbix_template.testminimaltmpl.id
	key    = "test.minimal.lld.trapper"
	name   = "Test Minimal LLD Trapper"
}
resource "zabbix_lld_agent" "testminimal" {
	hostid = zabbix_template.testminimaltmpl.id
	key    = "test.minimal.lld.agent"
	name   = "Test Minimal LLD Agent"
}
resource "zabbix_lld_simple" "testminimal" {
	hostid = zabbix_template.testminimaltmpl.id
	key    = "test.minimal.lld.simple"
	name   = "Test Minimal LLD Simple"
}
resource "zabbix_lld_external" "testminimal" {
	hostid = zabbix_template.testminimaltmpl.id
	key    = "test.minimal.lld.external"
	name   = "Test Minimal LLD External"
}
resource "zabbix_lld_internal" "testminimal" {
	hostid = zabbix_template.testminimaltmpl.id
	key    = "test.minimal.lld.internal"
	name   = "Test Minimal LLD Internal"
}
resource "zabbix_lld_snmp" "testminimal" {
	hostid   = zabbix_template.testminimaltmpl.id
	key      = "test.minimal.lld.snmp"
	name     = "Test Minimal LLD Snmp"
	snmp_oid = "1.3.6.1.2.1.2.2.1.1"
}
resource "zabbix_lld_http" "testminimal" {
	hostid = zabbix_template.testminimaltmpl.id
	key    = "test.minimal.lld.http"
	name   = "Test Minimal LLD Http"
	url    = "http://localhost/minimal/discovery"
}
resource "zabbix_lld_dependent" "testminimal" {
	hostid        = zabbix_template.testminimaltmpl.id
	key           = "test.minimal.lld.dependent"
	name          = "Test Minimal LLD Dependent"
	master_itemid = zabbix_item_trapper.testminimalmaster.id
}
resource "zabbix_item_trapper" "testminimalmaster" {
	hostid    = zabbix_template.testminimaltmpl.id
	key       = "test.minimal.lld.master"
	name      = "Test Minimal LLD Master"
	valuetype = "text"
}
`

func TestAccMinimalLLDs(t *testing.T) {
	config := hcl(t, minimalTemplateHCL+minimalLLDsHCL)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testminimal", "delay", "0"),
					resource.TestCheckResourceAttr("zabbix_lld_dependent.testminimal", "delay", "0"),
				),
			},
			{Config: config, PlanOnly: true},
		},
	})
}

// minimalProtoItemsHCL is all ten item prototypes at their minimum. A
// prototype needs a rule, and its key must carry an LLD macro or Zabbix
// rejects it -- both are genuine requirements of the object, not of the
// provider, and both are already Required in the schema.
const minimalProtoItemsHCL = `
resource "zabbix_lld_trapper" "testminimalrule" {
	hostid = zabbix_template.testminimaltmpl.id
	key    = "test.minimal.proto.rule"
	name   = "Test Minimal Proto Rule"
}
resource "zabbix_proto_item_trapper" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	ruleid    = zabbix_lld_trapper.testminimalrule.id
	key       = "test.minimal.proto.trapper[{#NAME}]"
	name      = "Test Minimal Proto Trapper {#NAME}"
	valuetype = "unsigned"
}
resource "zabbix_proto_item_agent" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	ruleid    = zabbix_lld_trapper.testminimalrule.id
	key       = "test.minimal.proto.agent[{#NAME}]"
	name      = "Test Minimal Proto Agent {#NAME}"
	valuetype = "unsigned"
}
resource "zabbix_proto_item_simple" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	ruleid    = zabbix_lld_trapper.testminimalrule.id
	key       = "test.minimal.proto.simple[{#NAME}]"
	name      = "Test Minimal Proto Simple {#NAME}"
	valuetype = "unsigned"
}
resource "zabbix_proto_item_external" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	ruleid    = zabbix_lld_trapper.testminimalrule.id
	key       = "test.minimal.proto.external[{#NAME}]"
	name      = "Test Minimal Proto External {#NAME}"
	valuetype = "unsigned"
}
resource "zabbix_proto_item_internal" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	ruleid    = zabbix_lld_trapper.testminimalrule.id
	key       = "test.minimal.proto.internal[{#NAME}]"
	name      = "Test Minimal Proto Internal {#NAME}"
	valuetype = "unsigned"
}
resource "zabbix_proto_item_snmptrap" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	ruleid    = zabbix_lld_trapper.testminimalrule.id
	key       = "snmptrap[{#NAME}]"
	name      = "Test Minimal Proto Snmptrap {#NAME}"
	valuetype = "text"
}
resource "zabbix_proto_item_snmp" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	ruleid    = zabbix_lld_trapper.testminimalrule.id
	key       = "test.minimal.proto.snmp[{#NAME}]"
	name      = "Test Minimal Proto Snmp {#NAME}"
	valuetype = "unsigned"
	snmp_oid  = "1.3.6.1.2.1.1.1.0"
}
resource "zabbix_proto_item_http" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	ruleid    = zabbix_lld_trapper.testminimalrule.id
	key       = "test.minimal.proto.http[{#NAME}]"
	name      = "Test Minimal Proto Http {#NAME}"
	valuetype = "text"
	url       = "http://localhost/minimal/{#NAME}"
}
resource "zabbix_proto_item_calculated" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	ruleid    = zabbix_lld_trapper.testminimalrule.id
	key       = "test.minimal.proto.calculated[{#NAME}]"
	name      = "Test Minimal Proto Calculated {#NAME}"
	valuetype = "unsigned"
	formula   = "last(/test-minimal-template/test.minimal.proto.trapper[{#NAME}])"
}
resource "zabbix_proto_item_dependent" "testminimal" {
	hostid        = zabbix_template.testminimaltmpl.id
	ruleid        = zabbix_lld_trapper.testminimalrule.id
	key           = "test.minimal.proto.dependent[{#NAME}]"
	name          = "Test Minimal Proto Dependent {#NAME}"
	valuetype     = "unsigned"
	master_itemid = zabbix_proto_item_trapper.testminimal.id
}
`

func TestAccMinimalProtoItems(t *testing.T) {
	config := hcl(t, minimalTemplateHCL+minimalProtoItemsHCL)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{Config: config},
			{Config: config, PlanOnly: true},
		},
	})
}

// TestAccMinimalTriggers -- name and expression, for both the trigger and the
// trigger prototype.
func TestAccMinimalTriggers(t *testing.T) {
	config := hcl(t, minimalTemplateHCL+`
resource "zabbix_item_trapper" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	key       = "test.minimal.trigger.item"
	name      = "Test Minimal Trigger Item"
	valuetype = "unsigned"
}
resource "zabbix_lld_trapper" "testminimalrule" {
	hostid = zabbix_template.testminimaltmpl.id
	key    = "test.minimal.trigger.rule"
	name   = "Test Minimal Trigger Rule"
}
resource "zabbix_proto_item_trapper" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	ruleid    = zabbix_lld_trapper.testminimalrule.id
	key       = "test.minimal.trigger.proto[{#NAME}]"
	name      = "Test Minimal Trigger Proto {#NAME}"
	valuetype = "unsigned"
}
resource "zabbix_trigger" "testminimal" {
	name       = "Test Minimal Trigger"
	expression = "last(/test-minimal-template/test.minimal.trigger.item)=1"

	depends_on = [ zabbix_item_trapper.testminimal ]
}
resource "zabbix_proto_trigger" "testminimal" {
	name       = "Test Minimal Proto Trigger {#NAME}"
	expression = "last(/test-minimal-template/test.minimal.trigger.proto[{#NAME}])=1"

	depends_on = [ zabbix_proto_item_trapper.testminimal ]
}
`)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{Config: config},
			{Config: config, PlanOnly: true},
		},
	})
}

// TestAccMinimalBlocks is the other half of "only what is required".
//
// A minimal top-level config omits every optional *block*, so it never reaches
// the defaults declared inside one -- and that is where the error_handler
// defect lived: `preprocessor` is optional, so no minimal config ever built a
// step, and every fixture that did build one wrote error_handler out by hand.
// The block defaults are therefore tested here, one block at a time, each
// carrying nothing but the attributes the block itself marks Required.
func TestAccMinimalBlocks(t *testing.T) {
	host := func(body string) string {
		return `
resource "zabbix_hostgroup" "testminimalblockgrp" {
	name = "test-minimal-block-group"
}
resource "zabbix_host" "testminimal" {
	host   = "test-minimal-block-host"
	groups = [ zabbix_hostgroup.testminimalblockgrp.id ]
` + body + `
}
`
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // an interface carrying nothing but an address: type defaults to
				// agent, main to true, and port is left for Zabbix to fill in
				Config: host(`
	interface {
		ip = "127.0.0.1"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testminimal", "interface.*", map[string]string{
						"ip":   "127.0.0.1",
						"type": "agent",
						"main": "true",
						"port": "10050",
					}),
				),
			},
			{ // the same by name rather than address
				Config: host(`
	interface {
		dns = "localhost"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testminimal", "interface.*", map[string]string{
						"dns":  "localhost",
						"type": "agent",
						"port": "10050",
					}),
				),
			},
			{ // an SNMP interface with none of the eight snmp_* defaults
				// written out. Zabbix stores the macro strings verbatim, so
				// these have to survive the round trip unresolved
				Config: host(`
	interface {
		type = "snmp"
		ip   = "127.0.0.1"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testminimal", "interface.*", map[string]string{
						"type":           "snmp",
						"port":           "161",
						"snmp_version":   "2",
						"snmp_bulk":      "true",
						"snmp_community": "{$SNMP_COMMUNITY}",
					}),
				),
			},
			{ // macro and tag at their own minimum: a tag with no value is
				// legal in Zabbix and means "the key alone"
				Config: host(`
	interface {
		ip = "127.0.0.1"
	}
	macro {
		name  = "{$MINIMAL}"
		value = "yes"
	}
	tag {
		key = "minimal"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("zabbix_host.testminimal", "tag.*", map[string]string{
						"key":   "minimal",
						"value": "",
					}),
				),
			},
		},
	})
}

// TestAccMinimalInterfaceNeedsAddress pins the one place a resource genuinely
// needs more than its Required set. Neither `ip` nor `dns` can be Required --
// either will do -- so the constraint is enforced in hostGenerateInterfaces
// instead, and an interface block with neither must say so rather than send
// Zabbix an address-less interface.
func TestAccMinimalInterfaceNeedsAddress(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: `
resource "zabbix_hostgroup" "testminimalifacegrp" {
	name = "test-minimal-iface-group"
}
resource "zabbix_host" "testminimal" {
	host   = "test-minimal-iface-host"
	groups = [ zabbix_hostgroup.testminimalifacegrp.id ]
	interface {
	}
}
`,
				ExpectError: regexp.MustCompile("interface requires either an IP or DNS entry"),
			},
		},
	})
}

// TestAccMinimalLLDPreprocessor is the LLD half of the error_handler defect.
// common_lld.go builds its own preprocessing steps -- it does not share
// common_item.go's -- so the item-side regression test in
// TestAccResourceItemAgentPreprocessorDefaults says nothing about this path,
// and every LLD preprocessing fixture in the suite writes error_handler out.
func TestAccMinimalLLDPreprocessor(t *testing.T) {
	config := hcl(t, minimalTemplateHCL+`
resource "zabbix_lld_trapper" "testminimal" {
	hostid = zabbix_template.testminimaltmpl.id
	key    = "test.minimal.lld.pre"
	name   = "Test Minimal LLD Preprocessing"

	preprocessor {
		type   = "jsonpath"
		params = [ "$.data" ]
	}
	preprocessor {
		type = "xml_to_json"
	}
}
`)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testminimal", "preprocessor.0.error_handler", "0"),
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testminimal", "preprocessor.1.error_handler", "0"),
					testAccCheckLLDPreprocessorCount("zabbix_lld_trapper.testminimal", 2),
				),
			},
			{Config: config, PlanOnly: true},
		},
	})
}

// TestAccMinimalGraphs -- name, width, height and a single item, for both the
// graph and the graph prototype. A graph prototype needs at least one item
// prototype among its items; the plain item is what makes the minimum
// reachable at all.
func TestAccMinimalGraphs(t *testing.T) {
	config := hcl(t, minimalTemplateHCL+`
resource "zabbix_item_agent" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	key       = "test.minimal.graph.item"
	name      = "Test Minimal Graph Item"
	valuetype = "unsigned"
}
resource "zabbix_lld_trapper" "testminimalrule" {
	hostid = zabbix_template.testminimaltmpl.id
	key    = "test.minimal.graph.rule"
	name   = "Test Minimal Graph Rule"
}
resource "zabbix_proto_item_agent" "testminimal" {
	hostid    = zabbix_template.testminimaltmpl.id
	ruleid    = zabbix_lld_trapper.testminimalrule.id
	key       = "test.minimal.graph.proto[{#NAME}]"
	name      = "Test Minimal Graph Proto {#NAME}"
	valuetype = "unsigned"
}
resource "zabbix_graph" "testminimal" {
	name   = "Test Minimal Graph"
	width  = "900"
	height = "200"

	item {
		itemid = zabbix_item_agent.testminimal.id
		color  = "1122AA"
	}
}
resource "zabbix_proto_graph" "testminimal" {
	name   = "Test Minimal Proto Graph {#NAME}"
	width  = "900"
	height = "200"

	item {
		itemid = zabbix_proto_item_agent.testminimal.id
		color  = "1122AA"
	}
}
`)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{Config: config},
			{Config: config, PlanOnly: true},
		},
	})
}
