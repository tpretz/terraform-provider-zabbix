package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	ctymsgpack "github.com/hashicorp/go-cty/cty/msgpack"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
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

// ---------------------------------------------------------------------------
// The v0.17.0 state fixtures
//
// testdata/v0.17.0-state.json          - Terraform 0.12+ state (JSON attributes)
// testdata/v0.17.0-state-flatmap.json  - Terraform 0.11 state (flatmap attributes)
//
// Both were written by provider v0.17.0 and between them contain every shape
// v2.0.0 changes: `applications` and the legacy SNMP item attributes that no
// longer exist, the three TypeList -> TypeSet collections, `zabbix_template.
// groups`, and two resources that were deleted outright.
//
// The tests below drive the *real* upgrade path - schema.NewGRPCProviderServer's
// UpgradeResourceState, which is what Terraform calls - rather than invoking the
// upgrade functions directly, because most of the behaviour under test belongs
// to the SDK's surrounding machinery rather than to our upgraders.
// ---------------------------------------------------------------------------

const (
	fixtureJSONPath    = "testdata/v0.17.0-state.json"
	fixtureFlatmapPath = "testdata/v0.17.0-state-flatmap.json"
)

// loadJSONFixture returns the fixture's resources keyed by "<type>.<name>",
// with each instance's attributes as raw JSON exactly as Terraform stores them.
func loadJSONFixture(t *testing.T) map[string]json.RawMessage {
	t.Helper()

	b, err := os.ReadFile(fixtureJSONPath)
	if err != nil {
		t.Fatalf("reading %s: %s", fixtureJSONPath, err)
	}

	var state struct {
		Version   int `json:"version"`
		Resources []struct {
			Type      string `json:"type"`
			Name      string `json:"name"`
			Instances []struct {
				SchemaVersion int             `json:"schema_version"`
				Attributes    json.RawMessage `json:"attributes"`
			} `json:"instances"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(b, &state); err != nil {
		t.Fatalf("parsing %s: %s", fixtureJSONPath, err)
	}
	if state.Version != 4 {
		t.Fatalf("%s: want a version 4 (Terraform 0.12+) state, got %d", fixtureJSONPath, state.Version)
	}

	out := map[string]json.RawMessage{}
	for _, r := range state.Resources {
		for _, i := range r.Instances {
			if i.SchemaVersion != 0 {
				t.Fatalf("%s.%s: fixture must be schema_version 0, got %d", r.Type, r.Name, i.SchemaVersion)
			}
			out[r.Type+"."+r.Name] = i.Attributes
		}
	}
	return out
}

// loadFlatmapFixture returns the flatmap fixture's resources keyed by
// "<type>.<name>", with each one's attributes as the flat string map Terraform
// 0.11 wrote.
func loadFlatmapFixture(t *testing.T) map[string]map[string]string {
	t.Helper()

	b, err := os.ReadFile(fixtureFlatmapPath)
	if err != nil {
		t.Fatalf("reading %s: %s", fixtureFlatmapPath, err)
	}

	var state struct {
		Version int `json:"version"`
		Modules []struct {
			Resources map[string]struct {
				Type    string `json:"type"`
				Primary struct {
					Attributes map[string]string `json:"attributes"`
				} `json:"primary"`
			} `json:"resources"`
		} `json:"modules"`
	}
	if err := json.Unmarshal(b, &state); err != nil {
		t.Fatalf("parsing %s: %s", fixtureFlatmapPath, err)
	}
	if state.Version != 3 {
		t.Fatalf("%s: want a version 3 (Terraform 0.11) state, got %d", fixtureFlatmapPath, state.Version)
	}

	out := map[string]map[string]string{}
	for _, mod := range state.Modules {
		for addr, r := range mod.Resources {
			if r.Type == "" {
				t.Fatalf("%s: resource %q has no type", fixtureFlatmapPath, addr)
			}
			out[addr] = r.Primary.Attributes
		}
	}
	return out
}

// upgradeFixtureState runs one fixture instance through the provider's real
// UpgradeResourceState handler and decodes the result against the *current*
// schema, which is what Terraform does with the response.
func upgradeFixtureState(t *testing.T, p *schema.Provider, typeName string, raw *tfprotov5.RawState) (cty.Value, error) {
	t.Helper()

	resp, err := schema.NewGRPCProviderServer(p).UpgradeResourceState(context.Background(), &tfprotov5.UpgradeResourceStateRequest{
		TypeName: typeName,
		Version:  0,
		RawState: raw,
	})
	if err != nil {
		return cty.NilVal, err
	}

	var errs []string
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov5.DiagnosticSeverityError {
			errs = append(errs, strings.TrimSpace(d.Summary+" "+d.Detail))
		}
	}
	if len(errs) > 0 {
		return cty.NilVal, fmt.Errorf("%s", strings.Join(errs, "; "))
	}

	res, ok := p.ResourcesMap[typeName]
	if !ok {
		return cty.NilVal, fmt.Errorf("%s is not a registered resource", typeName)
	}
	if resp.UpgradedState == nil {
		return cty.NilVal, fmt.Errorf("%s: no upgraded state returned", typeName)
	}
	return ctymsgpack.Unmarshal(resp.UpgradedState.MsgPack, res.CoreConfigSchema().ImpliedType())
}

// attrStrings pulls a set or list of nested objects out of an upgraded value
// and renders one named attribute of each element, sorted so the result does
// not depend on set iteration order.
func attrStrings(t *testing.T, v cty.Value, block, attr string) []string {
	t.Helper()

	col := v.GetAttr(block)
	if col.IsNull() {
		t.Fatalf("%q is null after upgrade", block)
	}
	out := []string{}
	for it := col.ElementIterator(); it.Next(); {
		_, ev := it.Element()
		av := ev.GetAttr(attr)
		if av.IsNull() {
			out = append(out, "")
			continue
		}
		out = append(out, av.AsString())
	}
	sort.Strings(out)
	return out
}

func assertStrings(t *testing.T, what string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %d elements %v, want %d %v", what, len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s: element %d is %q, want %q", what, i, got[i], want[i])
		}
	}
}

// TestV0StateFixtureItemsDropRemovedAttributes is the answer to "what does an
// un-migrated `applications` key actually do".
//
// Nothing. helper/schema's UpgradeResourceState calls removeAttributes over the
// prior state before decoding it (grpc_provider.go), which deletes every key
// the current schema no longer declares. `applications` - and the legacy SNMP
// item attributes that went with the 6.0 floor - are dropped silently, with no
// diagnostic and no SchemaVersion bump required. This test exists to hold that
// behaviour still: if a future SDK ever made an unknown key an error instead,
// every zabbix_item_* resource would need a real upgrader and this fails first.
func TestV0StateFixtureItemsDropRemovedAttributes(t *testing.T) {
	p := Provider()
	fixture := loadJSONFixture(t)

	cases := map[string][]string{
		"zabbix_item_agent.cpu_load": {"applications"},
		"zabbix_item_snmp.if_in": {
			"applications", "snmp_version", "snmp_community",
			"snmp3_authpassphrase", "snmp3_authprotocol", "snmp3_contextname",
			"snmp3_privpassphrase", "snmp3_privprotocol", "snmp3_securitylevel",
			"snmp3_securityname",
		},
		"zabbix_proto_item_agent.fs_used": {"applications"},
	}

	for addr, removed := range cases {
		t.Run(addr, func(t *testing.T) {
			attrs, ok := fixture[addr]
			if !ok {
				t.Fatalf("%s is missing from %s", addr, fixtureJSONPath)
			}

			// The fixture has to actually contain the removed attributes or
			// this test proves nothing.
			var prior map[string]interface{}
			if err := json.Unmarshal(attrs, &prior); err != nil {
				t.Fatalf("parsing fixture attributes: %s", err)
			}
			for _, k := range removed {
				if _, ok := prior[k]; !ok {
					t.Fatalf("fixture no longer carries the removed attribute %q; the test is not exercising anything", k)
				}
			}

			typeName := strings.SplitN(addr, ".", 2)[0]

			// Negative control. The removed keys really are invalid against
			// the v2 schema: decoding the prior state straight into it - what
			// UpgradeResourceState does *after* its removeAttributes pass -
			// fails outright. So the pass is load-bearing, not a no-op, and
			// this test is measuring something.
			var raw map[string]interface{}
			if err := json.Unmarshal(attrs, &raw); err != nil {
				t.Fatalf("parsing fixture attributes: %s", err)
			}
			if _, err := schema.JSONMapToStateValue(raw, p.ResourcesMap[typeName].CoreConfigSchema()); err == nil {
				t.Errorf("decoding v0.17.0 state against the v2 schema unexpectedly succeeded; %v must be genuinely unknown to it", removed)
			}

			got, err := upgradeFixtureState(t, p, typeName, &tfprotov5.RawState{JSON: attrs})
			if err != nil {
				t.Fatalf("upgrading v0.17.0 state containing %v: %s", removed, err)
			}

			for _, k := range removed {
				if got.Type().HasAttribute(k) {
					t.Errorf("%q is still in the v2 schema; the fixture is out of date", k)
				}
			}

			// everything else has to have survived
			for _, k := range []string{"id", "hostid", "key", "name", "valuetype", "delay"} {
				if got.GetAttr(k).IsNull() {
					t.Errorf("%q was lost in the upgrade", k)
				}
				if want, ok := prior[k].(string); ok && got.GetAttr(k).AsString() != want {
					t.Errorf("%q = %q after upgrade, want %q", k, got.GetAttr(k).AsString(), want)
				}
			}
			if got.GetAttr("preprocessor").IsNull() {
				t.Error("preprocessor was lost in the upgrade")
			}
		})
	}
}

// TestV0StateFixtureRemovedResourcesHaveNoUpgradePath pins the other half of
// the item story: zabbix_application and zabbix_item_aggregate are gone, and no
// state upgrader can help - the provider cannot decode their state at all. That
// is why MIGRATING.md tells users to `terraform state rm` them.
func TestV0StateFixtureRemovedResourcesHaveNoUpgradePath(t *testing.T) {
	p := Provider()
	fixture := loadJSONFixture(t)

	for _, addr := range []string{"zabbix_application.general", "zabbix_item_aggregate.cluster_load"} {
		t.Run(addr, func(t *testing.T) {
			attrs, ok := fixture[addr]
			if !ok {
				t.Fatalf("%s is missing from %s", addr, fixtureJSONPath)
			}
			typeName := strings.SplitN(addr, ".", 2)[0]
			if _, ok := p.ResourcesMap[typeName]; ok {
				t.Fatalf("%s is still registered; it was supposed to be removed in v2.0.0", typeName)
			}
			_, err := upgradeFixtureState(t, p, typeName, &tfprotov5.RawState{JSON: attrs})
			if err == nil {
				t.Fatal("expected an error for a resource type the provider no longer has")
			}
			if !strings.Contains(err.Error(), "unknown resource type") {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

// TestV0StateFixtureTypeSetConversions runs the three TypeList -> TypeSet
// collections through the upgrade in JSON form. A list and a set are the same
// array in JSON state, so the interesting part is that nothing is dropped or
// reordered into the wrong element.
func TestV0StateFixtureTypeSetConversions(t *testing.T) {
	p := Provider()
	fixture := loadJSONFixture(t)
	assertTypeSetConversions(t, p, func(addr string) (*tfprotov5.RawState, bool) {
		attrs, ok := fixture[addr]
		if !ok {
			return nil, false
		}
		return &tfprotov5.RawState{JSON: attrs}, true
	})
}

// TestV0FlatmapStateFixtureTypeSetConversions is the case the TypeSet upgraders
// exist for. In flatmap state a list's elements are keyed positionally -
// `item.0.color` - and a set's by hash, so the cty type the upgrader declares
// decides how the SDK reads the prior state. Getting it wrong here is silent
// data loss, and no acceptance test can reach it.
func TestV0FlatmapStateFixtureTypeSetConversions(t *testing.T) {
	p := Provider()
	fixture := loadFlatmapFixture(t)
	assertTypeSetConversions(t, p, func(addr string) (*tfprotov5.RawState, bool) {
		attrs, ok := fixture[addr]
		if !ok {
			return nil, false
		}
		return &tfprotov5.RawState{Flatmap: attrs}, true
	})
}

// assertTypeSetConversions is shared by the JSON and flatmap fixture tests: the
// expected result is identical, only the encoding of the prior state differs.
func assertTypeSetConversions(t *testing.T, p *schema.Provider, raw func(string) (*tfprotov5.RawState, bool)) {
	t.Helper()

	cases := []struct {
		addr  string
		block string
		attr  string
		want  []string
	}{
		{"zabbix_host.web01", "interface", "id", []string{"5", "6"}},
		{"zabbix_host.web01", "macro", "name", []string{"{$ROLE}", "{$TIER}"}},
		{"zabbix_graph.cpu", "item", "itemid", []string{"31201", "31202"}},
		{"zabbix_proto_graph.fs", "item", "itemid", []string{"31203"}},
		{"zabbix_lld_agent.mounts", "condition", "macro", []string{"{#FSNAME}", "{#FSTYPE}"}},
		{"zabbix_template.linux", "macro", "name", []string{"{$CPU_LOAD_CRIT}"}},
	}

	for _, c := range cases {
		t.Run(c.addr+"."+c.block, func(t *testing.T) {
			rs, ok := raw(c.addr)
			if !ok {
				t.Skipf("%s is not in this fixture", c.addr)
			}
			typeName := strings.SplitN(c.addr, ".", 2)[0]

			got, err := upgradeFixtureState(t, p, typeName, rs)
			if err != nil {
				t.Fatalf("upgrade: %s", err)
			}

			// the current schema must be a set...
			if ty := got.Type().AttributeType(c.block); !ty.IsSetType() {
				t.Fatalf("%q is %s in the v2 schema, want a set", c.block, ty.FriendlyName())
			}
			// ...and every element must have come through
			assertStrings(t, c.addr+"."+c.block+"."+c.attr, attrStrings(t, got, c.block, c.attr), c.want)
		})
	}
}

// TestV0FlatmapStateFixtureItems checks the removed item attributes are dropped
// from flatmap state as well, and that nothing else is lost on the way through
// hcl2shim's flatmap decoder.
func TestV0FlatmapStateFixtureItems(t *testing.T) {
	p := Provider()
	fixture := loadFlatmapFixture(t)

	for _, addr := range []string{
		"zabbix_item_agent.cpu_load",
		"zabbix_item_snmp.if_in",
		"zabbix_proto_item_agent.fs_used",
	} {
		t.Run(addr, func(t *testing.T) {
			attrs, ok := fixture[addr]
			if !ok {
				t.Fatalf("%s is missing from %s", addr, fixtureFlatmapPath)
			}
			if _, ok := attrs["applications.#"]; !ok {
				t.Fatal("fixture no longer carries applications; the test is not exercising anything")
			}

			typeName := strings.SplitN(addr, ".", 2)[0]
			got, err := upgradeFixtureState(t, p, typeName, &tfprotov5.RawState{Flatmap: attrs})
			if err != nil {
				t.Fatalf("upgrading flatmap state containing applications: %s", err)
			}

			for _, k := range []string{"id", "hostid", "key", "name", "valuetype"} {
				if got.GetAttr(k).IsNull() {
					t.Errorf("%q was lost in the upgrade", k)
				} else if want := attrs[k]; want != "" && got.GetAttr(k).AsString() != want {
					t.Errorf("%q = %q after upgrade, want %q", k, got.GetAttr(k).AsString(), want)
				}
			}
		})
	}
}

// TestV0FlatmapStateFixtureSNMPItemPreprocessor guards the one nested list that
// is still a list: preprocessing steps are ordered and must survive flatmap
// decoding in the order they were written.
func TestV0FlatmapStateFixtureSNMPItemPreprocessor(t *testing.T) {
	p := Provider()
	fixture := loadFlatmapFixture(t)

	attrs := fixture["zabbix_item_snmp.if_in"]
	got, err := upgradeFixtureState(t, p, "zabbix_item_snmp", &tfprotov5.RawState{Flatmap: attrs})
	if err != nil {
		t.Fatalf("upgrade: %s", err)
	}

	pre := got.GetAttr("preprocessor")
	if pre.IsNull() || pre.LengthInt() != 1 {
		t.Fatalf("preprocessor = %#v, want one step", pre)
	}
	step := pre.Index(cty.NumberIntVal(0))
	if v := step.GetAttr("type").AsString(); v != "1" {
		t.Errorf("preprocessor.0.type = %q, want \"1\"", v)
	}
	params := step.GetAttr("params")
	if params.IsNull() || params.LengthInt() != 1 || params.Index(cty.NumberIntVal(0)).AsString() != "8" {
		t.Errorf("preprocessor.0.params = %#v, want [\"8\"]", params)
	}
}

// ---------------------------------------------------------------------------
// zabbix_template.groups - verify, don't rewrite
// ---------------------------------------------------------------------------

// fakeZabbix stands up an httptest server answering the handful of JSON-RPC
// methods the template upgrader calls, and returns an API client pointed at it.
// The upgrader's whole job is deciding what to do with what templategroup.get
// says, so the interesting cases are server responses, not server versions.
func fakeZabbix(t *testing.T, version string, results map[string]interface{}) *zabbix.API {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			ID     int32  `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("fake zabbix: bad request: %s", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		res, ok := results[req.Method]
		if !ok {
			t.Errorf("fake zabbix: unexpected method %q", req.Method)
			res = []interface{}{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  res,
		}); err != nil {
			t.Errorf("fake zabbix: %s", err)
		}
	}))
	t.Cleanup(srv.Close)

	if results == nil {
		results = map[string]interface{}{}
	}
	results["APIInfo.version"] = version

	api, err := zabbix.NewAPI(zabbix.Config{Url: srv.URL})
	if err != nil {
		t.Fatalf("fake zabbix: %s", err)
	}
	return api
}

