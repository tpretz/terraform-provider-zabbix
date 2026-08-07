package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// terraform resource handler for item type
func resourceItemExternal() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a Zabbix external check item: a value produced by a script the server or proxy executes from its ExternalScripts directory.",
		Create:      itemGetCreateWrapper(itemExternalModFunc, itemExternalReadFunc),
		Read:        itemGetReadWrapper(itemExternalReadFunc),
		Update:      itemGetUpdateWrapper(itemExternalModFunc, itemExternalReadFunc),
		Delete:      resourceItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema),
	}
}
func resourceProtoItemExternal() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a Zabbix external check item prototype. One external check is created from it for each entity its discovery rule finds.",
		Create:      protoItemGetCreateWrapper(itemExternalModFunc, itemExternalReadFunc),
		Read:        protoItemGetReadWrapper(itemExternalReadFunc),
		Update:      protoItemGetUpdateWrapper(itemExternalModFunc, itemExternalReadFunc),
		Delete:      resourceProtoItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, itemPrototypeSchema),
	}
}
func resourceLLDExternal() *schema.Resource {
	s := mergeSchemas(lldCommonSchema, lldDelaySchema, itemInterfaceSchema)
	return &schema.Resource{
		Description: "Manages a Zabbix low-level discovery rule backed by an external script. Prototypes attached to it are instantiated for every entity it discovers.",
		Create:      lldGetCreateWrapper(lldExternalModFunc, lldExternalReadFunc),
		Read:        lldGetReadWrapper(lldExternalReadFunc),
		Update:      lldGetUpdateWrapper(lldExternalModFunc, lldExternalReadFunc),
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
func itemExternalModFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	item.Type = zabbix.ExternalCheck
	item.InterfaceID = d.Get("interfaceid").(string)
	item.Delay = d.Get("delay").(string)
}
func lldExternalModFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
	item.Type = zabbix.ExternalCheck
	item.InterfaceID = d.Get("interfaceid").(string)
}

// Custom read handler for item type
func itemExternalReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("delay", item.Delay)
}
func lldExternalReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
	d.Set("interfaceid", item.InterfaceID)
}
