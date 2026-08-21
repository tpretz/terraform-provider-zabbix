package provider

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// Backend-type checking for the item / prototype / LLD triad.
//
// Every resource in the triad represents exactly one Zabbix *item type* --
// zabbix_item_snmp represents type 20, zabbix_lld_http represents type 19 --
// and until this file existed nothing anywhere checked that. resourceItemRead
// fetched the item, copied its shared properties into state and called the
// backend's read function, which sets only the attributes its own schema
// declares. An SNMP item read as a zabbix_item_agent therefore populated
// state cleanly: snmp_oid is simply not an attribute of the agent resource,
// nothing in state records the type, and the next plan was **empty**.
//
//	terraform import zabbix_item_agent.x <itemid of an SNMP item>   # succeeded
//
// The item was then managed by the wrong resource, and the first edit to any
// unrelated attribute sent itemAgentModFunc's `type: 0` along with it. Zabbix
// accepted the rewrite and the item stopped collecting data -- silently, with
// its history left behind. The same hole let a type changed in the frontend go
// unreported for ever: drift in the one property that decides what the object
// *is* was the one property never compared.
//
// So a read now rejects an object whose type the resource does not represent.
// Three pieces make that hard to forget:
//
//   - itemGetReadWrapper and its five siblings take an itemTypeSet. A new
//     backend cannot be wired up without naming its types, because the call
//     does not compile otherwise.
//   - itemBackends below maps every Zabbix item type to the resources that
//     represent it, which is what turns the rejection into an actionable
//     message ("import it as zabbix_item_snmp") rather than a complaint.
//   - TestItemBackendTypes drives every registered resource in the triad
//     against a stub Zabbix server, once per item type, and asserts that the
//     set it accepts is exactly the set itemBackends claims for it. A wrong or
//     empty set fails the build without needing a live server.

// itemFamily is which of the three resources built from one
// resource_<backend>_common.go file is doing the reading. It decides both the
// resource-name prefix and the noun the error message uses.
type itemFamily int

const (
	familyItem itemFamily = iota
	familyProtoItem
	familyLLD
)

// prefix is the resource-name prefix for this family.
func (f itemFamily) prefix() string {
	switch f {
	case familyProtoItem:
		return "zabbix_proto_item_"
	case familyLLD:
		return "zabbix_lld_"
	}
	return "zabbix_item_"
}

// noun is what a user calls one of these objects.
func (f itemFamily) noun() string {
	switch f {
	case familyProtoItem:
		return "item prototype"
	case familyLLD:
		return "discovery rule"
	}
	return "item"
}

// itemBackend is one row of the map between Zabbix's item type codes and the
// Terraform resources that represent them.
type itemBackend struct {
	// label is what Zabbix's own documentation and frontend call the type.
	// The mismatch error names it, because "type 20" is not something to show
	// a user.
	label string
	// suffix completes the resource name -- zabbix_item_<suffix>,
	// zabbix_proto_item_<suffix>, zabbix_lld_<suffix>. Empty where this
	// provider has no resource for the type at all.
	suffix string
	// families lists which of the three resources the provider registers for
	// this backend. Zabbix has no calculated or SNMP-trap discovery rule, and
	// the seven backends with no suffix have nothing at all yet.
	families []itemFamily
}

// itemBackends covers every item type Zabbix defines, including the ones this
// provider does not expose: an item of one of those is exactly what a user is
// most likely to import by mistake, and "browser" is a far better answer than
// "type 22". The seven unexposed backends are the resources PLAN.md § 4a still
// has outstanding, plus web items, which Zabbix creates from web scenarios and
// no configuration writes by hand.
var itemBackends = map[zabbix.ItemType]itemBackend{
	zabbix.ZabbixAgent: {
		label: "Zabbix agent", suffix: "agent",
		families: []itemFamily{familyItem, familyProtoItem, familyLLD},
	},
	zabbix.ZabbixTrapper: {
		label: "Zabbix trapper", suffix: "trapper",
		families: []itemFamily{familyItem, familyProtoItem, familyLLD},
	},
	zabbix.SimpleCheck: {
		label: "simple check", suffix: "simple",
		families: []itemFamily{familyItem, familyProtoItem, familyLLD},
	},
	zabbix.ZabbixInternal: {
		label: "Zabbix internal", suffix: "internal",
		families: []itemFamily{familyItem, familyProtoItem, familyLLD},
	},
	// active and passive agent checks are one Terraform resource told apart by
	// the `active` attribute, which is why the accepted set is a set at all
	zabbix.ZabbixAgentActive: {
		label: "Zabbix agent (active)", suffix: "agent",
		families: []itemFamily{familyItem, familyProtoItem, familyLLD},
	},
	zabbix.WebItem:         {label: "web item"},
	zabbix.ExternalCheck:   {label: "external check", suffix: "external", families: []itemFamily{familyItem, familyProtoItem, familyLLD}},
	zabbix.DatabaseMonitor: {label: "database monitor"},
	zabbix.IPMIAgent:       {label: "IPMI agent"},
	zabbix.SSHAgent:        {label: "SSH agent"},
	zabbix.TELNETAgent:     {label: "TELNET agent"},
	zabbix.Calculated: {
		label: "calculated", suffix: "calculated",
		families: []itemFamily{familyItem, familyProtoItem},
	},
	zabbix.JMXAgent: {label: "JMX agent"},
	zabbix.SNMPTrap: {
		label: "SNMP trap", suffix: "snmptrap",
		families: []itemFamily{familyItem, familyProtoItem},
	},
	zabbix.Dependent: {
		label: "dependent", suffix: "dependent",
		families: []itemFamily{familyItem, familyProtoItem, familyLLD},
	},
	zabbix.HTTPAgent: {
		label: "HTTP agent", suffix: "http",
		families: []itemFamily{familyItem, familyProtoItem, familyLLD},
	},
	zabbix.SNMPAgent: {
		label: "SNMP agent", suffix: "snmp",
		families: []itemFamily{familyItem, familyProtoItem, familyLLD},
	},
	zabbix.Script:  {label: "script"},
	zabbix.Browser: {label: "browser"},
}

