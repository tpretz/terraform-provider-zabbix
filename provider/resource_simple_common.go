package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// terraform resource handler for item type
func resourceItemSimple() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Zabbix simple check item: an agentless check such as icmpping or net.tcp.service, performed by the server or proxy.",
		Create:        itemGetCreateWrapper(itemSimpleModFunc, itemSimpleReadFunc),
		Read:          itemGetReadWrapper(itemSimpleReadFunc),
		Update:        itemGetUpdateWrapper(itemSimpleModFunc, itemSimpleReadFunc),
		Delete:        resourceItemDelete,
		CustomizeDiff: itemTrendsCustomizeDiff,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema),
	}
}
func resourceProtoItemSimple() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Zabbix simple check item prototype. One simple check is created from it for each entity its discovery rule finds.",
		Create:        protoItemGetCreateWrapper(itemSimpleModFunc, itemSimpleReadFunc),
		Read:          protoItemGetReadWrapper(itemSimpleReadFunc),
		Update:        protoItemGetUpdateWrapper(itemSimpleModFunc, itemSimpleReadFunc),
		Delete:        resourceProtoItemDelete,
		CustomizeDiff: itemTrendsCustomizeDiff,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, itemPrototypeSchema),
	}
}
func resourceLLDSimple() *schema.Resource {
	s := mergeSchemas(lldCommonSchema, lldDelaySchema, itemInterfaceSchema)
	return &schema.Resource{
		Description: "Manages a Zabbix low-level discovery rule backed by a simple check. Prototypes attached to it are instantiated for every entity it discovers.",
		Create:      lldGetCreateWrapper(lldSimpleModFunc, lldSimpleReadFunc),
		Read:        lldGetReadWrapper(lldSimpleReadFunc),
		Update:      lldGetUpdateWrapper(lldSimpleModFunc, lldSimpleReadFunc),
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

// Custom mod handler for item type
func itemSimpleModFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	item.Delay = d.Get("delay").(string)
	item.Type = zabbix.SimpleCheck
	item.InterfaceID = d.Get("interfaceid").(string)
}
func lldSimpleModFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
	item.Type = zabbix.SimpleCheck
	item.InterfaceID = d.Get("interfaceid").(string)
}

// Custom read handler for item type
func itemSimpleReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("delay", item.Delay)
}
func lldSimpleReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
	d.Set("interfaceid", item.InterfaceID)
}
