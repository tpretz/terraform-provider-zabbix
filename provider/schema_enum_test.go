package provider

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
	"zabbix_item_agent":       {"valuetype"},
	"zabbix_proto_item_agent": {"valuetype"},
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
		{"HOST_IFACE_TYPES", HOST_IFACE_TYPES, HOST_IFACE_TYPES_REV, nil},
		{"HTTP_METHODS", HTTP_METHODS, HTTP_METHODS_REV, HTTP_METHODS_ARR},
		{"HTTP_RETRIEVEMODE", HTTP_RETRIEVEMODE, HTTP_RETRIEVEMODE_REV, HTTP_RETRIEVEMODE_ARR},
		{"HTTP_POSTTYPE", HTTP_POSTTYPE, HTTP_POSTTYPE_REV, HTTP_POSTTYPE_ARR},
		{"HTTP_AUTHTYPE", HTTP_AUTHTYPE, HTTP_AUTHTYPE_REV, HTTP_AUTHTYPE_ARR},
		{"PROXY_MODE_LOOKUP", PROXY_MODE_LOOKUP, PROXY_MODE_LOOKUP_REV, PROXY_MODE_LOOKUP_ARR},
		{"PROXY_TLS_LOOKUP", PROXY_TLS_LOOKUP, PROXY_TLS_LOOKUP_REV, PROXY_TLS_LOOKUP_ARR},
		{"TRIGGER_PRIORITY", TRIGGER_PRIORITY, TRIGGER_PRIORITY_REV, TRIGGER_PRIORITY_ARR},
		{"TRIGGER_CORRELATION", TRIGGER_CORRELATION, TRIGGER_CORRELATION_REV, TRIGGER_CORRELATION_ARR},
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

// enumDescriptionExempt lists the enums whose description deliberately does
// not enumerate its values.
//
// zabbix_host.interface.type is not deliberate: it is the one enum written as
// an inline []string instead of through the _LOOKUP/_ARR idiom, so there is
// no generated list for its description to interpolate and it says only
// "Interface type". Left as found and reported rather than fixed -- schema
// descriptions belong to PLAN.md Phase 5 -- with
// TestSchemaHostInterfaceTypeMatchesLookup standing in for the drift the
// idiom would have prevented.
var enumDescriptionExempt = map[string]bool{
	"interface.type": true,
}

// TestEnumDescriptionsListValues keeps the generated documentation honest.
// These descriptions are what tfplugindocs will publish, and one built from a
// stale list is a lie the compiler cannot catch: the reader is told to use a
// value the provider rejects, or not told about one it accepts.
func TestEnumDescriptionsListValues(t *testing.T) {
	for name, res := range providerSchemas() {
		walkSchema(res.Schema, "", func(path string, s *schema.Schema) {
			allowed := enumValues(s, path)
			if allowed == nil || enumDescriptionExempt[path] {
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
