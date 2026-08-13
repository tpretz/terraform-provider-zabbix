package provider

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// U1-U4 -- in-life updates (PLAN.md § "The unit of work").
//
// The suite covers create, plural collections, drift, negative paths, defaults
// and clearability. What none of them covers is the ordinary thing a user does
// most: change a value on a resource that already exists. Both failure modes
// have shipped in this release and both were found by accident:
//
//   - the provider permits an update Zabbix rejects. Item "hostid" became
//     create-only at 7.0 and prototype "ruleid" at 7.0 as well, which made
//     every zabbix_proto_item_* resource un-updatable on a current server.
//   - the provider marks ForceNew where Zabbix would have taken an update.
//     This one is worse than an inconvenience: destroying and recreating a
//     Zabbix item discards its entire history, so a spurious ForceNew is
//     silent data loss.
//
// | | Case | What it must do |
// |---|---|---|
// | U1 | changed in life | every settable attribute is changed on an existing object at least once, and the new value asserted against a *server re-read* rather than against Terraform state |
// | U2 | proven to be an update | the step carries plancheck.ExpectResourceAction(...Update), so a silent replace cannot pass for success |
// | U3 | create-only is ForceNew | every attribute Zabbix refuses to update is ForceNew and asserts replacement, in acc_forcenew_test.go's shape |
// | U4 | nothing else is ForceNew | no attribute is ForceNew that Zabbix would have accepted an update for. Probed against live servers, not inferred |
//
// U1 needs the server re-read for the same reason C6 does: Terraform state is
// written by the provider's own read function, so an update the provider never
// sent still looks applied in state whenever the read path shares the mistake.
// testAccCheckServerAttrs below re-reads the object straight from Zabbix and
// compares raw API properties.
//
// U2 is the check that catches a wrong ForceNew. Asserting the new value alone
// passes just as happily when Terraform destroyed the object and built a new
// one, which is precisely the case that loses a user's history.
//
// Where the coverage lives
// ------------------------
//
// Attributes are deduplicated by *pointer identity*: mergeSchemas copies
// *schema.Schema pointers, so all twenty zabbix_item_* and zabbix_proto_item_*
// resources share one "hostid" declaration, and covering it once covers it
// everywhere. TestUpdateCoverageComplete below does that grouping itself, from
// the provider's own ResourcesMap, so an attribute added tomorrow appears in
// the guard with no test change. That is the point of the guard: it stops the
// backlog regrowing, the way the collection and enum guards already do.
//
//	schema fragment           file                     coverage
//	------------------------  -----------------------  -------------------------
//	itemCommonSchema          common_item.go           TestAccUpdateItemAgent
//	itemDelaySchema           common_item.go           TestAccUpdateItemAgent
//	itemInterfaceSchema       common_item.go           TestAccUpdateItemInterfaceID
//	itemPreprocessorSchema    common_item.go           TestAccUpdateItemAgent
//	itemPrototypeSchema       common_item.go           TestAccForceNewProtoItemRuleid
//	tagSetSchema              common_tag.go            TestAccUpdateItemAgent
//	macroSetSchema            common_macro.go          TestAccUpdateTemplate
//	lldCommonSchema           common_lld.go            TestAccUpdateLLDAgent
//	lldDelaySchema            common_lld.go            TestAccUpdateLLDAgent
//	lldZeroDelaySchema        common_lld.go            exempt -- one valid value
//	lldInterfaceSchema        common_lld.go            TestAccUpdateLLDAgent
//	lldPreprocessorSchema     common_lld.go            TestAccUpdateLLDAgent
//	lldFilterConditionSchema  common_lld.go            TestAccUpdateLLDAgent
//	lldMacroPathSchema        common_lld.go            TestAccUpdateLLDAgent
//	http item/LLD fragment    resource_http_common.go  TestAccUpdateItemHttp
//	graph + graph item        resource_graph.go        TestAccUpdateGraph
//	trigger                   resource_trigger.go      TestAccUpdateTrigger
//	host, interface,          resource_host.go         TestAccUpdateHost,
//	inventory                                          TestAccUpdateHostInterface,
//	                                                   TestAccUpdateHostInventory
//	template                  resource_template.go     TestAccUpdateTemplate
//	proxy                     resource_proxy.go        TestAccUpdateProxy

// ---------------------------------------------------------------------------
// server re-read
// ---------------------------------------------------------------------------

// serverGetter re-reads one object from Zabbix by id and hands back its raw
// API properties. Raw rather than the typed client structs on purpose: the
// typed ones only carry the fields the provider happens to use, and a property
// the provider silently drops is exactly what U1 is looking for.
type serverGetter func(api *zabbix.API, id string) (map[string]interface{}, error)