// TestTemplateStateUpgradeVerifiesFixtureGroups drives the two branches that
// need a server - the verify pass-through and the refusal - off the checked-in
// fixture's real v0.17.0 `groups` value, with no live Zabbix.
//
// TestAccTemplateStateUpgradeV0 (resource_templategroup_test.go) already covers
// the same two against a real server, and TestTemplateStateUpgradeV0Offline the
// nil-meta and pre-6.2 branches. What is added here is that they run in plain
// `go test ./provider/`, and that the input is a state file rather than a map
// written to suit the assertion.
//
// The upgrader must never rewrite an id: Zabbix's 6.2 split kept the id for some
// groups and allocated a new one for others, so translating is guesswork, and
// pointing a template at the wrong group is worse than stopping.
func TestTemplateStateUpgradeVerifiesFixtureGroups(t *testing.T) {
	fixture := loadJSONFixture(t)
	var prior map[string]interface{}
	if err := json.Unmarshal(fixture["zabbix_template.linux"], &prior); err != nil {
		t.Fatalf("parsing fixture attributes: %s", err)
	}
	if ids := stateStringList(prior["groups"]); len(ids) != 1 || ids[0] != "16" {
		t.Fatalf("fixture zabbix_template.linux groups = %v, want [16]", ids)
	}

	t.Run("ids are already template groups", func(t *testing.T) {
		// The converted-in-place case, and every re-run after a manual fix.
		api := fakeZabbix(t, "7.4.13", map[string]interface{}{
			"templategroup.get": []map[string]string{{"groupid": "16"}},
		})
		out, err := resourceTemplateStateUpgradeV0(context.Background(), prior, api)
		if err != nil {
			t.Fatalf("want a pass-through for a valid template group, got %s", err)
		}
		assertStrings(t, "groups", stateStringList(out["groups"]), []string{"16"})
	})

	t.Run("ids are stale host groups", func(t *testing.T) {
		api := fakeZabbix(t, "7.4.13", map[string]interface{}{
			"templategroup.get": []map[string]string{},
			"hostgroup.get":     []map[string]string{{"groupid": "16", "name": "Templates/Applications"}},
		})
		out, err := resourceTemplateStateUpgradeV0(context.Background(), prior, api)
		if err == nil {
			t.Fatalf("want an error for a host group id, got a pass-through: %v", out)
		}
		if out != nil {
			t.Error("the upgrader must not return partially-rewritten state alongside an error")
		}

		msg := err.Error()
		for _, want := range []string{
			`zabbix_template "tf-linux-base"`,          // which resource
			`16 (host group "Templates/Applications")`, // which id, and its likely replacement
			"zabbix_templategroup",                     // what to use instead
			"terraform import zabbix_template",         // how to fix state
			"MIGRATING.md",                             // where the full story is
		} {
			if !strings.Contains(msg, want) {
				t.Errorf("error message does not mention %q:\n%s", want, msg)
			}
		}
	})
}

