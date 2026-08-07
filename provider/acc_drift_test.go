package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// E1 -- drift, i.e. the object disappearing from Zabbix behind Terraform's
// back (PLAN.md Phase 8).
//
// Every resource's Read has to notice that the object it is refreshing no
// longer exists and clear the id, so that the next plan proposes recreating
// it. A Read that instead returns the stale state leaves Terraform believing
// the object still exists *forever*: every subsequent plan is empty, every
// apply is a no-op, and the only way out is a manual `terraform state rm`.
// Nothing about that failure is loud -- there is no error and no diff -- so
// it can only be found by deliberately deleting the object and asserting on
// the plan that follows.
//
// The shape used here is the "disappears" pattern. The deletion happens in a
// step's Check, which the SDK runs after apply and *before* the two plans it
// uses to prove idempotency. `ExpectNonEmptyPlan` then inverts the meaning of
// those plans: the harness requires the post-refresh plan to be non-empty
// (testing_new_config.go errors with "Expected a non-empty plan, but got an
// empty refresh plan" otherwise), which is precisely the assertion that the
// stale-state bug fails. On top of that a plan check requires the action to
// be *create* for each deleted address rather than merely non-empty, so an
// unrelated diff cannot be mistaken for recovery. A second step re-applies
// the same configuration and, having no ExpectNonEmptyPlan of its own, must
// converge to an empty plan.
//
// Coverage is by *object*, not by Terraform resource name, but no resource is
// left out: the ten zabbix_item_* types share one resourceItemRead, the ten
// zabbix_proto_item_* types one prototype=true variant of it, and the eight
// zabbix_lld_* types one resourceLLDRead -- so those are exercised as three
// fixtures that each stand up every member of the family at once and delete
// the lot. That costs one apply instead of twenty-eight and still names every
// registered resource type.

// testAccDisappear returns a TestCheckFunc that deletes, straight through the
// Zabbix API, the objects recorded in state at each of addrs. All of them go
// in a single delete call, which is both faster and the only way to remove
// objects Zabbix would refuse to delete one at a time.
func testAccDisappear(del func(*zabbix.API, []string) error, addrs ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		api, ok := testAccProvider.Meta().(*zabbix.API)
		if !ok || api == nil {
			return fmt.Errorf("testAccDisappear: provider not configured")
		}

		ids := make([]string, 0, len(addrs))
		for _, addr := range addrs {
			id, err := testAccStateID(s, addr)
			if err != nil {
				return err
			}
			ids = append(ids, id)
		}

		if err := del(api, ids); err != nil {
			return fmt.Errorf("deleting %v (%v) out of band: %s", addrs, ids, err)
		}
		return nil
	}
}

// testAccExpectRecreated builds the plan checks asserting each address is
// planned for creation. Merely requiring a non-empty plan would also pass on
// a plan that recreates something else entirely, or that proposes an update
// to an unrelated attribute.
func testAccExpectRecreated(addrs ...string) []plancheck.PlanCheck {
	checks := make([]plancheck.PlanCheck, 0, len(addrs))
	for _, addr := range addrs {
		checks = append(checks, plancheck.ExpectResourceAction(addr, plancheck.ResourceActionCreate))
	}
	return checks
}

// testAccDriftSteps is the whole E1 assertion for one fixture: apply, delete
// the named objects out of band, require the refreshed plan to recreate
// exactly them, then re-apply and require the result to settle.
func testAccDriftSteps(config string, del func(*zabbix.API, []string) error, addrs ...string) []resource.TestStep {
	return []resource.TestStep{
		{
			Config:             config,
			Check:              testAccDisappear(del, addrs...),
			ExpectNonEmptyPlan: true,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PostApplyPostRefresh: testAccExpectRecreated(addrs...),
			},
		},
		{ // recovery: the same config must now apply cleanly and converge
			Config: config,
		},
	}
}

func testAccDeleteHostGroups(api *zabbix.API, ids []string) error {
	return api.HostGroupsDeleteByIds(ids)
}

func testAccDeleteTemplateGroups(api *zabbix.API, ids []string) error {
	return api.TemplateGroupsDeleteByIds(ids)
}

// testAccDeleteGroups picks the delete call matching whichever object hcl()
// has rewritten the fixture's zabbix_templategroup into.
func testAccDeleteGroups(t *testing.T) func(*zabbix.API, []string) error {
	if testAccTemplateGroups(t) {
		return testAccDeleteTemplateGroups
	}
	return testAccDeleteHostGroups
}

