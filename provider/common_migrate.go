package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// State upgrades for the v2.0.0 TypeList -> TypeSet conversions.
//
// Three collections modelled as ordered lists are in fact unordered: graph
// `item`, host `interface` and LLD filter `condition`. Zabbix returns all
// three in an order of its own choosing, and that order is not stable across
// versions - 8.0 reorders graph items, and 7.2 reorders LLD filter conditions
// relative to 6.0. A TypeList asserts an order the server does not keep, so
// every one of them is now a TypeSet, which is a breaking schema change.
//
// The upgrade itself is a no-op on the data, and deliberately so:
//
//   - In JSON state (Terraform 0.12+) a list and a set are both encoded as an
//     array. Handing the array straight back is correct; the SDK re-decodes it
//     against the new schema, which is what turns it into a set.
//   - In flatmap state (Terraform 0.11 and earlier, i.e. anyone coming from
//     provider v0.x) the element keys are positional - `item.0.color`. Those
//     cannot be reused, because a set keys its elements by hash. That
//     conversion is handled for us by the SDK, which decodes the flatmap using
//     the `Type` declared on the StateUpgrader below (the *old*, list-shaped
//     schema) before calling the upgrade function.
//
// So all these upgraders have to do is exist, declare the prior shape, and
// pass the data through. Bumping SchemaVersion without them would leave
// Terraform refusing to work with any state written by an older provider.
func typeSetStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	return rawState, nil
}

// asTypeList copies a schema map, replacing one attribute with a TypeList of
// the same elements. Only the container type matters here: it is the one thing
// that differs between the v0 and v1 schemas, and it is the only thing the
// upgrader's cty type needs to get right.
func asTypeList(s map[string]*schema.Schema, key string) map[string]*schema.Schema {
	o := make(map[string]*schema.Schema, len(s))
	for k, v := range s {
		o[k] = v
	}

	if cur, ok := o[key]; ok {
		v0 := *cur
		v0.Type = schema.TypeList
		v0.Set = nil
		o[key] = &v0
	}

	return o
}

// resourceGraphV0 is zabbix_graph / zabbix_proto_graph as shipped through
// v0.17.0, where `item` was a TypeList. Never registered with the provider.
func resourceGraphV0() *schema.Resource {
	return &schema.Resource{Schema: asTypeList(schemaGraph, "item")}
}

func graphStateUpgraders() []schema.StateUpgrader {
	return []schema.StateUpgrader{
		{
			Version: 0,
			Type:    resourceGraphV0().CoreConfigSchema().ImpliedType(),
			Upgrade: typeSetStateUpgradeV0,
		},
	}
}

// resourceHostV0 is zabbix_host as shipped through v0.17.0, where `interface`
// was a TypeList. Never registered with the provider.
func resourceHostV0() *schema.Resource {
	return &schema.Resource{Schema: asTypeList(hostResourceSchema(hostSchemaBase), "interface")}
}

func hostStateUpgraders() []schema.StateUpgrader {
	return []schema.StateUpgrader{
		{
			Version: 0,
			Type:    resourceHostV0().CoreConfigSchema().ImpliedType(),
			Upgrade: typeSetStateUpgradeV0,
		},
	}
}

// Why zabbix_item_* and zabbix_proto_item_* have no state upgrader
//
// Phase 2b deleted `applications` from every item schema, and the legacy SNMP
// item attributes (`snmp_version`, `snmp_community`, `snmp3_*`) with it. A
// v0.17.0 state carries all of them, so the obvious move is a SchemaVersion
// bump plus an upgrader that strips them. It would do nothing.
//
// helper/schema removes attributes the current schema no longer declares
// *before* it decodes prior state - `removeAttributes` in
// grpc_provider.go's UpgradeResourceState, which runs whether or not the
// resource declares an upgrader, on both the JSON and the flatmap path. The
// removed keys are dropped silently, with no diagnostic. An upgrader here
// would be handed state those keys had already been deleted from.
//
// This is not an assumption. TestV0StateFixtureItemsDropRemovedAttributes
// drives real v0.17.0 state through UpgradeResourceState and asserts both
// halves: the upgrade succeeds and the rest of the item survives, *and*
// decoding the same state directly against the v2 schema fails with
// `unsupported attribute "applications"` - so the strip is load-bearing SDK
// behaviour rather than a coincidence. If a future SDK stops doing it, that
// test fails and this comment is the instruction to write the upgraders.
//
// What no upgrader can fix is the *configuration*: `applications = [...]` in
// HCL is now "An argument named "applications" is not expected here". That is
// a hand edit, and MIGRATING.md documents it.

// lldStateUpgraders builds the v0 -> v1 upgrader for one zabbix_lld_* resource.
// Every LLD backend type has a different merged schema, so the prior shape has
// to be derived from the resource's own schema rather than written out once.
func lldStateUpgraders(s map[string]*schema.Schema) []schema.StateUpgrader {
	return []schema.StateUpgrader{
		{
			Version: 0,
			Type:    (&schema.Resource{Schema: asTypeList(s, "condition")}).CoreConfigSchema().ImpliedType(),
			Upgrade: typeSetStateUpgradeV0,
		},
	}
}
