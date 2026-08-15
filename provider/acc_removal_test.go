package provider

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
var removalOwner = map[string]string{}

// removalExempt names the R1 attributes whose line cannot meaningfully be
// deleted, with the reason. Exemption is by name and never by omission.
var removalExempt = map[string]string{}

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
	"zabbix_graph.do3d":                                   "coverage pending",
	"zabbix_graph.item.drawtype":                          "coverage pending",
	"zabbix_graph.item.function":                          "coverage pending",
	"zabbix_graph.item.sortorder":                         "coverage pending",
	"zabbix_graph.item.type":                              "coverage pending",
	"zabbix_graph.item.yaxis_side":                        "coverage pending",
	"zabbix_graph.legend":                                 "coverage pending",
	"zabbix_graph.percent_left":                           "coverage pending",
	"zabbix_graph.percent_right":                          "coverage pending",
	"zabbix_graph.type":                                   "coverage pending",
	"zabbix_graph.work_period":                            "coverage pending",
	"zabbix_graph.ymax":                                   "coverage pending",
	"zabbix_graph.ymax_type":                              "coverage pending",
	"zabbix_graph.ymin":                                   "coverage pending",
	"zabbix_graph.ymin_type":                              "coverage pending",
	"zabbix_host.enabled":                                 "coverage pending",
	"zabbix_host.interface.main":                          "coverage pending",
	"zabbix_host.interface.port":                          "coverage pending",
	"zabbix_host.interface.snmp3_authpassphrase":          "coverage pending",
	"zabbix_host.interface.snmp3_authprotocol":            "coverage pending",
	"zabbix_host.interface.snmp3_contextname":             "coverage pending",
	"zabbix_host.interface.snmp3_privpassphrase":          "coverage pending",
	"zabbix_host.interface.snmp3_privprotocol":            "coverage pending",
	"zabbix_host.interface.snmp3_securitylevel":           "coverage pending",
	"zabbix_host.interface.snmp3_securityname":            "coverage pending",
	"zabbix_host.interface.snmp_bulk":                     "coverage pending",
	"zabbix_host.interface.snmp_community":                "coverage pending",
	"zabbix_host.interface.snmp_version":                  "coverage pending",
	"zabbix_host.interface.type":                          "coverage pending",
	"zabbix_host.inventory_mode":                          "coverage pending",
	"zabbix_host.ipmi_authtype":                           "coverage pending",
	"zabbix_host.ipmi_privilege":                          "coverage pending",
	"zabbix_host.name":                                    "coverage pending",
	"zabbix_host.proxyid":                                 "coverage pending",
	"zabbix_host.tls_accept":                              "coverage pending",
	"zabbix_host.tls_connect":                             "coverage pending",
	"zabbix_item_agent.active":                            "coverage pending",
	"zabbix_item_agent.delay":                             "coverage pending",
	"zabbix_item_agent.history":                           "coverage pending",
	"zabbix_item_agent.interfaceid":                       "coverage pending",
	"zabbix_item_agent.preprocessor.error_handler":        "coverage pending",
	"zabbix_item_agent.preprocessor.error_handler_params": "coverage pending",
	"zabbix_item_agent.trends":                            "coverage pending",
	"zabbix_item_http.auth_type":                          "coverage pending",
	"zabbix_item_http.follow_redirects":                   "coverage pending",
	"zabbix_item_http.post_type":                          "coverage pending",
	"zabbix_item_http.request_method":                     "coverage pending",
	"zabbix_item_http.retrieve_mode":                      "coverage pending",
	"zabbix_item_http.status_codes":                       "coverage pending",
	"zabbix_item_http.timeout":                            "coverage pending",
	"zabbix_item_http.verify_host":                        "coverage pending",
	"zabbix_item_http.verify_peer":                        "coverage pending",
	"zabbix_lld_agent.condition.operator":                 "coverage pending",
	"zabbix_lld_agent.delay":                              "coverage pending",
	"zabbix_lld_agent.evaltype":                           "coverage pending",
	"zabbix_lld_agent.formula":                            "coverage pending",
	"zabbix_lld_agent.interfaceid":                        "coverage pending",
	"zabbix_lld_agent.lifetime":                           "coverage pending",
	"zabbix_lld_agent.preprocessor.error_handler":         "coverage pending",
	"zabbix_lld_agent.preprocessor.error_handler_params":  "coverage pending",
	"zabbix_lld_dependent.delay":                          "coverage pending",
	"zabbix_lld_trapper.delay":                            "coverage pending",
	"zabbix_proto_trigger.correlation_mode":               "coverage pending",
	"zabbix_proto_trigger.enabled":                        "coverage pending",
	"zabbix_proto_trigger.manual_close":                   "coverage pending",
	"zabbix_proto_trigger.multiple":                       "coverage pending",
	"zabbix_proto_trigger.priority":                       "coverage pending",
	"zabbix_proto_trigger.recovery_none":                  "coverage pending",
	"zabbix_proxy.address":                                "coverage pending",
	"zabbix_proxy.operating_mode":                         "coverage pending",
	"zabbix_proxy.port":                                   "coverage pending",
	"zabbix_proxy.tls_accept":                             "coverage pending",
	"zabbix_proxy.tls_connect":                            "coverage pending",
	"zabbix_template.name":                                "coverage pending",
	"zabbix_template.wizard_ready":                        "coverage pending",
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
