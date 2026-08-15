package provider

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// R1-R2 -- removing an attribute from the configuration (PLAN.md § "The unit
// of work").
//
// `S9d` asks whether an attribute the user set can be set to an *empty value*
// again. This is the other half of the same question, and it is the edit a
// user actually makes: **deleting the line**. Nobody writes `timeout = ""`;
// they delete the `timeout = "10s"` they no longer want.
//
// What deletion means depends entirely on one flag, and the two halves behave
// nothing alike:
//
//   - `Optional` with a `Default:` -- deletion plans a change back to the
//     default and the provider has to *send* it. The failure mode is the one
//     the six omitempty bugs had: a plan that shows the revert while the
//     server quietly keeps the old value. State is written by the provider's
//     own read, so it agrees with whatever the server kept and nothing looks
//     wrong.
//   - `Optional + Computed` -- deletion produces **no diff at all**. That is
//     Terraform's contract for the flag ("if absent from config, keep whatever
//     the provider last returned"), so the value sticks forever and the user
//     has no way to unset it. Sometimes that is exactly right, because the
//     server derives the value; sometimes it is an accident that traps the
//     user.
//
// | | Case | What it must do |
// |---|---|---|
// | R1 | Optional with a Default | removing the line reverts to the schema default, asserted against a *server re-read* rather than against Terraform state |
// | R2 | Optional + Computed | the stickiness is a deliberate decision, recorded next to the attribute, and asserted either way |
//
// R2 has no single right answer, so the deliverable is the decision. Every
// Optional+Computed attribute is named in removalComputed with the verdict and
// the reason, and the test asserts the verdict it claims -- an "intended" one
// asserts that the value sticks and that the documented way back works.
//
// Where the coverage lives
// ------------------------
//
// Attributes are grouped by *pointer identity*, exactly as
// TestUpdateCoverageComplete does it: mergeSchemas copies *schema.Schema
// pointers, so all twenty item and prototype resources share one `history`
// declaration and covering it once covers it everywhere.
//
//	schema fragment           file                     coverage
//	------------------------  -----------------------  -------------------------
//	itemCommonSchema          common_item.go           TestAccRemoveItemDefaults
//	itemDelaySchema           common_item.go           TestAccRemoveItemDefaults
//	itemInterfaceSchema       common_item.go           TestAccRemoveItemInterfaceID
//	itemPreprocessorSchema    common_item.go           TestAccRemoveItemDefaults
//	lldCommonSchema           common_lld.go            TestAccRemoveLLDDefaults
//	lldDelaySchema            common_lld.go            TestAccRemoveLLDDefaults
//	lldZeroDelaySchema        common_lld.go            exempt -- one valid value
//	lldInterfaceSchema        common_lld.go            TestAccRemoveLLDInterfaceID
//	lldPreprocessorSchema     common_lld.go            TestAccRemoveLLDDefaults
//	lldFilterConditionSchema  common_lld.go            TestAccRemoveLLDDefaults
//	http item/LLD fragment    resource_http_common.go  TestAccRemoveHttpItemDefaults
//	graph + graph item        resource_graph.go        TestAccRemoveGraphDefaults
//	trigger                   resource_trigger.go      TestAccRemoveTriggerDefaults
//	host body                 resource_host.go         TestAccRemoveHostDefaults
//	host interface            resource_host.go         TestAccRemoveHostInterfaceDefaults,
//	                                                   TestAccRemoveHostInterfaceMain,
//	                                                   TestAccRemoveHostInterfaceType
//	template                  resource_template.go     TestAccRemoveTemplateDefaults
//	proxy                     resource_proxy.go        TestAccRemoveProxyDefaults

// ---------------------------------------------------------------------------
// the registries
// ---------------------------------------------------------------------------

