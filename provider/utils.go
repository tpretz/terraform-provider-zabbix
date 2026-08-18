package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

func flattenHostGroupIds(list zabbix.HostGroupIDs) *schema.Set {
	s := schema.NewSet(schema.HashString, []interface{}{})
	for _, v := range list {
		s.Add(v.GroupID)
	}
	return s
}

func flattenTemplateIds(list zabbix.TemplateIDs) *schema.Set {
	s := schema.NewSet(schema.HashString, []interface{}{})
	for _, v := range list {
		s.Add(v.TemplateID)
	}
	return s
}

func buildHostGroupIds(s *schema.Set) zabbix.HostGroupIDs {
	list := s.List()

	groups := make(zabbix.HostGroupIDs, len(list))

	for i := 0; i < len(list); i++ {
		groups[i] = zabbix.HostGroupID{
			GroupID: list[i].(string),
		}
	}

	return groups
}

func buildTriggerIds(s *schema.Set) zabbix.TriggerIDs {
	list := s.List()

	groups := make(zabbix.TriggerIDs, len(list))

	for i := 0; i < len(list); i++ {
		groups[i] = zabbix.TriggerID{
			TriggerID: list[i].(string),
		}
	}

	return groups
}

func buildTemplateIds(s *schema.Set) zabbix.TemplateIDs {
	list := s.List()

	groups := make(zabbix.TemplateIDs, len(list))

	for i := 0; i < len(list); i++ {
		groups[i] = zabbix.TemplateID{
			TemplateID: list[i].(string),
		}
	}

	return groups
}

// derefOr returns *p, or def when p is nil. The API client uses pointers for
// properties that only exist from a given Zabbix version on, where nil means
// "this server does not have the field" as opposed to "the field is empty".
func derefOr[T any](p *T, def T) T {
	if p == nil {
		return def
	}
	return *p
}

