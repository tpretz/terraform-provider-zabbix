package provider

import (
	"fmt"
	"sort"
	"strings"

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