// serverObject builds a serverGetter over a Zabbix `<object>.get` method.
func serverObject(method, idParam string, extra zabbix.Params) serverGetter {
	return func(api *zabbix.API, id string) (map[string]interface{}, error) {
		params := zabbix.Params{"output": "extend", idParam: []string{id}}
		for k, v := range extra {
			params[k] = v
		}
		var res []map[string]interface{}
		if err := api.CallWithErrorParse(method, params, &res); err != nil {
			return nil, err
		}
		if len(res) != 1 {
			return nil, fmt.Errorf("%s returned %d objects for id %s, want 1", method, len(res), id)
		}
		return res[0], nil
	}
}

var (
	serverItem = serverObject("item.get", "itemids", zabbix.Params{
		"selectTags": "extend", "selectPreprocessing": "extend",
	})
	serverProtoItem = serverObject("itemprototype.get", "itemids", zabbix.Params{
		"selectTags": "extend", "selectPreprocessing": "extend",
	})
	serverLLD = serverObject("discoveryrule.get", "itemids", zabbix.Params{
		"selectPreprocessing": "extend", "selectFilter": "extend", "selectLLDMacroPaths": "extend",
	})
	serverTemplate = serverObject("template.get", "templateids", zabbix.Params{
		"selectMacros": "extend", "selectParentTemplates": "extend",
	})
	serverHostGroup     = serverObject("hostgroup.get", "groupids", nil)
	serverTemplateGroup = serverObject("templategroup.get", "groupids", nil)
	serverGraph         = serverObject("graph.get", "graphids", zabbix.Params{"selectGraphItems": "extend"})
	serverProtoGraph    = serverObject("graphprototype.get", "graphids", zabbix.Params{"selectGraphItems": "extend"})
	serverTrigger       = serverObject("trigger.get", "triggerids", zabbix.Params{
		"selectTags": "extend", "selectDependencies": "extend", "expandExpression": true,
	})
	serverProtoTrigger = serverObject("triggerprototype.get", "triggerids", zabbix.Params{
		"selectTags": "extend", "selectDependencies": "extend", "expandExpression": true,
	})
)

// serverHost needs the group select parameter Zabbix renamed in 7.2, so it is
// written out rather than built by serverObject.
func serverHost(api *zabbix.API, id string) (map[string]interface{}, error) {
	return serverObject("host.get", "hostids", zabbix.Params{
		hostGroupSelectParam(api): "extend",
		"selectInterfaces":        "extend",
		"selectMacros":            "extend",
		"selectTags":              "extend",
		"selectInventory":         "extend",
		"selectParentTemplates":   "extend",
	})(api, id)
}

// serverProxy reads through the typed client rather than raw JSON because the
// proxy object was renamed property by property in 7.0 -- host/name,
// status/operating_mode, a nested interface object that became plain
// address+port -- and ProxiesGet is where that translation already lives.
// Re-implementing it here would test the test.
func serverProxy(api *zabbix.API, id string) (map[string]interface{}, error) {
	res, err := api.ProxiesGet(zabbix.Params{"proxyids": id})
	if err != nil {
		return nil, err
	}
	if len(res) != 1 {
		return nil, fmt.Errorf("proxy.get returned %d proxies for id %s, want 1", len(res), id)
	}
	p := res[0]
	return map[string]interface{}{
		"name":              p.Name,
		"operating_mode":    strconv.Itoa(int(p.Mode)),
		"address":           p.Address,
		"port":              p.Port,
		"allowed_addresses": p.AllowedAddresses,
		"description":       p.Description,
		"tls_connect":       string(p.TLSConnect),
		"tls_accept":        string(p.TLSAccept),
		"tls_issuer":        p.TLSIssuer,
		"tls_subject":       p.TLSSubject,
	}, nil
}

// serverValue resolves a dotted path through the decoded JSON of an object:
// map keys by name, array elements by index. "preprocessing.0.params" and
// "filter.conditions.1.macro" both work.
func serverValue(obj interface{}, path string) (interface{}, bool) {
	cur := obj
	for _, part := range strings.Split(path, ".") {
		switch v := cur.(type) {
		case map[string]interface{}:
			next, ok := v[part]
			if !ok {
				return nil, false
			}
			cur = next
		case []interface{}:
			i, err := strconv.Atoi(part)
			if err != nil || i < 0 || i >= len(v) {
				return nil, false
			}
			cur = v[i]
		default:
			return nil, false
		}
	}
	return cur, true
}

// serverString renders a decoded JSON value the way a test wants to compare
// it. Zabbix returns almost everything as a string; the rest is rendered
// canonically so a want value can be written by hand.
func serverString(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprintf("%v", t)
		}
		return string(b)
	}
}