func testAccDeleteHosts(api *zabbix.API, ids []string) error {
	return api.HostsDeleteByIds(ids)
}

func testAccDeleteTemplates(api *zabbix.API, ids []string) error {
	return api.TemplatesDeleteByIds(ids)
}

func testAccDeleteProxies(api *zabbix.API, ids []string) error {
	return api.ProxiesDeleteByIds(ids)
}

func testAccDeleteItems(api *zabbix.API, ids []string) error {
	return api.ItemsDeleteByIds(ids)
}

func testAccDeleteProtoItems(api *zabbix.API, ids []string) error {
	return api.ProtoItemsDeleteByIds(ids)
}

func testAccDeleteLLDs(api *zabbix.API, ids []string) error {
	return api.LLDDeleteByIds(ids)
}

func testAccDeleteTriggers(api *zabbix.API, ids []string) error {
	return api.TriggersDeleteByIds(ids)
}

func testAccDeleteProtoTriggers(api *zabbix.API, ids []string) error {
	return api.ProtoTriggersDeleteByIds(ids)
}

func testAccDeleteGraphs(api *zabbix.API, ids []string) error {
	return api.GraphsDeleteByIds(ids)
}

func testAccDeleteProtoGraphs(api *zabbix.API, ids []string) error {
	return api.GraphProtosDeleteByIds(ids)
}

func TestAccDriftHostgroup(t *testing.T) {
	config := `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-drift-group"
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps:             testAccDriftSteps(config, testAccDeleteHostGroups, "zabbix_hostgroup.testgrp"),
	})
}

func TestAccDriftTemplategroup(t *testing.T) {
	config := hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-drift-template-group"
}
`)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: testAccDriftSteps(config, testAccDeleteGroups(t),
			tmplGroupAddr(t, "testtmplgrp")),
	})
}

func TestAccDriftHost(t *testing.T) {
	config := `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-drift-group"
}
resource "zabbix_host" "testhost" {
	host   = "test-drift-host"
	groups = [zabbix_hostgroup.testgrp.id]
	interface {
		type = "agent"
		ip   = "127.0.0.1"
	}
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps:             testAccDriftSteps(config, testAccDeleteHosts, "zabbix_host.testhost"),
	})
}

func TestAccDriftTemplate(t *testing.T) {
	config := hcl(t, `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-drift-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host   = "test-drift-template"
}
`)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps:             testAccDriftSteps(config, testAccDeleteTemplates, "zabbix_template.testtmpl"),
	})
}

func TestAccDriftProxy(t *testing.T) {
	config := `
resource "zabbix_proxy" "testproxy" {
	name = "test-drift-proxy"
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps:             testAccDriftSteps(config, testAccDeleteProxies, "zabbix_proxy.testproxy"),
	})
}

// driftTemplateHCL is the owner every item, prototype, LLD rule, graph and
// trigger fixture below hangs off. Its names carry the "test" prefix the
// sweepers key on and are distinct from the shared fixtures' so that a drift
// test aborted mid-flight cannot be mistaken for -- or collide with -- one of
// them.
const driftTemplateHCL = `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-drift-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host   = "test-drift-template"
}
resource "zabbix_item_trapper" "testmaster" {
	hostid    = zabbix_template.testtmpl.id
	key       = "drift.master"
	name      = "Drift Master Item"
	valuetype = "text"
}
`