// hashElementExcept builds a TypeSet hash over every attribute of a nested
// element except the ones named.
//
// **A set element's hash is its entire identity to helper/schema.** `diffSet`
// short-circuits the moment the old and new sets have the same list of hash
// codes (schema.go, `reflect.DeepEqual(os.listCode(), ns.listCode())`) and
// never looks at the elements themselves, so an attribute left out of the hash
// can never be seen to change: editing it produces an empty plan and the edit
// is silently discarded. Hashing over "just the identifying fields" is
// therefore wrong here, however natural it reads - every user-settable
// attribute has to be in the hash, and the price is that an edit shows up as a
// delete plus an add rather than an update in place.
//
// What must be excluded is the other side of the same coin: a purely Computed
// attribute (a server-assigned id) is empty in config and populated in state,
// so leaving it in the hash makes every element look replaced on every plan.
//
// Values are rendered with %v, which also makes the hash insensitive to
// whether a number arrived as an int or an int64 - `d.Set` hands back whatever
// the flatten function built, and the field readers re-type it.
func hashElementExcept(v interface{}, except ...string) int {
	m, ok := v.(map[string]interface{})
	if !ok {
		return 0
	}

	skip := make(map[string]bool, len(except))
	for _, k := range except {
		skip[k] = true
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		if !skip[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%v;", k, m[k])
	}
	return schema.HashString(b.String())
}

// ptr returns a pointer to v. The API client uses *string for properties
// where "" and "not set" have to be told apart on the wire -- see the note on
// Item.Posts -- and a literal is not addressable.
func ptr[T any](v T) *T {
	return &v
}

// dataSourceFound turns "the lookup matched nothing" into an error, and is
// what every data source read must end with.
//
// The read functions are shared with the resources, and there finding nothing
// is not an error at all: clearing the id is how drift is reported, and it is
// what makes the next plan recreate the object (see acc_drift_test.go). A data
// source has no such recovery, and helper/schema does not treat the empty id
// as a failure either -- it produces a state object with the placeholder id
// "id-attribute-not-set" and every other attribute at its zero value, and
// Terraform reports no problem whatsoever.
//
// So without this the failure surfaces somewhere else entirely: whatever
// referenced the data source is handed the literal string
// "id-attribute-not-set" as an object id and Zabbix rejects it with
// `Invalid parameter "/groupids/1": a number is expected` or, for hosts and
// templates, with the entirely unhelpful "Database error occurred". Naming
// the lookup that failed, at the point it failed, is the whole fix.
func dataSourceFound(d *schema.ResourceData, kind string, by ...string) error {
	if d.Id() != "" {
		return nil
	}

	var terms []string
	for _, k := range by {
		if v, ok := d.GetOk(k); ok {
			terms = append(terms, fmt.Sprintf("%s = %q", k, v))
		}
	}
	if len(terms) == 0 {
		return fmt.Errorf("no %s found", kind)
	}
	return fmt.Errorf("no %s found matching %s", kind, strings.Join(terms, ", "))
}

// mergeSchemas, take a varadic list of schemas and merge, latter overwrites former
func mergeSchemas(schemas ...map[string]*schema.Schema) map[string]*schema.Schema {
	n := map[string]*schema.Schema{}

	for _, s := range schemas {
		for k, v := range s {
			n[k] = v
		}
	}

	return n
}

// visibleNameCustomizeDiff derives the display name of a host or a template
// from its technical name, in the *plan*, for a configuration that does not
// give one.
//
// Zabbix derives `name` from `host` when `name` is empty -- verified on
// 6.0.48, 7.0.29, 7.4.13 and 8.0-trunk for host.create and template.create
// alike -- which is why the attribute is Optional+Computed rather than
// defaulted (R2, acc_removal_test.go). Left at that, the plan for a new host
// says `name = (known after apply)` for a value that is sitting in the same
// resource block, one line up.
//
// Deriving it costs nothing on create, where there is no prior value to
// destroy. On an existing object it is a different question, and the answer
// has to be no in the general case: a host imported into a configuration that
// does not manage `name` has a display name somebody chose, and re-deriving
// would silently overwrite it. That is the concrete harm R2 records against
// re-deriving this attribute, and it is why the derivation is not simply run
// on every plan.
//
// There is one case in between, and it is the one a user hits: renaming
// `host`. Before this, the display name stayed at whatever the technical name
// had been at create -- for ever, and invisibly, since the configuration says
// nothing about it. So the rename is followed **only when the stored display
// name is exactly the old technical name**, which is to say only when it holds
// nothing the derivation did not put there. A display name that differs by so
// much as a capital letter is left alone.
//
// Zabbix's own rule is looser than that: host.update with `host` and no `name`
// overwrites the display name with the new technical name whatever it was,
// which the provider has never triggered because it always sends `name` from
// state. template.update does not do it at all. Mirroring either would mean
// clobbering an imported display name on one resource and not the other, so
// the provider applies the same conservative rule to both.
func visibleNameCustomizeDiff(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	// the configuration owns the value
	if _, given := configuredString(d, "name"); given {
		return nil
	}
	// an unresolved reference: nothing to derive from yet. The create path
	// still sends "" and lets the server derive, as it always did
	if !d.NewValueKnown("host") {
		return nil
	}
	oldHost, newHost := d.GetChange("host")

	if d.Id() == "" {
		return d.SetNew("name", newHost)
	}
	if oldHost.(string) == newHost.(string) {
		return nil
	}
	// a display name of its own is not ours to move
	if d.Get("name").(string) != oldHost.(string) {
		return nil
	}
	return d.SetNew("name", newHost)
}

// rawConfigured is what configuredString needs, and both *schema.ResourceData
// and *schema.ResourceDiff satisfy it -- the same question has to be asked on
// the write path and in CustomizeDiff, and the two carry different types.
type rawConfigured interface {
	GetRawConfig() cty.Value
}

// configuredString reports the value the *configuration* gives a top-level
// string attribute, and whether it gives one at all.
//
// d.Get cannot answer the second question for an Optional+Computed attribute:
// when the configuration is silent it hands back the value in state, which is
// exactly the case that has to be told apart. The raw configuration is the
// only place the difference survives -- an attribute the user did not write is
// null there whatever state holds.
//
// A value that is not yet known (an unresolved reference at plan time) counts
// as "not given", because there is nothing to compare it against; the write
// path is reached again at apply, when it is known.
func configuredString(d rawConfigured, key string) (string, bool) {
	raw := d.GetRawConfig()
	if raw.IsNull() || !raw.Type().IsObjectType() || !raw.Type().HasAttribute(key) {
		return "", false
	}
	v := raw.GetAttr(key)
	if v.IsNull() || !v.IsKnown() {
		return "", false
	}
	return v.AsString(), true
}