// testAccCheckServerAttrs re-reads addr's object from Zabbix and requires the
// named API properties to hold the given values. This is U1: the assertion is
// against the server, not against the state the provider wrote itself.
func testAccCheckServerAttrs(addr string, get serverGetter, want map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		api, ok := testAccProvider.Meta().(*zabbix.API)
		if !ok || api == nil {
			return fmt.Errorf("testAccCheckServerAttrs: provider not configured")
		}
		id, err := testAccStateID(s, addr)
		if err != nil {
			return err
		}
		obj, err := get(api, id)
		if err != nil {
			return fmt.Errorf("%s (%s): re-reading from the server: %s", addr, id, err)
		}

		paths := make([]string, 0, len(want))
		for p := range want {
			paths = append(paths, p)
		}
		sort.Strings(paths)

		var errs []string
		for _, p := range paths {
			got, ok := serverValue(obj, p)
			if !ok {
				errs = append(errs, fmt.Sprintf("%s: the server did not return it at all", p))
				continue
			}
			if g := serverString(got); g != want[p] {
				errs = append(errs, fmt.Sprintf("%s: server has %q, want %q", p, g, want[p]))
			}
		}
		if len(errs) > 0 {
			return fmt.Errorf("%s (%s) after update:\n\t%s", addr, id, strings.Join(errs, "\n\t"))
		}
		return nil
	}
}

// testAccCheckServerElem is the collection form of testAccCheckServerAttrs:
// find the element of the array at arrayPath whose keyProp equals keyVal, and
// require the rest of its properties to match. Elements are located by content
// because Zabbix's return order for a set-like collection is not stable across
// versions.
func testAccCheckServerElem(addr string, get serverGetter, arrayPath, keyProp, keyVal string, want map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		api, ok := testAccProvider.Meta().(*zabbix.API)
		if !ok || api == nil {
			return fmt.Errorf("testAccCheckServerElem: provider not configured")
		}
		id, err := testAccStateID(s, addr)
		if err != nil {
			return err
		}
		obj, err := get(api, id)
		if err != nil {
			return fmt.Errorf("%s (%s): re-reading from the server: %s", addr, id, err)
		}
		raw, ok := serverValue(obj, arrayPath)
		if !ok {
			return fmt.Errorf("%s (%s): the server returned no %s", addr, id, arrayPath)
		}
		arr, ok := raw.([]interface{})
		if !ok {
			return fmt.Errorf("%s (%s): %s is %T, not an array", addr, id, arrayPath, raw)
		}

		for _, e := range arr {
			v, ok := serverValue(e, keyProp)
			if !ok || serverString(v) != keyVal {
				continue
			}
			paths := make([]string, 0, len(want))
			for p := range want {
				paths = append(paths, p)
			}
			sort.Strings(paths)

			var errs []string
			for _, p := range paths {
				got, ok := serverValue(e, p)
				if !ok {
					errs = append(errs, fmt.Sprintf("%s: absent", p))
					continue
				}
				if g := serverString(got); g != want[p] {
					errs = append(errs, fmt.Sprintf("%s: server has %q, want %q", p, g, want[p]))
				}
			}
			if len(errs) > 0 {
				return fmt.Errorf("%s (%s) %s[%s=%s]:\n\t%s", addr, id, arrayPath, keyProp, keyVal, strings.Join(errs, "\n\t"))
			}
			return nil
		}
		body, _ := json.Marshal(arr)
		return fmt.Errorf("%s (%s): no element of %s has %s == %q; server returned %s",
			addr, id, arrayPath, keyProp, keyVal, body)
	}
}

// expectUpdate is U2. Every step that changes a value on an existing object
// carries it, so that a replacement -- which loses an item's history -- cannot
// pass for a successful update.
func expectUpdate(addrs ...string) resource.ConfigPlanChecks {
	checks := make([]plancheck.PlanCheck, 0, len(addrs))
	for _, a := range addrs {
		checks = append(checks, plancheck.ExpectResourceAction(a, plancheck.ResourceActionUpdate))
	}
	return resource.ConfigPlanChecks{PreApply: checks}
}

// ---------------------------------------------------------------------------
// the completeness guard
// ---------------------------------------------------------------------------

