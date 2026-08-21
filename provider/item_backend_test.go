package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// Completeness guard for the item / prototype / LLD backend-type check
// (item_backend.go).
//
// The wrapper signatures already make a *missing* check a compile error: no
// resource in the triad can be constructed without handing its itemTypeSet to
// itemGetReadWrapper or one of its five siblings. What a signature cannot
// catch is a set that is wrong -- zabbix_item_snmptrap declaring
// zabbix.SNMPAgent, say, which would let an SNMP poller be imported as a trap
// item and rewritten on the first edit, exactly the defect the check exists to
// stop.
//
// So this drives the real thing. Every resource the provider registers under
// zabbix_item_, zabbix_proto_item_ or zabbix_lld_ is read once per Zabbix item
// type against a stub server that answers item.get / itemprototype.get /
// discoveryrule.get with an object of that type, and the set of types it
// accepts is compared against what itemBackends claims. No Docker and no live
// server: this runs in `go test ./provider/`, so a backend added tomorrow with
// a copy-pasted type set fails the ordinary build.

// stubAPI is a *zabbix.API pointed at an httptest server that answers
// apiinfo.version and the three .get methods with one object of the type in
// *itemType. Nothing else is implemented; a read that reached anything else
// would fail loudly, which is the correct outcome for a test that only reads.
func stubAPI(t *testing.T, itemType *zabbix.ItemType) *zabbix.API {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			ID     int32  `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("stub server: undecodable request: %v", err)
			return
		}

		var result interface{}
		switch strings.ToLower(req.Method) {
		case "apiinfo.version":
			// 7.0 rather than 6.0: the version only reaches version gates on
			// the write path here, and picking the floor would leave the
			// newest gates untested by accident
			result = "7.0.29"
		case "item.get", "itemprototype.get":
			result = []map[string]interface{}{{
				"itemid":        "12345",
				"type":          fmt.Sprintf("%d", int(*itemType)),
				"key_":          "stub.key",
				"name":          "stub item",
				"value_type":    "3",
				"delay":         "1m",
				"description":   "",
				"units":         "",
				"preprocessing": []interface{}{},
				"tags":          []interface{}{},
			}}
		case "discoveryrule.get":
			result = []map[string]interface{}{{
				"itemid":      "12345",
				"type":        fmt.Sprintf("%d", int(*itemType)),
				"key_":        "stub.key",
				"name":        "stub rule",
				"delay":       "1h",
				"lifetime":    "30d",
				"description": "",
				"filter": map[string]interface{}{
					"conditions": []interface{}{},
					"evaltype":   "0",
					"formula":    "",
				},
				"preprocessing":   []interface{}{},
				"lld_macro_paths": []interface{}{},
			}}
		default:
			t.Errorf("stub server: unexpected method %q; this test only reads", req.Method)
			result = []interface{}{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0", "id": req.ID, "result": result,
		}); err != nil {
			t.Errorf("stub server: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	api, err := zabbix.NewAPI(zabbix.Config{Url: srv.URL})
	if err != nil {
		t.Fatalf("stub API: %v", err)
	}
	api.Auth = "stub-token"
	return api
}

// itemFamilyOf classifies a resource name. Order matters: zabbix_proto_item_
// has to be tested before zabbix_item_ would ever be, and it is not a prefix
// of it, so the switch is written longest-first to keep that obvious.
func itemFamilyOf(name string) (itemFamily, bool) {
	switch {
	case strings.HasPrefix(name, "zabbix_proto_item_"):
		return familyProtoItem, true
	case strings.HasPrefix(name, "zabbix_lld_"):
		return familyLLD, true
	case strings.HasPrefix(name, "zabbix_item_"):
		return familyItem, true
	}
	return 0, false
}

// TestItemBackendRegistrationComplete is the cheap half: itemBackends and the
// provider's ResourcesMap have to name the same resources, in both directions.
// A backend added to provider.go without a row here would have no label and no
// "import it as" hint; a row here with no resource would offer a hint naming
// something that does not exist.
func TestItemBackendRegistrationComplete(t *testing.T) {
	registered := map[itemFamily]map[string]bool{
		familyItem: {}, familyProtoItem: {}, familyLLD: {},
	}
	for name := range Provider().ResourcesMap {
		if f, ok := itemFamilyOf(name); ok {
			registered[f][name] = true
		}
	}

	total := 0
	for _, f := range []itemFamily{familyItem, familyProtoItem, familyLLD} {
		claimed := itemFamilyResources(f)
		for _, name := range claimed {
			if !registered[f][name] {
				t.Errorf("itemBackends claims %s but the provider does not register it", name)
			}
		}
		set := map[string]bool{}
		for _, name := range claimed {
			set[name] = true
		}
		names := make([]string, 0, len(registered[f]))
		for name := range registered[f] {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if !set[name] {
				t.Errorf("%s is registered but no itemBackends row claims it; it has no type label and no import hint", name)
			}
		}
		total += len(claimed)
	}

	if total == 0 {
		t.Fatal("no item-family resources found at all; itemFamilyOf has stopped working")
	}

	// every row has to carry a label, including the seven backends with no
	// resource: the label is what an unimportable type is named by
	for typ, b := range itemBackends {
		if b.label == "" {
			t.Errorf("itemBackends[%d] has no label", int(typ))
		}
		if b.suffix == "" && len(b.families) > 0 {
			t.Errorf("itemBackends[%d] names families but no suffix", int(typ))
		}
		if b.suffix != "" && len(b.families) == 0 {
			t.Errorf("itemBackends[%d] has a suffix but no families", int(typ))
		}
	}
}

// TestItemBackendTypes is the expensive half, and the one that catches a wrong
// set rather than a missing one. It reads every registered resource in the
// triad once per item type and asserts the accepted set is exactly what
// itemBackends claims.
func TestItemBackendTypes(t *testing.T) {
	var served zabbix.ItemType
	api := stubAPI(t, &served)

	types := make([]zabbix.ItemType, 0, len(itemBackends))
	for typ := range itemBackends {
		types = append(types, typ)
	}
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })

	names := make([]string, 0)
	for name := range Provider().ResourcesMap {
		if _, ok := itemFamilyOf(name); ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatal("no item-family resources found at all")
	}

	for _, name := range names {
		name := name
		family, _ := itemFamilyOf(name)
		res := Provider().ResourcesMap[name]

		t.Run(name, func(t *testing.T) {
			if res.Read == nil {
				t.Fatalf("%s has no Read function", name)
			}

			want := map[zabbix.ItemType]bool{}
			for _, typ := range types {
				if itemTypeResource(typ, family) == name {
					want[typ] = true
				}
			}
			if len(want) == 0 {
				t.Fatalf("itemBackends claims no item type for %s", name)
			}

			for _, typ := range types {
				served = typ
				d := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{})
				d.SetId("12345")

				err := res.Read(d, api)

				switch {
				case want[typ] && err != nil:
					t.Errorf("%s refused %s (type %d), which itemBackends says it represents: %v",
						name, itemTypeLabel(typ), int(typ), err)
				case !want[typ] && err == nil:
					t.Errorf("%s accepted %s (type %d); a wrong-typed object read into this "+
						"resource is rewritten to its own type on the first update",
						name, itemTypeLabel(typ), int(typ))
				case !want[typ]:
					// the message has to name what the object actually is,
					// or the user has no way to act on it
					if !strings.Contains(err.Error(), itemTypeLabel(typ)) {
						t.Errorf("%s rejected type %d without naming it: %v", name, int(typ), err)
					}
					if hint := itemTypeResource(typ, family); hint != "" &&
						!strings.Contains(err.Error(), hint) {
						t.Errorf("%s rejected type %d without pointing at %s: %v",
							name, int(typ), hint, err)
					}
				}
			}
		})
	}
}

// TestItemBackendTypeMessages pins the wording, because it is the whole
// deliverable of a rejection: a user reading it has to learn what the object
// is and which resource takes it.
func TestItemBackendTypeMessages(t *testing.T) {
	for _, c := range []struct {
		name     string
		actual   zabbix.ItemType
		accepted itemTypeSet
		family   itemFamily
		want     string
	}{
		{
			name:   "item, both sides have a resource",
			actual: zabbix.SNMPAgent, accepted: itemTypeSet{zabbix.ZabbixAgent, zabbix.ZabbixAgentActive},
			family: familyItem,
			want:   "item 12345 is a SNMP agent item, not a Zabbix agent or Zabbix agent (active) item; import it as zabbix_item_snmp",
		},
		{
			name:   "prototype",
			actual: zabbix.HTTPAgent, accepted: itemTypeSet{zabbix.SNMPAgent},
			family: familyProtoItem,
			want:   "item prototype 12345 is a HTTP agent item prototype, not a SNMP agent item prototype; import it as zabbix_proto_item_http",
		},
		{
			name:   "discovery rule",
			actual: zabbix.ZabbixTrapper, accepted: itemTypeSet{zabbix.SNMPAgent},
			family: familyLLD,
			want:   "discovery rule 12345 is a Zabbix trapper discovery rule, not a SNMP agent discovery rule; import it as zabbix_lld_trapper",
		},
		{
			name:   "a type this provider has no resource for",
			actual: zabbix.Browser, accepted: itemTypeSet{zabbix.SNMPAgent},
			family: familyItem,
			want:   "item 12345 is a browser item, not a SNMP agent item, which this provider has no resource for",
		},
		{
			name:   "a discovery rule type with no LLD resource",
			actual: zabbix.Calculated, accepted: itemTypeSet{zabbix.SNMPAgent},
			family: familyLLD,
			want:   "discovery rule 12345 is a calculated discovery rule, not a SNMP agent discovery rule, which this provider has no resource for",
		},
		{
			name:   "a type newer than this provider",
			actual: zabbix.ItemType(99), accepted: itemTypeSet{zabbix.SNMPAgent},
			family: familyItem,
			want:   "item 12345 is a type 99 item, not a SNMP agent item, which this provider has no resource for",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := checkItemBackendType("12345", c.actual, c.accepted, c.family)
			if err == nil {
				t.Fatal("expected a rejection")
			}
			if err.Error() != c.want {
				t.Errorf("message\n got: %s\nwant: %s", err.Error(), c.want)
			}
		})
	}

	// and the accepting side stays silent
	if err := checkItemBackendType("1", zabbix.ZabbixAgentActive,
		itemTypeSet{zabbix.ZabbixAgent, zabbix.ZabbixAgentActive}, familyItem); err != nil {
		t.Errorf("an accepted type was rejected: %v", err)
	}
}