// TestAccDriftItems covers all ten zabbix_item_* resources at once: they are
// one Zabbix object type behind one shared resourceItemRead, so a fixture per
// type would be ten applies proving the same thing. The master trapper item
// is deliberately left alive -- deleting it would cascade to the dependent
// item and the plan check could then pass for the wrong reason.
func TestAccDriftItems(t *testing.T) {
	config := hcl(t, driftTemplateHCL+`
resource "zabbix_item_agent" "testagent" {
	hostid    = zabbix_template.testtmpl.id
	key       = "drift.agent"
	name      = "Drift Agent"
	valuetype = "unsigned"
}
resource "zabbix_item_calculated" "testcalculated" {
	hostid    = zabbix_template.testtmpl.id
	key       = "drift.calculated"
	name      = "Drift Calculated"
	valuetype = "float"
	formula   = "avg(/Zabbix Server/zabbix[wcache,values],10m)"
}
resource "zabbix_item_dependent" "testdependent" {
	hostid        = zabbix_template.testtmpl.id
	key           = "drift.dependent"
	name          = "Drift Dependent"
	valuetype     = "text"
	master_itemid = zabbix_item_trapper.testmaster.id
}
resource "zabbix_item_external" "testexternal" {
	hostid    = zabbix_template.testtmpl.id
	key       = "drift.external[\"a\"]"
	name      = "Drift External"
	valuetype = "text"
}
resource "zabbix_item_http" "testhttp" {
	hostid    = zabbix_template.testtmpl.id
	key       = "drift.http"
	name      = "Drift Http"
	valuetype = "text"
	url       = "http://localhost/drift"
}
resource "zabbix_item_internal" "testinternal" {
	hostid    = zabbix_template.testtmpl.id
	key       = "drift.internal[\"a\"]"
	name      = "Drift Internal"
	valuetype = "unsigned"
}
resource "zabbix_item_simple" "testsimple" {
	hostid    = zabbix_template.testtmpl.id
	key       = "drift.simple[\"a\"]"
	name      = "Drift Simple"
	valuetype = "unsigned"
}
resource "zabbix_item_snmp" "testsnmp" {
	hostid    = zabbix_template.testtmpl.id
	key       = "drift.snmp[\"a\"]"
	name      = "Drift Snmp"
	valuetype = "unsigned"
	snmp_oid  = "1.2.3.4"
}
resource "zabbix_item_snmptrap" "testsnmptrap" {
	hostid    = zabbix_template.testtmpl.id
	key       = "snmptrap[\"drift\"]"
	name      = "Drift Snmptrap"
	valuetype = "text"
}
resource "zabbix_item_trapper" "testtrapper" {
	hostid    = zabbix_template.testtmpl.id
	key       = "drift.trapper"
	name      = "Drift Trapper"
	valuetype = "text"
}
`)
	addrs := []string{
		"zabbix_item_agent.testagent",
		"zabbix_item_calculated.testcalculated",
		"zabbix_item_dependent.testdependent",
		"zabbix_item_external.testexternal",
		"zabbix_item_http.testhttp",
		"zabbix_item_internal.testinternal",
		"zabbix_item_simple.testsimple",
		"zabbix_item_snmp.testsnmp",
		"zabbix_item_snmptrap.testsnmptrap",
		"zabbix_item_trapper.testtrapper",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps:             testAccDriftSteps(config, testAccDeleteItems, addrs...),
	})
}