// removalOwner names, for every `Optional` attribute carrying a `Default:`,
// the acceptance test that deletes the line from the configuration and
// asserts the default reaching the server.
//
// A key is "<resource>.<dotted attribute path>", and only one resource per
// shared declaration needs naming; see the note on updateOwner.
var removalOwner = map[string]string{
	"zabbix_graph.do3d":                                   "TestAccRemoveGraphDefaults",
	"zabbix_graph.item.drawtype":                          "TestAccRemoveGraphDefaults",
	"zabbix_graph.item.function":                          "TestAccRemoveGraphDefaults",
	"zabbix_graph.item.sortorder":                         "TestAccRemoveGraphDefaults",
	"zabbix_graph.item.type":                              "TestAccRemoveGraphDefaults",
	"zabbix_graph.item.yaxis_side":                        "TestAccRemoveGraphDefaults",
	"zabbix_graph.legend":                                 "TestAccRemoveGraphDefaults",
	"zabbix_graph.percent_left":                           "TestAccRemoveGraphDefaults",
	"zabbix_graph.percent_right":                          "TestAccRemoveGraphDefaults",
	"zabbix_graph.type":                                   "TestAccRemoveGraphDefaults",
	"zabbix_graph.work_period":                            "TestAccRemoveGraphDefaults",
	"zabbix_graph.ymax":                                   "TestAccRemoveGraphDefaults",
	"zabbix_graph.ymax_type":                              "TestAccRemoveGraphDefaults",
	"zabbix_graph.ymin":                                   "TestAccRemoveGraphDefaults",
	"zabbix_graph.ymin_type":                              "TestAccRemoveGraphDefaults",
	"zabbix_host.enabled":                                 "TestAccRemoveHostDefaults",
	"zabbix_host.interface.main":                          "TestAccRemoveHostInterfaceMain",
	"zabbix_host.interface.snmp3_authpassphrase":          "TestAccRemoveHostInterfaceDefaults",
	"zabbix_host.interface.snmp3_authprotocol":            "TestAccRemoveHostInterfaceDefaults",
	"zabbix_host.interface.snmp3_contextname":             "TestAccRemoveHostInterfaceDefaults",
	"zabbix_host.interface.snmp3_privpassphrase":          "TestAccRemoveHostInterfaceDefaults",
	"zabbix_host.interface.snmp3_privprotocol":            "TestAccRemoveHostInterfaceDefaults",
	"zabbix_host.interface.snmp3_securitylevel":           "TestAccRemoveHostInterfaceDefaults",
	"zabbix_host.interface.snmp3_securityname":            "TestAccRemoveHostInterfaceDefaults",
	"zabbix_host.interface.snmp_bulk":                     "TestAccRemoveHostInterfaceDefaults",
	"zabbix_host.interface.snmp_community":                "TestAccRemoveHostInterfaceDefaults",
	"zabbix_host.interface.snmp_version":                  "TestAccRemoveHostInterfaceDefaults",
	"zabbix_host.interface.type":                          "TestAccRemoveHostInterfaceType",
	"zabbix_host.inventory_mode":                          "TestAccRemoveHostDefaults",
	"zabbix_host.ipmi_authtype":                           "TestAccRemoveHostDefaults",
	"zabbix_host.ipmi_privilege":                          "TestAccRemoveHostDefaults",
	"zabbix_host.proxyid":                                 "TestAccRemoveHostDefaults",
	"zabbix_host.tls_accept":                              "TestAccRemoveHostDefaults",
	"zabbix_host.tls_connect":                             "TestAccRemoveHostDefaults",
	"zabbix_item_agent.active":                            "TestAccRemoveItemDefaults",
	"zabbix_item_agent.delay":                             "TestAccRemoveItemDefaults",
	"zabbix_item_agent.history":                           "TestAccRemoveItemDefaults",
	"zabbix_item_agent.interfaceid":                       "TestAccRemoveItemInterfaceID",
	"zabbix_item_agent.preprocessor.error_handler":        "TestAccRemoveItemDefaults",
	"zabbix_item_agent.preprocessor.error_handler_params": "TestAccRemoveItemDefaults",
	"zabbix_item_http.auth_type":                          "TestAccRemoveHttpItemDefaults",
	"zabbix_item_http.follow_redirects":                   "TestAccRemoveHttpItemDefaults",
	"zabbix_item_http.post_type":                          "TestAccRemoveHttpItemDefaults",
	"zabbix_item_http.request_method":                     "TestAccRemoveHttpItemDefaults",
	"zabbix_item_http.retrieve_mode":                      "TestAccRemoveHttpItemDefaults",
	"zabbix_item_http.status_codes":                       "TestAccRemoveHttpItemDefaults",
	"zabbix_item_http.timeout":                            "TestAccRemoveHttpItemDefaults",
	"zabbix_item_http.verify_host":                        "TestAccRemoveHttpItemDefaults",
	"zabbix_item_http.verify_peer":                        "TestAccRemoveHttpItemDefaults",
	"zabbix_lld_agent.condition.operator":                 "TestAccRemoveLLDDefaults",
	"zabbix_lld_agent.delay":                              "TestAccRemoveLLDDefaults",
	"zabbix_lld_agent.evaltype":                           "TestAccRemoveLLDDefaults",
	"zabbix_lld_agent.formula":                            "TestAccRemoveLLDDefaults",
	"zabbix_lld_agent.interfaceid":                        "TestAccRemoveLLDInterfaceID",
	"zabbix_lld_agent.lifetime":                           "TestAccRemoveLLDDefaults",
	"zabbix_lld_agent.preprocessor.error_handler":         "TestAccRemoveLLDDefaults",
	"zabbix_lld_agent.preprocessor.error_handler_params":  "TestAccRemoveLLDDefaults",
	"zabbix_proto_trigger.enabled":                        "TestAccRemoveTriggerDefaults",
	"zabbix_proto_trigger.manual_close":                   "TestAccRemoveTriggerDefaults",
	"zabbix_proto_trigger.multiple":                       "TestAccRemoveTriggerDefaults",
	"zabbix_proto_trigger.priority":                       "TestAccRemoveTriggerDefaults",
	"zabbix_proto_trigger.recovery_none":                  "TestAccRemoveTriggerDefaults",
	"zabbix_proxy.address":                                "TestAccRemoveProxyDefaults",
	"zabbix_proxy.operating_mode":                         "TestAccRemoveProxyDefaults",
	"zabbix_proxy.port":                                   "TestAccRemoveProxyDefaults",
	"zabbix_proxy.tls_accept":                             "TestAccRemoveProxyDefaults",
	"zabbix_proxy.tls_connect":                            "TestAccRemoveProxyDefaults",
	"zabbix_template.wizard_ready":                        "TestAccRemoveTemplateDefaults",
}

