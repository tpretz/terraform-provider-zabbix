package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

// terraform resource handler for item type
func resourceItemExternal() *schema.Resource {
	return &schema.Resource{
		Create: itemGetCreateWrapper(itemExternalModFunc, itemExternalReadFunc),
		Read:   itemGetReadWrapper(itemExternalReadFunc),
		Update: itemGetUpdateWrapper(itemExternalModFunc, itemExternalReadFunc),
		Delete: resourceItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema),
	}
}
func resourceProtoItemExternal() *schema.Resource {
	return &schema.Resource{
		Create: protoItemGetCreateWrapper(itemExternalModFunc, itemExternalReadFunc),
		Read:   protoItemGetReadWrapper(itemExternalReadFunc),
		Update: protoItemGetUpdateWrapper(itemExternalModFunc, itemExternalReadFunc),
		Delete: resourceProtoItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, itemPrototypeSchema),
	}
}
func resourceLLDExternal() *schema.Resource {
	return &schema.Resource{
		Create: lldGetCreateWrapper(lldExternalModFunc, lldExternalReadFunc),
		Read:   lldGetReadWrapper(lldExternalReadFunc),
		Update: lldGetUpdateWrapper(lldExternalModFunc, lldExternalReadFunc),
		Delete: resourceLLDDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(lldCommonSchema, itemInterfaceSchema),
	}
}

// Custom mod handler for item type
func itemExternalModFunc(d *schema.ResourceData, item *zabbix.Item) {
	item.Type = zabbix.ExternalCheck
	item.InterfaceID = d.Get("interfaceid").(string)
	item.Delay = d.Get("delay").(string)
}
func lldExternalModFunc(d *schema.ResourceData, item *zabbix.LLDRule) {
	item.Type = zabbix.ExternalCheck
	item.InterfaceID = d.Get("interfaceid").(string)
}

// Custom read handler for item type
func itemExternalReadFunc(d *schema.ResourceData, item *zabbix.Item) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("delay", item.Delay)
}
func lldExternalReadFunc(d *schema.ResourceData, item *zabbix.LLDRule) {
	d.Set("interfaceid", item.InterfaceID)
}