// updateOwner names, for every settable attribute, the acceptance test that
// changes it on an object that already exists.
//
// A key is "<resource>.<dotted attribute path>". Only one resource per shared
// declaration needs naming: attributes are grouped by *pointer identity*
// before the lookup, so "zabbix_item_agent.hostid" satisfies the same
// declaration in all twenty item and prototype resources. Which resource is
// named is therefore a statement about where the fixture lives, nothing more.
var updateOwner = map[string]string{
	"zabbix_item_agent.active":                            "TestAccUpdateItemAgent",
	"zabbix_item_agent.delay":                             "TestAccUpdateItemAgent",
	"zabbix_item_agent.history":                           "TestAccUpdateItemAgent",
	"zabbix_item_agent.interfaceid":                       "TestAccUpdateItemInterfaceID",
	"zabbix_item_agent.key":                               "TestAccUpdateItemAgent",
	"zabbix_item_agent.name":                              "TestAccUpdateItemAgent",
	"zabbix_item_agent.preprocessor":                      "TestAccUpdateItemAgent",
	"zabbix_item_agent.preprocessor.error_handler":        "TestAccUpdateItemAgent",
	"zabbix_item_agent.preprocessor.error_handler_params": "TestAccUpdateItemAgent",
	"zabbix_item_agent.preprocessor.params":               "TestAccUpdateItemAgent",
	"zabbix_item_agent.preprocessor.type":                 "TestAccUpdateItemAgent",
	"zabbix_item_agent.tag":                               "TestAccUpdateItemAgent",
	"zabbix_item_agent.tag.key":                           "TestAccUpdateItemAgent",
	"zabbix_item_agent.tag.value":                         "TestAccUpdateItemAgent",
	"zabbix_item_agent.trends":                            "TestAccUpdateItemAgent",
	"zabbix_item_agent.valuetype":                         "TestAccUpdateItemAgent",
	"zabbix_item_calculated.formula":                      "TestAccUpdateItemTypeSpecific",
	"zabbix_item_dependent.master_itemid":                 "TestAccUpdateItemTypeSpecific",
	"zabbix_item_http.auth_type":                          "TestAccUpdateItemHttp",
	"zabbix_item_http.follow_redirects":                   "TestAccUpdateItemHttp",
	"zabbix_item_http.headers":                            "TestAccUpdateItemHttp",
	"zabbix_item_http.password":                           "TestAccUpdateItemHttp",
	"zabbix_item_http.post_type":                          "TestAccUpdateItemHttp",
	"zabbix_item_http.posts":                              "TestAccUpdateItemHttp",
	"zabbix_item_http.proxy":                              "TestAccUpdateItemHttp",
	"zabbix_item_http.request_method":                     "TestAccUpdateItemHttp",
	"zabbix_item_http.retrieve_mode":                      "TestAccUpdateItemHttp",
	"zabbix_item_http.status_codes":                       "TestAccUpdateItemHttp",
	"zabbix_item_http.timeout":                            "TestAccUpdateItemHttp",
	"zabbix_item_http.url":                                "TestAccUpdateItemHttp",
	"zabbix_item_http.username":                           "TestAccUpdateItemHttp",
	"zabbix_item_http.verify_host":                        "TestAccUpdateItemHttp",
	"zabbix_item_http.verify_peer":                        "TestAccUpdateItemHttp",
	"zabbix_item_snmp.snmp_oid":                           "TestAccUpdateItemTypeSpecific",
}

// updateForceNew is U3 and U4 together: the attributes Zabbix refuses to
// update, each with the test that proves the resource is replaced and the
// probe that established the refusal. The guard requires this list and the
// schema's ForceNew flags to be the same set in both directions, so a ForceNew
// added without a live probe fails here rather than silently costing a user
// their item history.
// All three were probed directly against live 6.0.48, 7.0.29, 7.4.13 and
// 8.0-trunk by calling the update method with the attribute changed:
//
//	item.update hostid           6.0: "Incorrect value for field \"hostid\": cannot be changed."
//	                             7.0/7.4/8.0: "unexpected parameter \"hostid\""
//	discoveryrule.update hostid  same on all four
//	itemprototype.update ruleid  6.0: ACCEPTED and silently ignored -- the
//	                             prototype stayed on its original rule
//	                             7.0/7.4/8.0: "unexpected parameter \"ruleid\""
//
// So all three are correct, and ruleid is correct on 6.0 for the worse of the
// two reasons: the call succeeds and does nothing, which without ForceNew
// would leave Terraform reporting a move that never happened.
var updateForceNew = map[string]string{
	"zabbix_item_agent.hostid":       "TestAccForceNewItemHostid; item.update rejects hostid on 6.0-8.0",
	"zabbix_lld_agent.hostid":        "TestAccForceNewLLDHostid; discoveryrule.update rejects hostid on 6.0-8.0",
	"zabbix_proto_item_agent.ruleid": "TestAccForceNewProtoItemRuleid; itemprototype.update rejects ruleid from 7.0 and ignores it on 6.0",
}

// updateExempt names the attributes that cannot be changed in life in any
// meaningful way, with the reason. Exemption is by name and never by omission:
// an attribute that quietly fell out of the list is exactly how prototype
// "ruleid" stayed unnoticed until it broke every prototype resource on 7.0.
var updateExempt = map[string]string{
	// lldZeroDelaySchema. Trapper and dependent discovery rules are not
	// polled, and Zabbix rejects any delay but "0" on them. The attribute
	// exists only so that the shared read path stays total -- see
	// CONTRIBUTING.md § "The item / prototype / LLD triad" -- and its
	// StringInSlice permits exactly one value, so there is no second value to
	// change it to.
	"zabbix_lld_trapper.delay":   "pinned to \"0\": a trapper discovery rule has no polling interval and the enum permits one value",
	"zabbix_lld_dependent.delay": "pinned to \"0\": a dependent discovery rule has no polling interval and the enum permits one value",
}