// removalExempt names the R1 attributes whose line cannot meaningfully be
// deleted, with the reason. Exemption is by name and never by omission.
var removalExempt = map[string]string{
	// lldZeroDelaySchema. Trapper and dependent discovery rules are not
	// polled and Zabbix rejects any delay but "0", so the validator permits
	// exactly one value -- which is also the default. Deleting the line is a
	// no-op by construction, and there is no second value to delete.
	"zabbix_lld_trapper.delay":   "pinned to \"0\": the default is the only value the enum permits, so removing the line cannot change anything",
	"zabbix_lld_dependent.delay": "pinned to \"0\": the default is the only value the enum permits, so removing the line cannot change anything",
}

// removalComputed is R2: every `Optional + Computed` attribute, the decision
// taken about it, and the test that asserts that decision.
//
// The value is "<verdict>: <reason>; <test>", and the verdict is one of
// "intended" or "converted". "intended" says the stickiness is a genuine
// server- or provider-derived default and the attribute stays Optional+
// Computed; the reason has to say what derives it and what the user writes to
// get back. "converted" says the flag was an accident and the attribute is now
// a plain Optional with a Default.
//
// Nothing here is converted, and each verdict was reached against live
// servers rather than by reading the schema -- the notes on each test say
// what was probed.
var removalComputed = map[string]string{}

// removalPending is a migration scaffold, not a fourth registry. It holds the
// declarations whose removal coverage is still being written, so that the
// guard can land before the coverage does and every commit in between stays
// green. It must be empty by the end of the series, which
// TestRemovalCoverageComplete enforces once it is.
var removalPending = map[string]string{
	"zabbix_host.interface.port":            "coverage pending",
	"zabbix_host.name":                      "coverage pending",
	"zabbix_item_agent.trends":              "coverage pending",
	"zabbix_proto_trigger.correlation_mode": "coverage pending",
	"zabbix_template.name":                  "coverage pending",
}

// ---------------------------------------------------------------------------
// the completeness guard
// ---------------------------------------------------------------------------

// removalAttrGroups groups the settable attributes of every resource by the
// declaration they come from and splits them into the two classes. Data
// sources are excluded: nothing is written to them, so there is no line to
// delete.
func removalAttrGroups() (defaulted, computed map[*schema.Schema][]updateAttrSite) {
	defaulted = map[*schema.Schema][]updateAttrSite{}
	computed = map[*schema.Schema][]updateAttrSite{}
	p := Provider()

	names := make([]string, 0, len(p.ResourcesMap))
	for name := range p.ResourcesMap {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		walkSchema(p.ResourcesMap[name].Schema, "", func(path string, s *schema.Schema) {
			if !s.Optional {
				return
			}
			switch {
			case s.Computed:
				computed[s] = append(computed[s], updateAttrSite{name, path})
			case s.Default != nil:
				defaulted[s] = append(defaulted[s], updateAttrSite{name, path})
			}
		})
	}
	return defaulted, computed
}