// TestAccDriftProtoItems is TestAccDriftItems for the prototype namespace.
// It is a separate test rather than a bigger fixture because item prototypes
// live in a different API namespace (itemprototype.get) and are read by the
// prototype=true branch of resourceItemRead -- an id gone from one is not
// gone from the other.
func TestAccDriftProtoItems(t *testing.T) {
	config := hcl(t, driftTemplateHCL+`
resource "zabbix_lld_trapper" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key    = "drift.lld.rule"
	name   = "Drift LLD Rule"
	delay  = "0"
}
resource "zabbix_proto_item_agent" "testagent" {
	hostid    = zabbix_template.testtmpl.id
	ruleid    = zabbix_lld_trapper.testlld.id
	key       = "drift.agent[{#FSNAME}]"
	name      = "Drift Proto Agent {#FSNAME}"
	valuetype = "unsigned"
}
resource "zabbix_proto_item_calculated" "testcalculated" {
	hostid    = zabbix_template.testtmpl.id
	ruleid    = zabbix_lld_trapper.testlld.id
	key       = "drift.calculated[{#FSNAME}]"
	name      = "Drift Proto Calculated {#FSNAME}"
	valuetype = "float"
	formula   = "avg(/Zabbix Server/zabbix[wcache,values],10m)"
}
resource "zabbix_proto_item_dependent" "testdependent" {
	hostid        = zabbix_template.testtmpl.id
	ruleid        = zabbix_lld_trapper.testlld.id
	key           = "drift.dependent[{#FSNAME}]"
	name          = "Drift Proto Dependent {#FSNAME}"
	valuetype     = "text"
	master_itemid = zabbix_item_trapper.testmaster.id
}
resource "zabbix_proto_item_external" "testexternal" {
	hostid    = zabbix_template.testtmpl.id
	ruleid    = zabbix_lld_trapper.testlld.id
	key       = "drift.external[{#FSNAME}]"
	name      = "Drift Proto External {#FSNAME}"
	valuetype = "text"
}
resource "zabbix_proto_item_http" "testhttp" {
	hostid    = zabbix_template.testtmpl.id
	ruleid    = zabbix_lld_trapper.testlld.id
	key       = "drift.http[{#FSNAME}]"
	name      = "Drift Proto Http {#FSNAME}"
	valuetype = "text"
	url       = "http://localhost/drift"
}
resource "zabbix_proto_item_internal" "testinternal" {
	hostid    = zabbix_template.testtmpl.id
	ruleid    = zabbix_lld_trapper.testlld.id
	key       = "drift.internal[{#FSNAME}]"
	name      = "Drift Proto Internal {#FSNAME}"
	valuetype = "unsigned"
}
resource "zabbix_proto_item_simple" "testsimple" {
	hostid    = zabbix_template.testtmpl.id
	ruleid    = zabbix_lld_trapper.testlld.id
	key       = "drift.simple[{#FSNAME}]"
	name      = "Drift Proto Simple {#FSNAME}"
	valuetype = "unsigned"
}
resource "zabbix_proto_item_snmp" "testsnmp" {
	hostid    = zabbix_template.testtmpl.id
	ruleid    = zabbix_lld_trapper.testlld.id
	key       = "drift.snmp[{#FSNAME}]"
	name      = "Drift Proto Snmp {#FSNAME}"
	valuetype = "unsigned"
	snmp_oid  = "1.2.3.4"
}
resource "zabbix_proto_item_snmptrap" "testsnmptrap" {
	hostid    = zabbix_template.testtmpl.id
	ruleid    = zabbix_lld_trapper.testlld.id
	key       = "snmptrap[{#FSNAME}]"
	name      = "Drift Proto Snmptrap {#FSNAME}"
	valuetype = "text"
}
resource "zabbix_proto_item_trapper" "testtrapper" {
	hostid    = zabbix_template.testtmpl.id
	ruleid    = zabbix_lld_trapper.testlld.id
	key       = "drift.trapper[{#FSNAME}]"
	name      = "Drift Proto Trapper {#FSNAME}"
	valuetype = "text"
}
`)
	addrs := []string{
		"zabbix_proto_item_agent.testagent",
		"zabbix_proto_item_calculated.testcalculated",
		"zabbix_proto_item_dependent.testdependent",
		"zabbix_proto_item_external.testexternal",
		"zabbix_proto_item_http.testhttp",
		"zabbix_proto_item_internal.testinternal",
		"zabbix_proto_item_simple.testsimple",
		"zabbix_proto_item_snmp.testsnmp",
		"zabbix_proto_item_snmptrap.testsnmptrap",
		"zabbix_proto_item_trapper.testtrapper",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps:             testAccDriftSteps(config, testAccDeleteProtoItems, addrs...),
	})
}

// TestAccDriftLLDs covers all eight zabbix_lld_* resources, which share
// resourceLLDRead -- a separate implementation from the item one, so the item
// test says nothing about it.
func TestAccDriftLLDs(t *testing.T) {
	config := hcl(t, driftTemplateHCL+`
resource "zabbix_lld_agent" "testagent" {
	hostid = zabbix_template.testtmpl.id
	key    = "drift.lld.agent"
	name   = "Drift LLD Agent"
}
resource "zabbix_lld_dependent" "testdependent" {
	hostid        = zabbix_template.testtmpl.id
	key           = "drift.lld.dependent"
	name          = "Drift LLD Dependent"
	master_itemid = zabbix_item_trapper.testmaster.id
}
resource "zabbix_lld_external" "testexternal" {
	hostid = zabbix_template.testtmpl.id
	key    = "drift.lld.external"
	name   = "Drift LLD External"
}
resource "zabbix_lld_http" "testhttp" {
	hostid = zabbix_template.testtmpl.id
	key    = "drift.lld.http"
	name   = "Drift LLD Http"
	url    = "http://localhost/drift"
}
resource "zabbix_lld_internal" "testinternal" {
	hostid = zabbix_template.testtmpl.id
	key    = "drift.lld.internal"
	name   = "Drift LLD Internal"
}
resource "zabbix_lld_simple" "testsimple" {
	hostid = zabbix_template.testtmpl.id
	key    = "drift.lld.simple"
	name   = "Drift LLD Simple"
}
resource "zabbix_lld_snmp" "testsnmp" {
	hostid   = zabbix_template.testtmpl.id
	key      = "drift.lld.snmp"
	name     = "Drift LLD Snmp"
	snmp_oid = "1.2.3.4"
}
resource "zabbix_lld_trapper" "testtrapper" {
	hostid = zabbix_template.testtmpl.id
	key    = "drift.lld.trapper"
	name   = "Drift LLD Trapper"
	delay  = "0"
}
`)
	addrs := []string{
		"zabbix_lld_agent.testagent",
		"zabbix_lld_dependent.testdependent",
		"zabbix_lld_external.testexternal",
		"zabbix_lld_http.testhttp",
		"zabbix_lld_internal.testinternal",
		"zabbix_lld_simple.testsimple",
		"zabbix_lld_snmp.testsnmp",
		"zabbix_lld_trapper.testtrapper",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps:             testAccDriftSteps(config, testAccDeleteLLDs, addrs...),
	})
}