// TestTemplateStateUpgradeWiredIntoResource proves the verify step is reached
// through the same UpgradeResourceState path Terraform uses, meta and all -
// the upgrader being correct is no use if the resource does not declare it.
func TestTemplateStateUpgradeWiredIntoResource(t *testing.T) {
	fixture := loadJSONFixture(t)
	attrs := fixture["zabbix_template.linux"]

	r := Provider().ResourcesMap["zabbix_template"]
	if r.SchemaVersion != 1 {
		t.Fatalf("zabbix_template SchemaVersion = %d, want 1", r.SchemaVersion)
	}
	if len(r.StateUpgraders) != 1 || r.StateUpgraders[0].Version != 0 {
		t.Fatalf("want exactly one v0 state upgrader, got %+v", r.StateUpgraders)
	}

	t.Run("valid template group", func(t *testing.T) {
		p := Provider()
		p.SetMeta(fakeZabbix(t, "7.4.13", map[string]interface{}{
			"templategroup.get": []map[string]string{{"groupid": "16"}},
		}))

		got, err := upgradeFixtureState(t, p, "zabbix_template", &tfprotov5.RawState{JSON: attrs})
		if err != nil {
			t.Fatalf("upgrade: %s", err)
		}
		assertStrings(t, "groups", ctyStringSet(t, got.GetAttr("groups")), []string{"16"})
	})

	t.Run("stale host group", func(t *testing.T) {
		p := Provider()
		p.SetMeta(fakeZabbix(t, "7.4.13", map[string]interface{}{
			"templategroup.get": []map[string]string{},
			"hostgroup.get":     []map[string]string{{"groupid": "16", "name": "Templates/Applications"}},
		}))

		_, err := upgradeFixtureState(t, p, "zabbix_template", &tfprotov5.RawState{JSON: attrs})
		if err == nil {
			t.Fatal("expected the upgrade to refuse a host group id")
		}
		if !strings.Contains(err.Error(), "MIGRATING.md") {
			t.Errorf("error does not point at the migration guide:\n%s", err)
		}
	})
}

// ctyStringSet renders a set (or list) of strings, sorted.
func ctyStringSet(t *testing.T, v cty.Value) []string {
	t.Helper()
	if v.IsNull() {
		return nil
	}
	out := []string{}
	for it := v.ElementIterator(); it.Next(); {
		_, ev := it.Element()
		out = append(out, ev.AsString())
	}
	sort.Strings(out)
	return out
}