// TestRemovalCoverageComplete is the guard: an attribute given a `Default:` or
// an `Optional + Computed` pair tomorrow fails here until somebody either
// covers its removal or writes down why it cannot be covered. Being a schema
// walk it needs no server.
func TestRemovalCoverageComplete(t *testing.T) {
	defaulted, computed := removalAttrGroups()
	if len(defaulted) == 0 || len(computed) == 0 {
		t.Fatal("no removable attributes found at all; removalAttrGroups has stopped working")
	}

	live := map[string]string{} // key -> "R1" or "R2"
	for _, sites := range defaulted {
		for _, s := range sites {
			live[s.key()] = "R1"
		}
	}
	for _, sites := range computed {
		for _, s := range sites {
			live[s.key()] = "R2"
		}
	}

	// A registry key has to name an attribute that exists, has to be in the
	// class its registry is about, and has to carry a reason. The class check
	// is the one that matters: adding a Default: to an Optional+Computed
	// attribute, or taking one away, moves it between the two halves of this
	// criterion, and it must not do so silently.
	for _, reg := range []struct {
		name, class string
		m           map[string]string
	}{
		{"removalOwner", "R1", removalOwner},
		{"removalExempt", "R1", removalExempt},
		{"removalComputed", "R2", removalComputed},
		{"removalPending", "", removalPending},
	} {
		keys := make([]string, 0, len(reg.m))
		for k := range reg.m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			switch {
			case live[k] == "":
				t.Errorf("%s has %q, which is not a removable attribute of any resource", reg.name, k)
			// the scaffold spans both classes, so it makes no claim about
			// which one an attribute is in
			case reg.class != "" && live[k] != reg.class:
				t.Errorf("%s has %q, which is %s, not %s -- its schema flags have changed and the decision recorded about it no longer applies",
					reg.name, k, live[k], reg.class)
			}
			if strings.TrimSpace(reg.m[k]) == "" {
				t.Errorf("%s[%q] has no reason", reg.name, k)
			}
		}
	}
	for k := range removalOwner {
		if _, dup := removalExempt[k]; dup {
			t.Errorf("%q is both covered and exempt; it can only be one", k)
		}
	}

	// R2's whole point is the decision, so a verdict that is not one of the
	// two words is not a decision.
	for k, v := range removalComputed {
		if !strings.HasPrefix(v, "intended:") && !strings.HasPrefix(v, "converted:") {
			t.Errorf("removalComputed[%q] must begin \"intended:\" or \"converted:\"; %q says nothing about which", k, v)
		}
	}

	covered, exempt, pending := 0, 0, 0
	var missing []string
	for _, sites := range defaulted {
		// precedence, not site order: while coverage is being written a
		// declaration may be named in more than one registry, and the
		// strongest claim is the true one
		state := ""
		for _, reg := range []struct {
			name string
			m    map[string]string
		}{
			{"covered", removalOwner},
			{"exempt", removalExempt},
			{"pending", removalPending},
		} {
			for _, s := range sites {
				if _, ok := reg.m[s.key()]; ok {
					state = reg.name
					break
				}
			}
			if state != "" {
				break
			}
		}
		switch state {
		case "covered":
			covered++
		case "exempt":
			exempt++
		case "pending":
			pending++
		default:
			missing = append(missing, removalSiteList(sites))
		}
	}

	decided := 0
	var undecided []string
	for _, sites := range computed {
		state := ""
		for _, reg := range []struct {
			name string
			m    map[string]string
		}{
			{"decided", removalComputed},
			{"pending", removalPending},
		} {
			for _, s := range sites {
				if _, ok := reg.m[s.key()]; ok {
					state = reg.name
					break
				}
			}
			if state != "" {
				break
			}
		}
		switch state {
		case "decided":
			decided++
		case "pending":
			pending++
		default:
			undecided = append(undecided, removalSiteList(sites))
		}
	}

	sort.Strings(missing)
	for _, m := range missing {
		t.Errorf("R1: no removal coverage: %s\n\tadd it to removalOwner with the test that deletes the line, or to removalExempt with the reason it cannot be deleted", m)
	}
	sort.Strings(undecided)
	for _, m := range undecided {
		t.Errorf("R2: no decision recorded: %s\n\tadd it to removalComputed as \"intended: ...\" or \"converted: ...\"; an Optional+Computed attribute can never be unset, and whether that is right is not something to leave implied", m)
	}

	if pending == 0 && len(removalPending) > 0 {
		t.Error("removalPending is non-empty but satisfies nothing; delete it")
	}
	t.Logf("R1: %d defaulted declarations, %d covered, %d exempt, %d uncovered; R2: %d Optional+Computed declarations, %d decided, %d undecided; %d pending",
		len(defaulted), covered, exempt, len(missing), len(computed), decided, len(undecided), pending)
}

// removalSiteList renders the places one declaration is reachable from, for
// an error message.
func removalSiteList(sites []updateAttrSite) string {
	keys := make([]string, 0, len(sites))
	for _, s := range sites {
		keys = append(keys, s.key())
	}
	sort.Strings(keys)
	if len(keys) > 4 {
		keys = append(append([]string{}, keys[:4]...), fmt.Sprintf("... and %d more", len(keys)-4))
	}
	return strings.Join(keys, ", ")
}

