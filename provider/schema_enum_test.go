package provider

import (
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// Unit-level half of E2 (PLAN.md Phase 8). The acceptance half, in
// acc_negative_test.go, proves the diagnostic reaches the user through a real
// terraform run; these tests prove the same validators are complete and
// consistent, for every enum at once and without a server.
//
// The enum idiom (CLAUDE.md § "Shared schema helpers & the lookup-table
// idiom") is a forward map plus a generated reverse map and value list. Its
// failure mode is silent in both directions: a value in the _ARR but not the
// forward map validates and is then sent to Zabbix as the zero code, and a
// code returned by Zabbix that is missing from the _REV reads back as the
// empty string and produces a diff that can never be applied away. Neither
// shows up in a happy-path test, because the happy path uses the values that
// are in both.

// enumSentinel is a value no enum could legitimately contain.
const enumSentinel = "zzz-not-a-valid-enum-value"

var (
	enumErrRe = regexp.MustCompile(`to be one of \[(.*)\], got `)
	enumValRe = regexp.MustCompile(`"([^"]*)"`)
)

// enumValues detects a validation.StringInSlice ValidateFunc by behaviour --
// feeding it a value nothing accepts and reading the permitted set back out
// of the resulting message -- and returns that set, or nil if s is validated
// some other way (or not at all).
//
// Behavioural detection rather than reflection is deliberate: it needs no
// register of which attributes are enums, so an enum added tomorrow is picked
// up with no test change, which is the only way a completeness check stays
// true.
func enumValues(s *schema.Schema, key string) []string {
	if s.ValidateFunc == nil {
		return nil
	}
	_, errs := s.ValidateFunc(enumSentinel, key)
	for _, err := range errs {
		m := enumErrRe.FindStringSubmatch(err.Error())
		if m == nil {
			continue
		}
		var out []string
		for _, v := range enumValRe.FindAllStringSubmatch(m[1], -1) {
			out = append(out, v[1])
		}
		return out
	}
	return nil
}

// walkSchema visits every attribute of a resource, descending into nested
// blocks, and reports each as a dotted path.
func walkSchema(m map[string]*schema.Schema, prefix string, fn func(path string, s *schema.Schema)) {
	for k, s := range m {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		fn(path, s)
		if r, ok := s.Elem.(*schema.Resource); ok {
			walkSchema(r.Schema, path, fn)
		}
	}
}

// providerSchemas returns every resource and data source keyed by its
// Terraform type name.
func providerSchemas() map[string]*schema.Resource {
	p := Provider()
	out := make(map[string]*schema.Resource, len(p.ResourcesMap)+len(p.DataSourcesMap))
	for k, v := range p.ResourcesMap {
		out[k] = v
	}
	for k, v := range p.DataSourcesMap {
		// a data source and a resource can share a name; the resource wins,
		// and the data source is reached under a distinct key
		if _, clash := out[k]; clash {
			out["data."+k] = v
			continue
		}
		out[k] = v
	}
	return out
}

// TestSchemaEnumsAcceptEveryDeclaredValue walks every enum in every resource
// and data source and requires that each value it advertises is a value it
// accepts. An _ARR that has drifted from its forward map advertises a value
// the mod func cannot translate; this is the direction no fixture can catch,
// because a fixture only ever uses the values it already knows work.
func TestSchemaEnumsAcceptEveryDeclaredValue(t *testing.T) {
	found := 0
	for name, res := range providerSchemas() {
		walkSchema(res.Schema, "", func(path string, s *schema.Schema) {
			allowed := enumValues(s, path)
			if allowed == nil {
				return
			}
			found++

			if len(allowed) == 0 {
				t.Errorf("%s.%s: enum permits no values at all", name, path)
				return
			}
			for _, v := range allowed {
				if _, errs := s.ValidateFunc(v, path); len(errs) > 0 {
					t.Errorf("%s.%s: rejects its own declared value %q: %v", name, path, v, errs)
				}
			}
			if _, errs := s.ValidateFunc(enumSentinel, path); len(errs) == 0 {
				t.Errorf("%s.%s: accepted %q", name, path, enumSentinel)
			}
		})
	}
	if found == 0 {
		t.Fatal("no enum-validated attributes found at all; the detection in enumValues has stopped working")
	}
	t.Logf("checked %d enum-validated attributes", found)
}

// requiredEnumSites is the checklist half of E2: every _LOOKUP_ARR enum, named
// where it is reachable from configuration. TestSchemaEnumsAcceptEveryDeclaredValue
// covers whatever validators happen to exist; this covers the ones that are
// supposed to exist, so silently dropping a ValidateFunc is a failure rather
// than a quietly smaller run.
//
// One resource per family is enough for the shared schema fragments -- all ten
// zabbix_item_* resources merge the same itemCommonSchema -- but each distinct
// fragment is represented.
var requiredEnumSites = map[string][]string{
	"zabbix_host": {
		"inventory_mode",
		"interface.type",
		"interface.snmp_version",
		"interface.snmp3_authprotocol",
		"interface.snmp3_privprotocol",
		"interface.snmp3_securitylevel",
		"ipmi_authtype",
		"ipmi_privilege",
		"tls_connect",
		"tls_accept",
	},
	"zabbix_proxy": {
		"operating_mode",
		"tls_connect",
		"tls_accept",
	},
	"zabbix_graph": {
		"type",
		"ymin_type",
		"ymax_type",
		"item.function",
		"item.drawtype",
		"item.type",
		"item.yaxis_side",
	},
	"zabbix_proto_graph": {
		"type",
		"ymin_type",
		"ymax_type",
		"item.function",
		"item.drawtype",
		"item.type",
		"item.yaxis_side",
	},
	"zabbix_trigger": {
		"priority",
		"correlation_mode",
	},
	"zabbix_proto_trigger": {
		"priority",
		"correlation_mode",
	},
	"zabbix_item_agent":       {"valuetype", "preprocessor.type"},
	"zabbix_proto_item_agent": {"valuetype", "preprocessor.type"},
	"zabbix_item_http": {
		"valuetype",
		"request_method",
		"post_type",
		"retrieve_mode",
		"auth_type",
	},
	"zabbix_lld_agent": {
		"evaltype",
		"condition.operator",
		// the discovery-rule preprocessing list, which is not the item one
		"preprocessor.type",
	},
	// the two discovery-rule types Zabbix requires delay == 0 for: the schema
	// pins the value with a one-element StringInSlice rather than leaving the
	// server to reject it
	"zabbix_lld_trapper":   {"delay", "evaltype", "condition.operator"},
	"zabbix_lld_dependent": {"delay", "evaltype", "condition.operator"},
}

func TestSchemaRequiredEnumSites(t *testing.T) {
	all := providerSchemas()

	for name, paths := range requiredEnumSites {
		res, ok := all[name]
		if !ok {
			t.Errorf("%s: not registered in the provider", name)
			continue
		}

		got := map[string][]string{}
		walkSchema(res.Schema, "", func(path string, s *schema.Schema) {
			if allowed := enumValues(s, path); allowed != nil {
				got[path] = allowed
			}
		})

		for _, path := range paths {
			if _, ok := got[path]; !ok {
				t.Errorf("%s.%s: expected an enum validator, found none", name, path)
			}
		}
	}
}

// TestSchemaHostInterfaceTypeMatchesLookup pins the one enum written as a
// literal slice rather than through the _LOOKUP/_ARR idiom. hostInterface
// "type" validates against an inline []string while the create path
// translates through HOST_IFACE_TYPES, so the two can drift apart with
// nothing to notice: a type accepted by the schema but missing from the map
// would be sent as interface type 0.
func TestSchemaHostInterfaceTypeMatchesLookup(t *testing.T) {
	res := Provider().ResourcesMap["zabbix_host"]
	iface, ok := res.Schema["interface"].Elem.(*schema.Resource)
	if !ok {
		t.Fatal("zabbix_host.interface has no nested resource")
	}

	allowed := enumValues(iface.Schema["type"], "type")
	if allowed == nil {
		t.Fatal("zabbix_host.interface.type has no enum validator")
	}

	var keys []string
	for k := range HOST_IFACE_TYPES {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sort.Strings(allowed)

	if !reflect.DeepEqual(keys, allowed) {
		t.Errorf("zabbix_host.interface.type validates %v but HOST_IFACE_TYPES has %v", allowed, keys)
	}
}

// lookupTable is one instance of the _LOOKUP / _LOOKUP_REV / _LOOKUP_ARR
// idiom. fwd and rev are held as interface{} and read reflectively because
// every one of them has a different value type.
type lookupTable struct {
	name string
	fwd  interface{}
	rev  interface{}
	arr  []string
}

func lookupTables() []lookupTable {
	return []lookupTable{
		{"ITEM_VALUE_TYPES", ITEM_VALUE_TYPES, ITEM_VALUE_TYPES_REV, ITEM_VALUE_TYPES_ARR},
		{"LLD_EVALTYPE", LLD_EVALTYPE, LLD_EVALTYPE_REV, LLD_EVALTYPE_ARR},
		{"LLD_OPERATOR", LLD_OPERATOR, LLD_OPERATOR_REV, LLD_OPERATOR_ARR},
		{"GRAPH_TYPE_LOOKUP", GRAPH_TYPE_LOOKUP, GRAPH_TYPE_LOOKUP_REV, GRAPH_TYPE_LOOKUP_ARR},
		{"GRAPH_AXIS_LOOKUP", GRAPH_AXIS_LOOKUP, GRAPH_AXIS_LOOKUP_REV, GRAPH_AXIS_LOOKUP_ARR},
		{"GRAPH_FUNC_LOOKUP", GRAPH_FUNC_LOOKUP, GRAPH_FUNC_LOOKUP_REV, GRAPH_FUNC_LOOKUP_ARR},
		{"GRAPH_DRAW_LOOKUP", GRAPH_DRAW_LOOKUP, GRAPH_DRAW_LOOKUP_REV, GRAPH_DRAW_LOOKUP_ARR},
		{"GRAPH_ITYPE_LOOKUP", GRAPH_ITYPE_LOOKUP, GRAPH_ITYPE_LOOKUP_REV, GRAPH_ITYPE_LOOKUP_ARR},
		{"GRAPH_SIDE_LOOKUP", GRAPH_SIDE_LOOKUP, GRAPH_SIDE_LOOKUP_REV, GRAPH_SIDE_LOOKUP_ARR},
		{"HINV_LOOKUP", HINV_LOOKUP, HINV_LOOKUP_REV, HINV_LOOKUP_ARR},
		{"HSNMP_AUTHPROTO", HSNMP_AUTHPROTO, HSNMP_AUTHPROTO_REV, HSNMP_AUTHPROTO_ARR},
		{"HSNMP_PRIVPROTO", HSNMP_PRIVPROTO, HSNMP_PRIVPROTO_REV, HSNMP_PRIVPROTO_ARR},
		{"HSNMP_SECLEVEL", HSNMP_SECLEVEL, HSNMP_SECLEVEL_REV, HSNMP_SECLEVEL_ARR},
		{"HOST_TLS_LOOKUP", HOST_TLS_LOOKUP, HOST_TLS_LOOKUP_REV, HOST_TLS_LOOKUP_ARR},
		{"HOST_IPMI_AUTHTYPE", HOST_IPMI_AUTHTYPE, HOST_IPMI_AUTHTYPE_REV, HOST_IPMI_AUTHTYPE_ARR},
		{"HOST_IPMI_PRIVILEGE", HOST_IPMI_PRIVILEGE, HOST_IPMI_PRIVILEGE_REV, HOST_IPMI_PRIVILEGE_ARR},
		{"HOST_IFACE_TYPES", HOST_IFACE_TYPES, HOST_IFACE_TYPES_REV, HOST_IFACE_TYPES_ARR},
		{"HTTP_METHODS", HTTP_METHODS, HTTP_METHODS_REV, HTTP_METHODS_ARR},
		{"HTTP_RETRIEVEMODE", HTTP_RETRIEVEMODE, HTTP_RETRIEVEMODE_REV, HTTP_RETRIEVEMODE_ARR},
		{"HTTP_POSTTYPE", HTTP_POSTTYPE, HTTP_POSTTYPE_REV, HTTP_POSTTYPE_ARR},
		{"HTTP_AUTHTYPE", HTTP_AUTHTYPE, HTTP_AUTHTYPE_REV, HTTP_AUTHTYPE_ARR},
		{"PROXY_MODE_LOOKUP", PROXY_MODE_LOOKUP, PROXY_MODE_LOOKUP_REV, PROXY_MODE_LOOKUP_ARR},
		{"PROXY_TLS_LOOKUP", PROXY_TLS_LOOKUP, PROXY_TLS_LOOKUP_REV, PROXY_TLS_LOOKUP_ARR},
		{"TRIGGER_PRIORITY", TRIGGER_PRIORITY, TRIGGER_PRIORITY_REV, TRIGGER_PRIORITY_ARR},
		{"TRIGGER_CORRELATION", TRIGGER_CORRELATION, TRIGGER_CORRELATION_REV, TRIGGER_CORRELATION_ARR},
		{"PREPROC_LOOKUP", PREPROC_LOOKUP, PREPROC_LOOKUP_REV, PREPROC_LOOKUP_ARR},
		{"LLD_PREPROC_LOOKUP", LLD_PREPROC_LOOKUP, LLD_PREPROC_LOOKUP_REV, LLD_PREPROC_LOOKUP_ARR},
	}
}

// TestEnumLookupTablesInSync checks the three halves of each lookup agree:
// the value list offered to validation and documentation, the forward map the
// write path uses, and the reverse map the read path uses.
func TestEnumLookupTablesInSync(t *testing.T) {
	for _, lt := range lookupTables() {
		t.Run(lt.name, func(t *testing.T) {
			fwd := reflect.ValueOf(lt.fwd)
			rev := reflect.ValueOf(lt.rev)

			if fwd.Len() == 0 {
				t.Fatalf("%s is empty", lt.name)
			}

			var fwdKeys []string
			for _, k := range fwd.MapKeys() {
				fwdKeys = append(fwdKeys, k.String())
			}

			if lt.arr != nil {
				arr := append([]string(nil), lt.arr...)
				keys := append([]string(nil), fwdKeys...)
				sort.Strings(arr)
				sort.Strings(keys)
				if !reflect.DeepEqual(arr, keys) {
					t.Errorf("%s_ARR is %v but %s has keys %v", lt.name, arr, lt.name, keys)
				}
			}

			// every code the write path can produce must be one the read
			// path can turn back into a name, or the attribute reads as ""
			// and the resource sits in a diff nothing can clear
			for _, k := range fwd.MapKeys() {
				code := fwd.MapIndex(k)
				got := rev.MapIndex(code)
				if !got.IsValid() {
					t.Errorf("%s_REV has no entry for %v (%q)", lt.name, code.Interface(), k.String())
				}
			}

			// and nothing the read path knows about should be absent from
			// the write path, which would mean a value that can be read but
			// never written
			codes := map[interface{}]bool{}
			for _, k := range fwd.MapKeys() {
				codes[fwd.MapIndex(k).Interface()] = true
			}
			for _, c := range rev.MapKeys() {
				if !codes[c.Interface()] {
					t.Errorf("%s_REV has %v -> %q which %s cannot produce",
						lt.name, c.Interface(), rev.MapIndex(c).String(), lt.name)
				}
			}
		})
	}
}

// TestEnumDescriptionsListValues keeps the generated documentation honest.
// These descriptions are what tfplugindocs publishes, and one built from a
// stale list is a lie the compiler cannot catch: the reader is told to use a
// value the provider rejects, or not told about one it accepts.
//
// There are no exemptions. zabbix_host.interface.type used to be one -- the
// single enum written as an inline []string rather than through the
// _LOOKUP/_ARR idiom, so it had no generated list to interpolate and said only
// "Interface type" -- and was fixed in Phase 5 by giving it a
// HOST_IFACE_TYPES_ARR to build the description from. The validator still
// reads an inline slice, deliberately, so that
// TestSchemaHostInterfaceTypeMatchesLookup keeps catching drift between the
// two rather than being trivially satisfied.
func TestEnumDescriptionsListValues(t *testing.T) {
	for name, res := range providerSchemas() {
		walkSchema(res.Schema, "", func(path string, s *schema.Schema) {
			allowed := enumValues(s, path)
			if allowed == nil {
				return
			}
			for _, v := range allowed {
				if !strings.Contains(s.Description, v) {
					t.Errorf("%s.%s: description omits permitted value %q: %q", name, path, v, s.Description)
				}
			}
		})
	}
}

// TestSchemaDescriptionsPresent requires every attribute of every resource and
// data source to carry a Description.
//
// This is not a style rule. The Description is the *only* source the
// documentation has: `tfplugindocs generate` renders the argument reference
// straight from the schema, so an attribute without one is published as a bare
// name with nothing beside it, and the same hole appears in
// `terraform providers schema -json` and in editor tooling. There is no
// separate place to write it down instead, which is why this is enforced
// rather than reviewed.
func TestSchemaDescriptionsPresent(t *testing.T) {
	total := 0
	for name, res := range providerSchemas() {
		walkSchema(res.Schema, "", func(path string, s *schema.Schema) {
			total++
			if s.Description == "" {
				t.Errorf("%s.%s: no Description; it would be published as a bare attribute name", name, path)
			}
		})
	}
	if total == 0 {
		t.Fatal("no attributes walked at all; providerSchemas or walkSchema has stopped working")
	}
	t.Logf("checked %d attributes", total)
}

// TestEnumValueListsAreSorted pins the ordering of the generated _ARR value
// lists.
//
// They are built by ranging over a map, and Go randomises map iteration
// order, so without an explicit sort the same build produces a different
// value list every run -- which means a different validation message and,
// worse, a different generated docs page. `make docs-check` would then fail
// at random. Each _ARR is sorted at init; this checks none loses that.
//
// Two lists are excluded, both because they are already deterministic by
// construction and their order says something sorting would destroy:
// ITEM_VALUE_TYPES_ARR is an explicit literal, not built from its map, and
// TRIGGER_PRIORITY_ARR is ordered by Zabbix severity code rather than
// alphabetically -- the list is a scale, and printing it as
// "average, disaster, high, ..." would hide that.
var unsortedByDesign = map[string]bool{
	"ITEM_VALUE_TYPES": true,
	"TRIGGER_PRIORITY": true,
}

func TestEnumValueListsAreSorted(t *testing.T) {
	for _, lt := range lookupTables() {
		if lt.arr == nil || unsortedByDesign[lt.name] {
			continue
		}
		if !sort.StringsAreSorted(lt.arr) {
			t.Errorf("%s_ARR is not sorted: %v -- map iteration order is random, so it must be sorted at init", lt.name, lt.arr)
		}
	}

	// the severity scale: deterministic, but ordered by code
	want := []string{"not_classified", "info", "warn", "average", "high", "disaster"}
	if !reflect.DeepEqual(TRIGGER_PRIORITY_ARR, want) {
		t.Errorf("TRIGGER_PRIORITY_ARR is %v, want severity order %v", TRIGGER_PRIORITY_ARR, want)
	}
}

// ---------------------------------------------------------------------------
// preprocessor.type
// ---------------------------------------------------------------------------
//
// The preprocessing step type was the last enum in the provider written as raw
// Zabbix numbering -- a TypeString validated by "^[0-9]+$", so `type = "12"`
// was the interface and the user had to know 12 means JSONPath. It is now a
// named enum like every other, with three properties nothing else in the
// provider has and which therefore need their own checks: it accepts the old
// numeric form as well, it normalises that form to the name before state, and
// the item list and the discovery-rule list are different lists.

// TestLLDPreprocLookupSubset requires the discovery-rule table to agree with
// the item table on what every name means. The two are separate maps -- an LLD
// rule accepts a strict subset of the item types -- and the failure mode of
// letting them drift is a name that silently means a different step depending
// on which resource it was written in.
func TestLLDPreprocLookupSubset(t *testing.T) {
	for name, code := range LLD_PREPROC_LOOKUP {
		itemCode, ok := PREPROC_LOOKUP[name]
		if !ok {
			t.Errorf("LLD_PREPROC_LOOKUP has %q, which PREPROC_LOOKUP does not", name)
			continue
		}
		if itemCode != code {
			t.Errorf("%q is code %s for an LLD rule but %s for an item", name, code, itemCode)
		}
	}
	if len(LLD_PREPROC_LOOKUP) >= len(PREPROC_LOOKUP) {
		t.Errorf("LLD_PREPROC_LOOKUP has %d entries and PREPROC_LOOKUP %d; the discovery-rule list is meant to be the smaller one",
			len(LLD_PREPROC_LOOKUP), len(PREPROC_LOOKUP))
	}

	// the codes established against the live servers, spelled out so that a
	// well-meaning "surely an LLD rule can do arithmetic too" edit fails here
	// rather than against a server the author did not happen to run
	want := []string{"5", "11", "12", "14", "15", "16", "17", "20", "21", "23", "24", "25", "27", "28", "29", "30"}
	var got []string
	for code := range LLD_PREPROC_LOOKUP_REV {
		got = append(got, code)
	}
	sort.Slice(got, func(i, j int) bool { return atoiOrZero(got[i]) < atoiOrZero(got[j]) })
	if !reflect.DeepEqual(got, want) {
		t.Errorf("discovery rules accept codes %v, want %v", got, want)
	}
}

func atoiOrZero(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// TestPreprocTypeGatesAreKnownTypes stops a gate being written for a name that
// does not exist. A typo there is invisible: the gate simply never fires and
// the type is offered on a server that does not have it.
func TestPreprocTypeGatesAreKnownTypes(t *testing.T) {
	for name := range PREPROC_MIN_VERSION {
		if _, ok := PREPROC_LOOKUP[name]; !ok {
			t.Errorf("PREPROC_MIN_VERSION gates %q, which is not a preprocessing type", name)
		}
	}
	for name := range LLD_PREPROC_MIN_VERSION {
		if _, ok := LLD_PREPROC_LOOKUP[name]; !ok {
			t.Errorf("LLD_PREPROC_MIN_VERSION gates %q, which is not a discovery-rule preprocessing type", name)
		}
	}

	// matches_regex is the one gate that differs between the two, and it is
	// the reason the LLD list has its own gate map rather than sharing the
	// item one. Zabbix 6.0 takes "14" on an item and rejects it on a discovery
	// rule with `unexpected value "14"`; 7.0 takes it on both. Verified on
	// live 6.0.48 and 7.0.29.
	if _, ok := PREPROC_MIN_VERSION["matches_regex"]; ok {
		t.Error("matches_regex is gated for items; it has existed since 6.0 there")
	}
	if g := LLD_PREPROC_MIN_VERSION["matches_regex"]; g.version != zabbix.V70 {
		t.Errorf("matches_regex on a discovery rule is gated at %d, want %d (7.0)", g.version, zabbix.V70)
	}
}

// TestPreprocTypeValidatorAcceptsNumeric covers the compatibility form: every
// name is accepted, so is the code it stands for, the code produces a
// deprecation warning rather than an error, and anything else is rejected with
// the message shape the rest of this file depends on.
func TestPreprocTypeValidatorAcceptsNumeric(t *testing.T) {
	for _, tc := range []struct {
		what     string
		lookup   map[string]string
		rev      map[string]string
		arr      []string
		rejected string
	}{
		{"item", PREPROC_LOOKUP, PREPROC_LOOKUP_REV, PREPROC_LOOKUP_ARR, "999"},
		// an item type that is not a discovery-rule type: the LLD validator
		// must turn it down, by name and by code alike
		{"lld", LLD_PREPROC_LOOKUP, LLD_PREPROC_LOOKUP_REV, LLD_PREPROC_LOOKUP_ARR, "multiplier"},
	} {
		t.Run(tc.what, func(t *testing.T) {
			v := preprocessorTypeValidator(tc.lookup, tc.rev, tc.arr)

			for name, code := range tc.lookup {
				if warns, errs := v(name, "type"); len(errs) > 0 || len(warns) > 0 {
					t.Errorf("name %q: warns %v errs %v, want silence", name, warns, errs)
				}
				warns, errs := v(code, "type")
				if len(errs) > 0 {
					t.Errorf("code %q (%s): rejected: %v", code, name, errs)
				}
				if len(warns) != 1 || !strings.Contains(warns[0], "deprecated") {
					t.Errorf("code %q (%s): warns %v, want one deprecation warning", code, name, warns)
				}
			}

			_, errs := v(tc.rejected, "type")
			if len(errs) != 1 {
				t.Fatalf("%q: errs %v, want one", tc.rejected, errs)
			}
			if !strings.Contains(errs[0].Error(), "to be one of") {
				t.Errorf("%q: %q does not carry the permitted set; schema_enum_test and "+
					"acc_negative_test both read the enum back out of that phrase",
					tc.rejected, errs[0])
			}
		})
	}

	// "0" is a code Zabbix does not use, and the empty string is what a
	// missing value decodes to; neither may slip through as a valid step
	for _, bad := range []string{"", "0", "31", "-1", "12.0", "JSONPath"} {
		if _, errs := preprocessorTypeValidator(PREPROC_LOOKUP, PREPROC_LOOKUP_REV, PREPROC_LOOKUP_ARR)(bad, "type"); len(errs) == 0 {
			t.Errorf("%q was accepted", bad)
		}
	}
}

// TestPreprocTypeStateFuncNormalises is the other half of the compatibility
// story. The validator letting "12" through is only safe because the value is
// rewritten to "jsonpath" before it is compared with state -- otherwise a
// config that kept the numeric form would plan a diff on every run, for ever.
func TestPreprocTypeStateFuncNormalises(t *testing.T) {
	f := preprocessorTypeStateFunc(PREPROC_LOOKUP_REV)

	for name, code := range PREPROC_LOOKUP {
		if got := f(code); got != name {
			t.Errorf("StateFunc(%q) = %q, want %q", code, got, name)
		}
		if got := f(name); got != name {
			t.Errorf("StateFunc(%q) = %q, want it left alone", name, got)
		}
	}

	// a code from a Zabbix newer than this provider is passed through rather
	// than blanked, so the resulting diff names something a human can look up
	if got := f("31"); got != "31" {
		t.Errorf("StateFunc(%q) = %q, want the unknown code passed through", "31", got)
	}
}

// TestResolvePreprocessorType covers the create/update path: name or code in,
// Zabbix's code out, and a refusal with the version in it for a type the
// server does not have. A ValidateFunc cannot do this -- it runs before the
// provider has spoken to any server -- so this is the only place the gate
// exists.
func TestResolvePreprocessorType(t *testing.T) {
	const v60, v64, v70 = 60048, 60400, 70029

	for _, tc := range []struct {
		in      string
		version int
		want    string
		errWant string
	}{
		{"jsonpath", v60, "12", ""},
		{"12", v60, "12", ""},
		{"matches_regex", v60, "14", ""}, // item side: 6.0 has it
		{"snmp_walk_value", v70, "28", ""},
		{"snmp_walk_value", v64, "28", ""},
		{"snmp_walk_value", v60, "", "requires Zabbix 6.4 or later"},
		{"29", v60, "", "requires Zabbix 6.4 or later"},
		{"snmp_get_value", v70, "30", ""},
		{"snmp_get_value", v64, "", "requires Zabbix 7.0 or later"},
		{"nonsense", v70, "", "unknown preprocessing step type"},
	} {
		got, err := resolvePreprocessorType(tc.in, PREPROC_LOOKUP, PREPROC_LOOKUP_REV, PREPROC_MIN_VERSION, tc.version)
		switch {
		case tc.errWant == "" && err != nil:
			t.Errorf("item %q at %d: %s", tc.in, tc.version, err)
		case tc.errWant == "" && got != tc.want:
			t.Errorf("item %q at %d = %q, want %q", tc.in, tc.version, got, tc.want)
		case tc.errWant != "" && err == nil:
			t.Errorf("item %q at %d = %q, want error %q", tc.in, tc.version, got, tc.errWant)
		case tc.errWant != "" && !strings.Contains(err.Error(), tc.errWant):
			t.Errorf("item %q at %d: %q does not contain %q", tc.in, tc.version, err, tc.errWant)
		}
	}

	// the discovery-rule difference, which is the whole reason the two tables
	// carry separate gates
	if _, err := resolvePreprocessorType("matches_regex", LLD_PREPROC_LOOKUP, LLD_PREPROC_LOOKUP_REV, LLD_PREPROC_MIN_VERSION, v60); err == nil {
		t.Error("matches_regex on a 6.0 discovery rule was allowed; 6.0 rejects code 14 there")
	} else if !strings.Contains(err.Error(), "7.0") {
		t.Errorf("matches_regex on 6.0: %q does not name the version that has it", err)
	}
	if _, err := resolvePreprocessorType("matches_regex", LLD_PREPROC_LOOKUP, LLD_PREPROC_LOOKUP_REV, LLD_PREPROC_MIN_VERSION, v70); err != nil {
		t.Errorf("matches_regex on a 7.0 discovery rule: %s", err)
	}

	// the message has to say what the server is, not what its encoded version
	// integer is -- 60048 is not a version anybody recognises
	_, err := resolvePreprocessorType("snmp_get_value", PREPROC_LOOKUP, PREPROC_LOOKUP_REV, PREPROC_MIN_VERSION, v60)
	if err == nil || !strings.Contains(err.Error(), "6.0.48") {
		t.Errorf("gate message %v does not report the server version readably", err)
	}
}

func TestZabbixVersionString(t *testing.T) {
	for in, want := range map[int]string{
		60048: "6.0.48",
		70029: "7.0.29",
		70413: "7.4.13",
		80000: "8.0.0",
	} {
		if got := zabbixVersionString(in); got != want {
			t.Errorf("zabbixVersionString(%d) = %q, want %q", in, got, want)
		}
	}
}