// updatePending is a migration scaffold, not a second exemption list. It holds
// the attributes whose update coverage is still being written, so that the
// guard can land before the coverage does and every commit in between stays
// green. It must be empty by the end of the series, which
// TestUpdateCoverageComplete enforces once it is.
var updatePending = map[string]string{
	"zabbix_graph.do3d":                                  "coverage pending",
	"zabbix_graph.height":                                "coverage pending",
	"zabbix_graph.item":                                  "coverage pending",
	"zabbix_graph.item.color":                            "coverage pending",
	"zabbix_graph.item.drawtype":                         "coverage pending",
	"zabbix_graph.item.function":                         "coverage pending",
	"zabbix_graph.item.itemid":                           "coverage pending",
	"zabbix_graph.item.sortorder":                        "coverage pending",
	"zabbix_graph.item.type":                             "coverage pending",
	"zabbix_graph.item.yaxis_side":                       "coverage pending",
	"zabbix_graph.legend":                                "coverage pending",
	"zabbix_graph.name":                                  "coverage pending",
	"zabbix_graph.percent_left":                          "coverage pending",
	"zabbix_graph.percent_right":                         "coverage pending",
	"zabbix_graph.type":                                  "coverage pending",
	"zabbix_graph.width":                                 "coverage pending",
	"zabbix_graph.work_period":                           "coverage pending",
	"zabbix_graph.ymax":                                  "coverage pending",
	"zabbix_graph.ymax_itemid":                           "coverage pending",
	"zabbix_graph.ymax_type":                             "coverage pending",
	"zabbix_graph.ymin":                                  "coverage pending",
	"zabbix_graph.ymin_itemid":                           "coverage pending",
	"zabbix_graph.ymin_type":                             "coverage pending",
	"zabbix_host.enabled":                                "coverage pending",
	"zabbix_host.groups":                                 "coverage pending",
	"zabbix_host.host":                                   "coverage pending",
	"zabbix_host.interface":                              "coverage pending",
	"zabbix_host.interface.dns":                          "coverage pending",
	"zabbix_host.interface.ip":                           "coverage pending",
	"zabbix_host.interface.main":                         "coverage pending",
	"zabbix_host.interface.port":                         "coverage pending",
	"zabbix_host.interface.snmp3_authpassphrase":         "coverage pending",
	"zabbix_host.interface.snmp3_authprotocol":           "coverage pending",
	"zabbix_host.interface.snmp3_contextname":            "coverage pending",
	"zabbix_host.interface.snmp3_privpassphrase":         "coverage pending",
	"zabbix_host.interface.snmp3_privprotocol":           "coverage pending",
	"zabbix_host.interface.snmp3_securitylevel":          "coverage pending",
	"zabbix_host.interface.snmp3_securityname":           "coverage pending",
	"zabbix_host.interface.snmp_bulk":                    "coverage pending",
	"zabbix_host.interface.snmp_community":               "coverage pending",
	"zabbix_host.interface.snmp_version":                 "coverage pending",
	"zabbix_host.interface.type":                         "coverage pending",
	"zabbix_host.inventory":                              "coverage pending",
	"zabbix_host.inventory.alias":                        "coverage pending",
	"zabbix_host.inventory.asset_tag":                    "coverage pending",
	"zabbix_host.inventory.chassis":                      "coverage pending",
	"zabbix_host.inventory.contact":                      "coverage pending",
	"zabbix_host.inventory.contract_number":              "coverage pending",
	"zabbix_host.inventory.date_hw_decomm":               "coverage pending",
	"zabbix_host.inventory.date_hw_expiry":               "coverage pending",
	"zabbix_host.inventory.date_hw_install":              "coverage pending",
	"zabbix_host.inventory.date_hw_purchase":             "coverage pending",
	"zabbix_host.inventory.deployment_status":            "coverage pending",
	"zabbix_host.inventory.hardware":                     "coverage pending",
	"zabbix_host.inventory.hardware_full":                "coverage pending",
	"zabbix_host.inventory.host_netmask":                 "coverage pending",
	"zabbix_host.inventory.host_networks":                "coverage pending",
	"zabbix_host.inventory.host_router":                  "coverage pending",
	"zabbix_host.inventory.hw_arch":                      "coverage pending",
	"zabbix_host.inventory.installer_name":               "coverage pending",
	"zabbix_host.inventory.location":                     "coverage pending",
	"zabbix_host.inventory.location_lat":                 "coverage pending",
	"zabbix_host.inventory.location_lon":                 "coverage pending",
	"zabbix_host.inventory.macaddress_a":                 "coverage pending",
	"zabbix_host.inventory.macaddress_b":                 "coverage pending",
	"zabbix_host.inventory.model":                        "coverage pending",
	"zabbix_host.inventory.name":                         "coverage pending",
	"zabbix_host.inventory.notes":                        "coverage pending",
	"zabbix_host.inventory.oob_ip":                       "coverage pending",
	"zabbix_host.inventory.oob_netmask":                  "coverage pending",
	"zabbix_host.inventory.oob_router":                   "coverage pending",
	"zabbix_host.inventory.os":                           "coverage pending",
	"zabbix_host.inventory.os_full":                      "coverage pending",
	"zabbix_host.inventory.os_short":                     "coverage pending",
	"zabbix_host.inventory.poc_1_cell":                   "coverage pending",
	"zabbix_host.inventory.poc_1_email":                  "coverage pending",
	"zabbix_host.inventory.poc_1_name":                   "coverage pending",
	"zabbix_host.inventory.poc_1_notes":                  "coverage pending",
	"zabbix_host.inventory.poc_1_phone_a":                "coverage pending",
	"zabbix_host.inventory.poc_1_phone_b":                "coverage pending",
	"zabbix_host.inventory.poc_1_screen":                 "coverage pending",
	"zabbix_host.inventory.poc_2_cell":                   "coverage pending",
	"zabbix_host.inventory.poc_2_email":                  "coverage pending",
	"zabbix_host.inventory.poc_2_name":                   "coverage pending",
	"zabbix_host.inventory.poc_2_notes":                  "coverage pending",
	"zabbix_host.inventory.poc_2_phone_a":                "coverage pending",
	"zabbix_host.inventory.poc_2_phone_b":                "coverage pending",
	"zabbix_host.inventory.poc_2_screen":                 "coverage pending",
	"zabbix_host.inventory.serialno_a":                   "coverage pending",
	"zabbix_host.inventory.serialno_b":                   "coverage pending",
	"zabbix_host.inventory.site_address_a":               "coverage pending",
	"zabbix_host.inventory.site_address_b":               "coverage pending",
	"zabbix_host.inventory.site_address_c":               "coverage pending",
	"zabbix_host.inventory.site_city":                    "coverage pending",
	"zabbix_host.inventory.site_country":                 "coverage pending",
	"zabbix_host.inventory.site_notes":                   "coverage pending",
	"zabbix_host.inventory.site_rack":                    "coverage pending",
	"zabbix_host.inventory.site_state":                   "coverage pending",
	"zabbix_host.inventory.site_zip":                     "coverage pending",
	"zabbix_host.inventory.software":                     "coverage pending",
	"zabbix_host.inventory.software_app_a":               "coverage pending",
	"zabbix_host.inventory.software_app_b":               "coverage pending",
	"zabbix_host.inventory.software_app_c":               "coverage pending",
	"zabbix_host.inventory.software_app_d":               "coverage pending",
	"zabbix_host.inventory.software_app_e":               "coverage pending",
	"zabbix_host.inventory.software_full":                "coverage pending",
	"zabbix_host.inventory.tag":                          "coverage pending",
	"zabbix_host.inventory.type":                         "coverage pending",
	"zabbix_host.inventory.type_full":                    "coverage pending",
	"zabbix_host.inventory.url_a":                        "coverage pending",
	"zabbix_host.inventory.url_b":                        "coverage pending",
	"zabbix_host.inventory.url_c":                        "coverage pending",
	"zabbix_host.inventory.vendor":                       "coverage pending",
	"zabbix_host.inventory_mode":                         "coverage pending",
	"zabbix_host.ipmi_authtype":                          "coverage pending",
	"zabbix_host.ipmi_password":                          "coverage pending",
	"zabbix_host.ipmi_privilege":                         "coverage pending",
	"zabbix_host.ipmi_username":                          "coverage pending",
	"zabbix_host.macro":                                  "coverage pending",
	"zabbix_host.macro.name":                             "coverage pending",
	"zabbix_host.macro.value":                            "coverage pending",
	"zabbix_host.name":                                   "coverage pending",
	"zabbix_host.proxyid":                                "coverage pending",
	"zabbix_host.tag":                                    "coverage pending",
	"zabbix_host.tag.key":                                "coverage pending",
	"zabbix_host.tag.value":                              "coverage pending",
	"zabbix_host.templates":                              "coverage pending",
	"zabbix_host.tls_accept":                             "coverage pending",
	"zabbix_host.tls_connect":                            "coverage pending",
	"zabbix_host.tls_issuer":                             "coverage pending",
	"zabbix_host.tls_psk":                                "coverage pending",
	"zabbix_host.tls_psk_identity":                       "coverage pending",
	"zabbix_host.tls_subject":                            "coverage pending",
	"zabbix_hostgroup.name":                              "coverage pending",
	"zabbix_item_agent.active":                           "coverage pending",
	"zabbix_item_agent.delay":                            "coverage pending",
	"zabbix_item_agent.history":                          "coverage pending",
	"zabbix_item_agent.interfaceid":                      "coverage pending",
	"zabbix_item_agent.key":                              "coverage pending",
	"zabbix_item_agent.name":                             "coverage pending",
	"zabbix_item_agent.preprocessor":                     "coverage pending",
	"zabbix_item_agent.preprocessor.error_handler":       "coverage pending",
	"zabbix_item_agent.preprocessor.params":              "coverage pending",
	"zabbix_item_agent.preprocessor.type":                "coverage pending",
	"zabbix_item_agent.tag":                              "coverage pending",
	"zabbix_item_agent.trends":                           "coverage pending",
	"zabbix_item_agent.valuetype":                        "coverage pending",
	"zabbix_item_calculated.formula":                     "coverage pending",
	"zabbix_item_dependent.master_itemid":                "coverage pending",
	"zabbix_item_http.auth_type":                         "coverage pending",
	"zabbix_item_http.follow_redirects":                  "coverage pending",
	"zabbix_item_http.headers":                           "coverage pending",
	"zabbix_item_http.password":                          "coverage pending",
	"zabbix_item_http.post_type":                         "coverage pending",
	"zabbix_item_http.posts":                             "coverage pending",
	"zabbix_item_http.proxy":                             "coverage pending",
	"zabbix_item_http.request_method":                    "coverage pending",
	"zabbix_item_http.retrieve_mode":                     "coverage pending",
	"zabbix_item_http.status_codes":                      "coverage pending",
	"zabbix_item_http.timeout":                           "coverage pending",
	"zabbix_item_http.url":                               "coverage pending",
	"zabbix_item_http.username":                          "coverage pending",
	"zabbix_item_http.verify_host":                       "coverage pending",
	"zabbix_item_http.verify_peer":                       "coverage pending",
	"zabbix_item_snmp.snmp_oid":                          "coverage pending",
	"zabbix_lld_agent.condition":                         "coverage pending",
	"zabbix_lld_agent.condition.id":                      "coverage pending",
	"zabbix_lld_agent.condition.macro":                   "coverage pending",
	"zabbix_lld_agent.condition.operator":                "coverage pending",
	"zabbix_lld_agent.condition.value":                   "coverage pending",
	"zabbix_lld_agent.delay":                             "coverage pending",
	"zabbix_lld_agent.evaltype":                          "coverage pending",
	"zabbix_lld_agent.formula":                           "coverage pending",
	"zabbix_lld_agent.interfaceid":                       "coverage pending",
	"zabbix_lld_agent.key":                               "coverage pending",
	"zabbix_lld_agent.lifetime":                          "coverage pending",
	"zabbix_lld_agent.macro":                             "coverage pending",
	"zabbix_lld_agent.macro.macro":                       "coverage pending",
	"zabbix_lld_agent.macro.path":                        "coverage pending",
	"zabbix_lld_agent.name":                              "coverage pending",
	"zabbix_lld_agent.preprocessor":                      "coverage pending",
	"zabbix_lld_agent.preprocessor.error_handler":        "coverage pending",
	"zabbix_lld_agent.preprocessor.error_handler_params": "coverage pending",
	"zabbix_lld_agent.preprocessor.params":               "coverage pending",
	"zabbix_lld_agent.preprocessor.type":                 "coverage pending",
	"zabbix_proto_trigger.comments":                      "coverage pending",
	"zabbix_proto_trigger.correlation_mode":              "coverage pending",
	"zabbix_proto_trigger.correlation_tag":               "coverage pending",
	"zabbix_proto_trigger.dependencies":                  "coverage pending",
	"zabbix_proto_trigger.enabled":                       "coverage pending",
	"zabbix_proto_trigger.event_name":                    "coverage pending",
	"zabbix_proto_trigger.expression":                    "coverage pending",
	"zabbix_proto_trigger.manual_close":                  "coverage pending",
	"zabbix_proto_trigger.multiple":                      "coverage pending",
	"zabbix_proto_trigger.name":                          "coverage pending",
	"zabbix_proto_trigger.opdata":                        "coverage pending",
	"zabbix_proto_trigger.priority":                      "coverage pending",
	"zabbix_proto_trigger.recovery_expression":           "coverage pending",
	"zabbix_proto_trigger.recovery_none":                 "coverage pending",
	"zabbix_proto_trigger.url":                           "coverage pending",
	"zabbix_proxy.address":                               "coverage pending",
	"zabbix_proxy.allowed_addresses":                     "coverage pending",
	"zabbix_proxy.description":                           "coverage pending",
	"zabbix_proxy.name":                                  "coverage pending",
	"zabbix_proxy.operating_mode":                        "coverage pending",
	"zabbix_proxy.port":                                  "coverage pending",
	"zabbix_proxy.tls_accept":                            "coverage pending",
	"zabbix_proxy.tls_connect":                           "coverage pending",
	"zabbix_proxy.tls_issuer":                            "coverage pending",
	"zabbix_proxy.tls_psk":                               "coverage pending",
	"zabbix_proxy.tls_psk_identity":                      "coverage pending",
	"zabbix_proxy.tls_subject":                           "coverage pending",
	"zabbix_template.description":                        "coverage pending",
	"zabbix_template.groups":                             "coverage pending",
	"zabbix_template.host":                               "coverage pending",
	"zabbix_template.macro":                              "coverage pending",
	"zabbix_template.name":                               "coverage pending",
	"zabbix_template.readme":                             "coverage pending",
	"zabbix_template.templates":                          "coverage pending",
	"zabbix_template.vendor_name":                        "coverage pending",
	"zabbix_template.vendor_version":                     "coverage pending",
	"zabbix_template.wizard_ready":                       "coverage pending",
	"zabbix_templategroup.name":                          "coverage pending",
}