// ---------------------------------------------------------------------------
// R1 -- Optional with a Default
// ---------------------------------------------------------------------------
//
// Every one of these has the same two-step shape: a first configuration that
// sets each attribute to something that is *not* its default, and a second
// that deletes the lines. The second step carries expectUpdate, so a silent
// replacement cannot pass for a revert, and asserts the defaults against a
// server re-read rather than against state -- state is written by the
// provider's own read, so a default the provider never sent still looks
// applied there.

// removalTemplateHCL gives the template-scoped tests somewhere to put an item
// that needs no host interface.
const removalTemplateHCL = `
resource "zabbix_templategroup" "testremtmplgrp" {
	name = "test-removal-template-group"
}
resource "zabbix_template" "testremtmpl" {
	groups = [ zabbix_templategroup.testremtmplgrp.id ]
	host   = "test-removal-template"
}
`

// TestAccRemoveGraphDefaults owns the fifteen defaults on zabbix_graph and its
// graph item. Only name, width, height and the item's itemid and color are
// Required, so the second configuration here is what the generated
// documentation tells a user to write.
func TestAccRemoveGraphDefaults(t *testing.T) {
	const addr = "zabbix_graph.testremgraph"

	graph := func(body, item string) string {
		return hcl(t, removalTemplateHCL+`
resource "zabbix_item_agent" "testremitem" {
	hostid    = zabbix_template.testremtmpl.id
	key       = "test.removal.graph.item"
	name      = "Removal Graph Item"
	valuetype = "unsigned"
}
resource "zabbix_graph" "testremgraph" {
	name   = "Removal Graph"
	width  = "900"
	height = "200"
`+body+`
	item {
		itemid = zabbix_item_agent.testremitem.id
		color  = "AA0000"
`+item+`
	}
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // every default written out as something else
				Config: graph(`
	type          = "stacked"
	do3d          = true
	legend        = false
	work_period   = false
	percent_left  = "10"
	percent_right = "20"
	ymin_type     = "fixed"
	ymin          = "5"
	ymax_type     = "fixed"
	ymax          = "50"
`, `
		drawtype   = "bold"
		function   = "average"
		sortorder  = "3"
		type       = "sum"
		yaxis_side = "right"
`),
				Check: testAccCheckServerAttrs(addr, serverGraph, map[string]string{
					"graphtype":        "1",
					"show_3d":          "1",
					"show_legend":      "0",
					"show_work_period": "0",
					"percent_left":     "10",
					"percent_right":    "20",
					"ymin_type":        "1",
					"ymax_type":        "1",
				}),
			},
			{ // and every one of those lines deleted again
				Config:           graph(``, ``),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerAttrs(addr, serverGraph, map[string]string{
						"graphtype":        "0",
						"show_3d":          "0",
						"show_legend":      "1",
						"show_work_period": "1",
						"percent_left":     "0",
						"percent_right":    "0",
						"ymin_type":        "0",
						"ymax_type":        "0",
						"yaxismin":         "0",
						"yaxismax":         "100",
					}),
					testAccCheckGraphItemFor(addr, "zabbix_item_agent.testremitem", map[string]string{
						"drawtype":  "0",
						"calc_fnc":  "1",
						"sortorder": "0",
						"type":      "0",
						"yaxisside": "0",
					}),
				),
			},
		},
	})
}

// TestAccRemoveItemDefaults owns the item defaults that are not HTTP-specific:
// itemCommonSchema's history and the preprocessing step's two error-handling
// attributes, itemDelaySchema's delay, and the agent backend's active.
//
// The item lives on a template so that `interfaceid` stays out of it; that one
// has its own test and its own answer.
func TestAccRemoveItemDefaults(t *testing.T) {
	const addr = "zabbix_item_agent.testremitem"

	item := func(body, preproc string) string {
		return hcl(t, removalTemplateHCL+`
resource "zabbix_item_agent" "testremitem" {
	hostid    = zabbix_template.testremtmpl.id
	key       = "test.removal.item"
	name      = "Removal Item"
	valuetype = "unsigned"
`+body+`
	preprocessor {
		type   = "multiplier"
		params = [ "10" ]
`+preproc+`
	}
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: item(`
	active  = true
	delay   = "30s"
	history = "7d"
`, `
		error_handler        = "2"
		error_handler_params = "42"
`),
				Check: testAccCheckServerAttrs(addr, serverItem, map[string]string{
					"type":                                 "7",
					"delay":                                "30s",
					"history":                              "7d",
					"preprocessing.0.error_handler":        "2",
					"preprocessing.0.error_handler_params": "42",
				}),
			},
			{
				Config:           item(``, ``),
				ConfigPlanChecks: expectUpdate(addr),
				Check: testAccCheckServerAttrs(addr, serverItem, map[string]string{
					"type":                                 "0",
					"delay":                                "1m",
					"history":                              "90d",
					"preprocessing.0.error_handler":        "0",
					"preprocessing.0.error_handler_params": "",
				}),
			},
		},
	})
}