// TestAccDriftGraph and TestAccDriftTrigger use the same fixture but delete
// through different API namespaces, so they cannot share a step.
const driftGraphTriggerHCL = driftTemplateHCL + `
resource "zabbix_item_agent" "testitem" {
	hostid    = zabbix_template.testtmpl.id
	key       = "drift.graph.item"
	name      = "Drift Graph Item"
	valuetype = "unsigned"
}
`

func TestAccDriftGraph(t *testing.T) {
	config := hcl(t, driftGraphTriggerHCL+`
resource "zabbix_graph" "testgraph" {
	name   = "test-drift-graph"
	width  = "600"
	height = "400"

	item {
		color  = "FFFF00"
		itemid = zabbix_item_agent.testitem.id
	}
}
`)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps:             testAccDriftSteps(config, testAccDeleteGraphs, "zabbix_graph.testgraph"),
	})
}

func TestAccDriftTrigger(t *testing.T) {
	config := hcl(t, driftGraphTriggerHCL+`
resource "zabbix_trigger" "testtrigger" {
	name       = "test-drift-trigger"
	expression = "last(/test-drift-template/drift.graph.item)>10"

	depends_on = [zabbix_item_agent.testitem]
}
`)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps:             testAccDriftSteps(config, testAccDeleteTriggers, "zabbix_trigger.testtrigger"),
	})
}

// driftProtoHCL adds the discovery rule and item prototype that a graph
// prototype and a trigger prototype need: neither carries a ruleid of its own,
// Zabbix infers the owning rule from the prototypes they reference.
const driftProtoHCL = driftTemplateHCL + `
resource "zabbix_lld_trapper" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key    = "drift.lld.rule"
	name   = "Drift LLD Rule"
	delay  = "0"
}
resource "zabbix_proto_item_trapper" "testproto" {
	hostid    = zabbix_template.testtmpl.id
	ruleid    = zabbix_lld_trapper.testlld.id
	key       = "drift.proto[{#FSNAME}]"
	name      = "Drift Proto Item {#FSNAME}"
	valuetype = "unsigned"
}
`

func TestAccDriftProtoGraph(t *testing.T) {
	config := hcl(t, driftProtoHCL+`
resource "zabbix_proto_graph" "testgraph" {
	name   = "test-drift-proto-graph {#FSNAME}"
	width  = "600"
	height = "400"

	item {
		color  = "FFFF00"
		itemid = zabbix_proto_item_trapper.testproto.id
	}
}
`)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps:             testAccDriftSteps(config, testAccDeleteProtoGraphs, "zabbix_proto_graph.testgraph"),
	})
}

func TestAccDriftProtoTrigger(t *testing.T) {
	config := hcl(t, driftProtoHCL+`
resource "zabbix_proto_trigger" "testtrigger" {
	name       = "test-drift-proto-trigger {#FSNAME}"
	expression = "last(/test-drift-template/drift.proto[{#FSNAME}])>10"

	depends_on = [zabbix_proto_item_trapper.testproto]
}
`)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps:             testAccDriftSteps(config, testAccDeleteProtoTriggers, "zabbix_proto_trigger.testtrigger"),
	})
}