// itemTypeLabel names an item type the way Zabbix does. A type this provider
// has never heard of -- a server newer than the provider -- falls back to the
// number, which is still more use in an error than nothing.
func itemTypeLabel(t zabbix.ItemType) string {
	if b, ok := itemBackends[t]; ok && b.label != "" {
		return b.label
	}
	return fmt.Sprintf("type %d", int(t))
}

// itemTypeResource names the resource that represents t in family f, or "" if
// this provider has none.
func itemTypeResource(t zabbix.ItemType, f itemFamily) string {
	b, ok := itemBackends[t]
	if !ok || b.suffix == "" {
		return ""
	}
	for _, have := range b.families {
		if have == f {
			return f.prefix() + b.suffix
		}
	}
	return ""
}

// itemFamilyResources lists every resource name this table claims for a
// family. TestItemBackendTypes compares it against the provider's own
// ResourcesMap in both directions, so a backend added to one and not the other
// fails the build.
func itemFamilyResources(f itemFamily) []string {
	seen := map[string]bool{}
	for t := range itemBackends {
		if name := itemTypeResource(t, f); name != "" {
			seen[name] = true
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// itemTypeSet is the set of Zabbix item types one resource represents. All but
// zabbix_item_agent and its two siblings hold exactly one.
type itemTypeSet []zabbix.ItemType

func (s itemTypeSet) contains(t zabbix.ItemType) bool {
	for _, have := range s {
		if have == t {
			return true
		}
	}
	return false
}

// describe renders the set for an error message: "Zabbix agent", or
// "Zabbix agent or Zabbix agent (active)". No article -- the caller adds one,
// because the article depends on the first label.
func (s itemTypeSet) describe() string {
	labels := make([]string, len(s))
	for i, t := range s {
		labels[i] = itemTypeLabel(t)
	}
	return strings.Join(labels, " or ")
}

// checkItemBackendType rejects an object whose Zabbix type the resource doing
// the reading does not represent.
//
// It is an error rather than a d.SetId("") "this is gone" for two reasons.
// Removing it from state would leave the object on the server and the next
// apply would try to create a second one under the same key, which Zabbix
// refuses; and the case that matters most is an import, where a silent empty
// state reads as "that id does not exist" and sends the user looking in the
// wrong place. Saying what the object actually is, and which resource manages
// it, turns both into a one-line fix.
func checkItemBackendType(id string, actual zabbix.ItemType, accepted itemTypeSet, f itemFamily) error {
	if len(accepted) == 0 {
		// a wiring mistake, not a user one
		return fmt.Errorf("internal error: %s %s was read by a resource that declares no item type", f.noun(), id)
	}
	if accepted.contains(actual) {
		return nil
	}

	msg := fmt.Sprintf("%s %s is %s %s, not %s %s",
		f.noun(), id,
		indefinite(itemTypeLabel(actual)), f.noun(),
		indefinite(accepted.describe()), f.noun())

	if name := itemTypeResource(actual, f); name != "" {
		return fmt.Errorf("%s; import it as %s", msg, name)
	}
	return fmt.Errorf("%s, which this provider has no resource for", msg)
}

// indefinite prefixes a label with "a" or "an". Only "external check", the
// initialisms and "item prototype" need "an", but deriving it from the first
// letter beats maintaining a second list.
func indefinite(label string) string {
	if label == "" {
		return "a"
	}
	switch strings.ToLower(label[:1]) {
	case "a", "e", "i", "o", "u":
		return "an " + label
	}
	return "a " + label
}