// TestAccRemoveHttpItemDefaults owns the nine defaults the HTTP agent fragment
// adds. posts, username and password have no defaults and are cleared here
// only because the values they carry are meaningless once the attribute that
// gave them meaning is gone.
func TestAccRemoveHttpItemDefaults(t *testing.T) {
	const addr = "zabbix_item_http.testremhttp"

	item := func(body string) string {
		return hcl(t, removalTemplateHCL+`
resource "zabbix_item_http" "testremhttp" {
	hostid    = zabbix_template.testremtmpl.id
	key       = "test.removal.http"
	name      = "Removal Http Item"
	valuetype = "text"
	url       = "http://localhost/removal"
`+body+`
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: item(`
	auth_type        = "basic"
	username         = "removal"
	password         = "removal"
	follow_redirects = false
	post_type        = "json"
	posts            = "{\"a\":1}"
	request_method   = "post"
	retrieve_mode    = "headers"
	status_codes     = "200,201"
	timeout          = "10s"
	verify_host      = false
	verify_peer      = false
`),
				Check: testAccCheckServerAttrs(addr, serverItem, map[string]string{
					"authtype":         "1",
					"follow_redirects": "0",
					"post_type":        "2",
					"request_method":   "1",
					"retrieve_mode":    "1",
					"status_codes":     "200,201",
					"timeout":          "10s",
					"verify_host":      "0",
					"verify_peer":      "0",
				}),
			},
			{
				Config:           item(``),
				ConfigPlanChecks: expectUpdate(addr),
				Check: testAccCheckServerAttrs(addr, serverItem, map[string]string{
					"authtype":         "0",
					"follow_redirects": "1",
					"post_type":        "0",
					"request_method":   "0",
					"retrieve_mode":    "0",
					"status_codes":     "200",
					"timeout":          "3s",
					"verify_host":      "1",
					"verify_peer":      "1",
				}),
			},
		},
	})
}

// TestAccRemoveLLDDefaults owns the discovery-rule defaults: lifetime,
// evaltype and formula from lldCommonSchema, delay from lldDelaySchema, the
// filter condition's operator, and the preprocessing step's error handling.
//
// evaltype and formula have to move together. formula is only read under
// evaltype "custom", so the only configuration that can set it to a non-empty
// value is one that also sets evaltype, and deleting one line without the
// other would leave a formula Zabbix ignores.
func TestAccRemoveLLDDefaults(t *testing.T) {
	const addr = "zabbix_lld_agent.testremlld"

	lld := func(body string) string {
		return hcl(t, removalTemplateHCL+`
resource "zabbix_lld_agent" "testremlld" {
	hostid = zabbix_template.testremtmpl.id
	key    = "test.removal.lld"
	name   = "Removal LLD"
`+body+`
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: lld(`
	delay    = "600"
	lifetime = "45d"
	evaltype = "custom"
	formula  = "A and B"
	condition {
		id       = "A"
		macro    = "{#AAA}"
		value    = "one"
		operator = "notmatch"
	}
	condition {
		id       = "B"
		macro    = "{#BBB}"
		value    = "two"
		operator = "notmatch"
	}
	preprocessor {
		type                 = "jsonpath"
		params               = [ "$.a" ]
		error_handler        = "3"
		error_handler_params = "boom"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerAttrs(addr, serverLLD, map[string]string{
						"delay":                                "600",
						"lifetime":                             "45d",
						"filter.evaltype":                      "3",
						"filter.formula":                       "A and B",
						"preprocessing.0.error_handler":        "3",
						"preprocessing.0.error_handler_params": "boom",
					}),
					testAccCheckServerElem(addr, serverLLD, "filter.conditions", "macro", "{#AAA}", map[string]string{
						"operator": "9",
					}),
				),
			},
			{
				Config: lld(`
	condition {
		macro = "{#AAA}"
		value = "one"
	}
	condition {
		macro = "{#BBB}"
		value = "two"
	}
	preprocessor {
		type   = "jsonpath"
		params = [ "$.a" ]
	}
`),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerAttrs(addr, serverLLD, map[string]string{
						"delay":                                "3600",
						"lifetime":                             "30d",
						"filter.evaltype":                      "0",
						"filter.formula":                       "",
						"preprocessing.0.error_handler":        "0",
						"preprocessing.0.error_handler_params": "",
					}),
					testAccCheckServerElem(addr, serverLLD, "filter.conditions", "macro", "{#AAA}", map[string]string{
						"operator": "8",
					}),
				),
			},
		},
	})
}

// TestAccRemoveTriggerDefaults owns the five trigger defaults.
func TestAccRemoveTriggerDefaults(t *testing.T) {
	const addr = "zabbix_trigger.testremtrigger"

	trigger := func(body string) string {
		return hcl(t, removalTemplateHCL+`
resource "zabbix_item_trapper" "testremtriggeritem" {
	hostid    = zabbix_template.testremtmpl.id
	key       = "test.removal.trigger.item"
	name      = "Removal Trigger Item"
	valuetype = "unsigned"
}
resource "zabbix_trigger" "testremtrigger" {
	name       = "Removal Trigger"
	expression = "last(/test-removal-template/test.removal.trigger.item)=1"

	depends_on = [ zabbix_item_trapper.testremtriggeritem ]
`+body+`
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: trigger(`
	enabled       = false
	manual_close  = true
	multiple      = true
	priority      = "high"
	recovery_none = true
`),
				Check: testAccCheckServerAttrs(addr, serverTrigger, map[string]string{
					"status":        "1",
					"manual_close":  "1",
					"type":          "1",
					"priority":      "4",
					"recovery_mode": "2",
				}),
			},
			{
				Config:           trigger(``),
				ConfigPlanChecks: expectUpdate(addr),
				Check: testAccCheckServerAttrs(addr, serverTrigger, map[string]string{
					"status":        "0",
					"manual_close":  "0",
					"type":          "0",
					"priority":      "0",
					"recovery_mode": "0",
				}),
			},
		},
	})
}

// TestAccRemoveProxyDefaults owns the five proxy defaults, and needs three
// steps rather than two because they are not simultaneously reachable:
// address and port apply to a passive proxy only, tls_accept to an active one
// only, and the resource rejects the mismatch itself rather than letting
// Zabbix silently keep the value it had.
func TestAccRemoveProxyDefaults(t *testing.T) {
	const addr = "zabbix_proxy.testremproxy"

	proxy := func(body string) string {
		return `
resource "zabbix_proxy" "testremproxy" {
	name = "test-removal-proxy"
` + body + `
}
`
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // active, so tls_accept is the encryption attribute in play
				Config: proxy(`
	tls_accept       = "psk"
	tls_psk_identity = "test-removal-psk"
	tls_psk          = "b7e3b0e9a94e4f2a8d1c6f5b3a2e9d8c7b6a5f4e3d2c1b0a9f8e7d6c5b4a3f2e"
`),
				Check: testAccCheckServerAttrs(addr, serverProxy, map[string]string{
					"operating_mode": "0",
					"tls_accept":     "2",
				}),
			},
			{ // passive, which is where address, port and tls_connect live.
				// tls_accept's line is gone, and its default has to arrive.
				Config: proxy(`
	operating_mode   = "passive"
	address          = "10.20.30.40"
	port             = "10555"
	tls_connect      = "psk"
	tls_psk_identity = "test-removal-psk"
	tls_psk          = "b7e3b0e9a94e4f2a8d1c6f5b3a2e9d8c7b6a5f4e3d2c1b0a9f8e7d6c5b4a3f2e"
`),
				ConfigPlanChecks: expectUpdate(addr),
				Check: testAccCheckServerAttrs(addr, serverProxy, map[string]string{
					"operating_mode": "1",
					"address":        "10.20.30.40",
					"port":           "10555",
					"tls_connect":    "2",
					"tls_accept":     "1",
				}),
			},
			{ // the documented minimum: a proxy is a name and nothing else.
				//
				// address and port are the one pair whose default is not what
				// the server ends up holding: an active proxy has no endpoint,
				// so Zabbix keeps both empty and resource_proxy.go reports the
				// schema defaults back rather than the emptiness, which is
				// what keeps an active proxy free of a permanent diff. The
				// server-side assertion is therefore that the endpoint is
				// *gone*, and the defaults are asserted where they actually
				// live, in state.
				Config:           proxy(``),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerAttrs(addr, serverProxy, map[string]string{
						"operating_mode": "0",
						"address":        "",
						"port":           "",
						"tls_connect":    "1",
						"tls_accept":     "1",
					}),
					resource.TestCheckResourceAttr(addr, "address", "127.0.0.1"),
					resource.TestCheckResourceAttr(addr, "port", "10051"),
				),
			},
		},
	})
}