// updateAttrSite is one place an attribute declaration is reachable from
// configuration.
type updateAttrSite struct {
	resource string
	path     string
}

func (s updateAttrSite) key() string { return s.resource + "." + s.path }

// updateAttrGroups groups every settable attribute of every *resource* by the
// declaration it comes from. Data sources are excluded: they are read-only, so
// "change it in life" is not a question they have.
func updateAttrGroups() map[*schema.Schema][]updateAttrSite {
	groups := map[*schema.Schema][]updateAttrSite{}
	p := Provider()

	names := make([]string, 0, len(p.ResourcesMap))
	for name := range p.ResourcesMap {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		walkSchema(p.ResourcesMap[name].Schema, "", func(path string, s *schema.Schema) {
			if !s.Optional && !s.Required {
				return // computed-only: nothing a user can change
			}
			groups[s] = append(groups[s], updateAttrSite{name, path})
		})
	}
	return groups
}

// TestUpdateCoverageComplete is U1's completeness half, and the reason this
// work is a guard rather than a backlog: an attribute added to any resource
// tomorrow fails here until somebody either covers it or writes down why it
// cannot be covered.
func TestUpdateCoverageComplete(t *testing.T) {
	groups := updateAttrGroups()
	if len(groups) == 0 {
		t.Fatal("no settable attributes found at all; updateAttrGroups has stopped working")
	}

	// every registry key must name an attribute that exists, or the registry
	// is quietly claiming coverage of something that has been renamed away
	live := map[string]bool{}
	for _, sites := range groups {
		for _, s := range sites {
			live[s.key()] = true
		}
	}
	for _, reg := range []struct {
		name string
		m    map[string]string
	}{
		{"updateOwner", updateOwner},
		{"updateForceNew", updateForceNew},
		{"updateExempt", updateExempt},
		{"updatePending", updatePending},
	} {
		keys := make([]string, 0, len(reg.m))
		for k := range reg.m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if !live[k] {
				t.Errorf("%s has %q, which is not a settable attribute of any resource", reg.name, k)
			}
			if reg.m[k] == "" {
				t.Errorf("%s[%q] has no %s", reg.name, k, map[string]string{
					"updateOwner": "owning test", "updateForceNew": "test and probe",
					"updateExempt": "reason", "updatePending": "note",
				}[reg.name])
			}
		}
	}
	for k := range updateOwner {
		if _, dup := updateExempt[k]; dup {
			t.Errorf("%q is both covered and exempt; it can only be one", k)
		}
		if _, dup := updateForceNew[k]; dup {
			t.Errorf("%q is claimed as both changed in life and ForceNew", k)
		}
	}

	// U4, at the schema level: a ForceNew flag is a claim about the server,
	// and every one of them has to be a claim somebody probed. Both directions
	// are checked, so neither adding a ForceNew nor removing one can slip past
	// the probe list.
	for sch, sites := range groups {
		declared := false
		for _, s := range sites {
			if _, ok := updateForceNew[s.key()]; ok {
				declared = true
				break
			}
		}
		switch {
		case sch.ForceNew && !declared:
			t.Errorf("%s is ForceNew but is not in updateForceNew; a ForceNew nobody probed against a live server is how an item loses its history for nothing", sites[0].key())
		case !sch.ForceNew && declared:
			t.Errorf("%s is in updateForceNew but the schema does not mark it ForceNew", sites[0].key())
		}
	}

	covered, forced, exempt, pending := 0, 0, 0, 0
	var missing []string
	for _, sites := range groups {
		// precedence, not site order: while coverage is being written a
		// declaration reachable from several resources may be named in more
		// than one registry, and the strongest claim is the true one
		state := ""
		for _, reg := range []struct {
			name string
			m    map[string]string
		}{
			{"covered", updateOwner},
			{"forcenew", updateForceNew},
			{"exempt", updateExempt},
			{"pending", updatePending},
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
		case "forcenew":
			forced++
		case "exempt":
			exempt++
		case "pending":
			pending++
		default:
			keys := make([]string, 0, len(sites))
			for _, s := range sites {
				keys = append(keys, s.key())
			}
			sort.Strings(keys)
			shown := keys
			if len(shown) > 4 {
				shown = append(append([]string{}, shown[:4]...), fmt.Sprintf("... and %d more", len(keys)-4))
			}
			missing = append(missing, strings.Join(shown, ", "))
		}
	}

	sort.Strings(missing)
	for _, m := range missing {
		t.Errorf("no in-life update coverage: %s\n\tadd it to updateOwner with the test that changes it, or to updateExempt with the reason it cannot be changed", m)
	}

	if pending == 0 && len(updatePending) > 0 {
		t.Error("updatePending is non-empty but satisfies nothing; delete it")
	}
	t.Logf("%d attribute declarations: %d changed in life, %d ForceNew, %d exempt, %d pending, %d uncovered",
		len(groups), covered, forced, exempt, pending, len(missing))
}
