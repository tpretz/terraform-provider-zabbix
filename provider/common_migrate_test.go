package provider

import (
	"context"
	"testing"
)

// TestTypeSetStateUpgraders checks that every resource whose collection became
// a TypeSet in v2.0.0 declares the upgrade, and declares it with the *old*
// shape. Getting the v0 type wrong is silent: it only shows up as a mangled
// state file for someone upgrading from provider v0.x, which no acceptance
// test here can reach.
func TestTypeSetStateUpgraders(t *testing.T) {
	p := Provider()

	cases := map[string]string{
		"zabbix_graph":       "item",
		"zabbix_proto_graph": "item",
		"zabbix_host":        "interface",
	}
	for name, r := range p.ResourcesMap {
		if _, ok := r.Schema["condition"]; ok && len(name) > 11 && name[:11] == "zabbix_lld_" {
			cases[name] = "condition"
		}
	}
	if len(cases) < 11 {
		t.Fatalf("expected the three graph/host resources plus eight zabbix_lld_* ones, got %d: %v", len(cases), cases)
	}

	for name, attr := range cases {
		t.Run(name, func(t *testing.T) {
			r, ok := p.ResourcesMap[name]
			if !ok {
				t.Fatalf("%s is not registered", name)
			}

			if r.SchemaVersion != 1 {
				t.Errorf("SchemaVersion = %d, want 1", r.SchemaVersion)
			}
			if len(r.StateUpgraders) != 1 || r.StateUpgraders[0].Version != 0 {
				t.Fatalf("want exactly one v0 state upgrader, got %+v", r.StateUpgraders)
			}

			// current schema: a set
			if got := r.CoreConfigSchema().ImpliedType().AttributeType(attr); !got.IsSetType() {
				t.Errorf("current %q is %s, want a set", attr, got.FriendlyName())
			}

			// prior schema, as declared to the upgrader: a list
			prior := r.StateUpgraders[0].Type
			if !prior.IsObjectType() || !prior.HasAttribute(attr) {
				t.Fatalf("state upgrader type has no %q attribute: %s", attr, prior.FriendlyName())
			}
			if got := prior.AttributeType(attr); !got.IsListType() {
				t.Errorf("v0 %q is %s, want a list", attr, got.FriendlyName())
			}

			// and the upgrade itself hands the elements straight back: a list
			// and a set are the same array in JSON state
			in := map[string]interface{}{
				attr: []interface{}{map[string]interface{}{"id": "1"}},
			}
			out, err := r.StateUpgraders[0].Upgrade(context.Background(), in, nil)
			if err != nil {
				t.Fatalf("upgrade: %s", err)
			}
			list, ok := out[attr].([]interface{})
			if !ok || len(list) != 1 {
				t.Errorf("upgrade dropped %q: %#v", attr, out[attr])
			}
		})
	}
}

// TestHostInterfaceHashIgnoresComputedID pins the two halves of the set-hash
// contract that are easy to break and impossible to notice: the server-assigned
// id must not affect the hash (or every plan shows a replacement), and an
// omitted port must hash the same as the type's default port written out.
func TestHostInterfaceHashIgnoresComputedID(t *testing.T) {
	base := map[string]interface{}{
		"type": "agent", "ip": "127.0.0.1", "dns": "", "main": true, "port": 0,
	}
	withID := map[string]interface{}{
		"type": "agent", "ip": "127.0.0.1", "dns": "", "main": true, "port": 10050,
		"id": "42",
	}

	if hostInterfaceHash(base) != hostInterfaceHash(withID) {
		t.Error("an interface read from config and the same one read from state hash differently")
	}

	changed := map[string]interface{}{
		"type": "agent", "ip": "127.0.0.1", "dns": "", "main": true, "port": 10051,
	}
	if hostInterfaceHash(base) == hostInterfaceHash(changed) {
		t.Error("changing the port did not change the hash, so the edit would be silently dropped")
	}
}

// TestGraphItemHashCoversEveryField is the same contract for graph items: a
// change to any attribute a user can write has to move the hash, because
// diffSet never looks past it.
func TestGraphItemHashCoversEveryField(t *testing.T) {
	base := map[string]interface{}{
		"id": "", "color": "FFFF00", "itemid": "1", "function": "min",
		"drawtype": "line", "sortorder": "0", "type": "simple", "yaxis_side": "left",
	}

	if hashElementExcept(base, "id") != graphItemHash(base) {
		t.Fatal("graphItemHash is not hashing over the whole element")
	}

	withID := map[string]interface{}{}
	for k, v := range base {
		withID[k] = v
	}
	withID["id"] = "99"
	if graphItemHash(base) != graphItemHash(withID) {
		t.Error("the computed gitemid affects the hash, which would replace every item on every plan")
	}

	for _, field := range []string{"color", "itemid", "function", "drawtype", "sortorder", "type", "yaxis_side"} {
		edited := map[string]interface{}{}
		for k, v := range base {
			edited[k] = v
		}
		edited[field] = "CHANGED"
		if graphItemHash(base) == graphItemHash(edited) {
			t.Errorf("changing %q did not change the hash, so the edit would be silently dropped", field)
		}
	}
}