// TestAccRemoveTemplateDefaults owns wizard_ready, which only exists from 7.4.
func TestAccRemoveTemplateDefaults(t *testing.T) {
	const addr = "zabbix_template.testremwiz"

	tmpl := func(body string) string {
		return hcl(t, `
resource "zabbix_templategroup" "testremwizgrp" {
	name = "test-removal-wizard-group"
}
resource "zabbix_template" "testremwiz" {
	groups = [ zabbix_templategroup.testremwizgrp.id ]
	host   = "test-removal-wizard-template"
`+body+`
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config:   tmpl(`	wizard_ready = true`),
				SkipFunc: skipBelow(t, zabbix.V74),
				Check: testAccCheckServerAttrs(addr, serverTemplate, map[string]string{
					"wizard_ready": "1",
				}),
			},
			{
				Config:           tmpl(``),
				SkipFunc:         skipBelow(t, zabbix.V74),
				ConfigPlanChecks: expectUpdate(addr),
				Check: testAccCheckServerAttrs(addr, serverTemplate, map[string]string{
					"wizard_ready": "0",
				}),
			},
		},
	})
}

// removalInterfaceHostHCL gives the interfaceid tests a host with two agent
// interfaces, so that the attribute can hold a value that is not its default
// in the first place.
const removalInterfaceHostHCL = `
resource "zabbix_hostgroup" "testremifidgrp" {
	name = "test-removal-interfaceid-group"
}
resource "zabbix_host" "testremifidhost" {
	host   = "test-removal-interfaceid-host"
	groups = [ zabbix_hostgroup.testremifidgrp.id ]
	interface {
		ip   = "127.0.0.1"
		port = 10050
		main = true
	}
	interface {
		ip   = "127.0.0.2"
		port = 10051
		main = false
	}
}
`

// removalNoInterfaceRe matches what each supported server says when it is
// asked to put an item or a discovery rule on interface "0" while the host
// has interfaces. Probed directly, by calling item.update and
// discoveryrule.update with interfaceid "0":
//
//	6.0.48       "No interface found."
//	7.0.29       "Invalid parameter "/1/interfaceid": the host interface ID is expected."
//	7.4.13       same
//	8.0-trunk    same
//
// Omitting the property entirely fares no better -- 6.0 gives the same "No
// interface found." and 7.0+ "the parameter "interfaceid" is missing" -- so
// the default is not reachable by any encoding, on any version.
var removalNoInterfaceRe = regexp.MustCompile(`No interface found|host interface ID is expected`)

// TestAccRemoveItemInterfaceID owns itemInterfaceSchema, and it is the one R1
// attribute whose default cannot be applied.
//
// "0" means "no interface", and every supported server refuses it for an item
// on a host that has interfaces; on a template, where "0" is the only legal
// value, there is no id to set in the first place. So there is nowhere the
// line can be deleted and the default arrive, and what the test asserts is
// the failure: the revert is planned, the update is attempted, and the server
// says no. A user who deletes the line gets a clear error rather than an item
// silently left on the interface they were trying to move it off.
func TestAccRemoveItemInterfaceID(t *testing.T) {
	const addr = "zabbix_item_agent.testremifiditem"

	item := func(body string) string {
		return removalInterfaceHostHCL + `
resource "zabbix_item_agent" "testremifiditem" {
	hostid    = zabbix_host.testremifidhost.id
	key       = "test.removal.interfaceid"
	name      = "Removal InterfaceID Item"
	valuetype = "unsigned"
` + body + `
}
`
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: item(`	interfaceid = one([ for i in zabbix_host.testremifidhost.interface : i.id if i.port == 10051 ])`),
				Check:  testAccCheckItemInterfacePort(addr, "10051"),
			},
			{
				Config:      item(``),
				ExpectError: removalNoInterfaceRe,
			},
		},
	})
}

// TestAccRemoveLLDInterfaceID owns lldInterfaceSchema. Discovery rules do not
// share the item write path -- LLDRule carries its own copy of the field -- so
// the answer on one says nothing about the other, and discoveryrule.update was
// probed separately.
func TestAccRemoveLLDInterfaceID(t *testing.T) {
	const addr = "zabbix_lld_agent.testremifidlld"

	lld := func(body string) string {
		return removalInterfaceHostHCL + `
resource "zabbix_lld_agent" "testremifidlld" {
	hostid = zabbix_host.testremifidhost.id
	key    = "test.removal.lld.interfaceid"
	name   = "Removal InterfaceID LLD"
` + body + `
}
`
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: lld(`	interfaceid = one([ for i in zabbix_host.testremifidhost.interface : i.id if i.port == 10051 ])`),
				Check: testAccCheckServerAttrs(addr, serverInterfaceOf(serverLLD), map[string]string{
					"port": "10051",
				}),
			},
			{
				Config:      lld(``),
				ExpectError: removalNoInterfaceRe,
			},
		},
	})
}
