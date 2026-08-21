package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// itemTypesAgent is the pair of Zabbix item types the agent resources
// represent. It is the only backend in the triad that covers more than one:
// passive (0) and active (7) agent checks are one Terraform resource, told
// apart by the `active` attribute, so both have to be accepted on read.
var itemTypesAgent = itemTypeSet{zabbix.ZabbixAgent, zabbix.ZabbixAgentActive}

var schemaAgent = map[string]*schema.Schema{
	"active": &schema.Schema{
		Type:        schema.TypeBool,
		Description: "Use an active agent check: the agent connects to the server and asks for its checks, rather than the server polling the agent. Active checks need no inbound connectivity to the host",
		Optional:    true,
		Default:     false,
	},
}

// resourceItemAgent terraform resource for agent items
func resourceItemAgent() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Zabbix agent item: a value collected by polling the Zabbix agent running on the host.",
		Create:        itemGetCreateWrapper(itemAgentModFunc, itemAgentReadFunc, itemTypesAgent),
		Read:          itemGetReadWrapper(itemAgentReadFunc, itemTypesAgent),
		Update:        itemGetUpdateWrapper(itemAgentModFunc, itemAgentReadFunc, itemTypesAgent),
		Delete:        resourceItemDelete,
		CustomizeDiff: itemTrendsCustomizeDiff,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, schemaAgent),
	}
}
func resourceProtoItemAgent() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Zabbix agent item prototype. One agent item is created from it for each entity its discovery rule finds.",
		Create:        protoItemGetCreateWrapper(itemAgentModFunc, itemAgentReadFunc, itemTypesAgent),
		Read:          protoItemGetReadWrapper(itemAgentReadFunc, itemTypesAgent),
		Update:        protoItemGetUpdateWrapper(itemAgentModFunc, itemAgentReadFunc, itemTypesAgent),
		Delete:        resourceProtoItemDelete,
		CustomizeDiff: itemTrendsCustomizeDiff,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, itemPrototypeSchema, schemaAgent),
	}
}
func resourceLLDAgent() *schema.Resource {
	s := mergeSchemas(lldCommonSchema, lldDelaySchema, lldInterfaceSchema, schemaAgent)
	return &schema.Resource{
		Description: "Manages a Zabbix low-level discovery rule backed by a Zabbix agent item. Item, trigger and graph prototypes attached to it are instantiated for every entity it discovers.",
		Create:      lldGetCreateWrapper(lldAgentModFunc, lldAgentReadFunc, itemTypesAgent),
		Read:        lldGetReadWrapper(lldAgentReadFunc, itemTypesAgent),
		Update:      lldGetUpdateWrapper(lldAgentModFunc, lldAgentReadFunc, itemTypesAgent),
		Delete:      resourceLLDDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		// v0 -> v1: "condition" became a TypeSet. See typeSetStateUpgradeV0.
		SchemaVersion:  1,
		StateUpgraders: lldStateUpgraders(s),

		Schema: s,
	}
}

func itemAgentModFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	t := zabbix.ZabbixAgent
	if d.Get("active").(bool) {
		t = zabbix.ZabbixAgentActive
	}
	item.Type = t
	item.InterfaceID = d.Get("interfaceid").(string)
	item.Delay = d.Get("delay").(string)
}

func lldAgentModFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
	t := zabbix.ZabbixAgent
	if d.Get("active").(bool) {
		t = zabbix.ZabbixAgentActive
	}
	item.Type = t
	item.InterfaceID = d.Get("interfaceid").(string)
}

func itemAgentReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("delay", item.Delay)
	d.Set("active", item.Type == zabbix.ZabbixAgentActive)
}

func lldAgentReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("active", item.Type == zabbix.ZabbixAgentActive)
}
