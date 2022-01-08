package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

// terraform resource handler for item type
func resourceItemSimple() *schema.Resource {
	return &schema.Resource{
		Create: itemGetCreateWrapper(itemSimpleModFunc, itemSimpleReadFunc),
		Read:   itemGetReadWrapper(itemSimpleReadFunc),
		Update: itemGetUpdateWrapper(itemSimpleModFunc, itemSimpleReadFunc),
		Delete: resourceItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema),
	}
}
func resourceProtoItemSimple() *schema.Resource {
	return &schema.Resource{
		Create: protoItemGetCreateWrapper(itemSimpleModFunc, itemSimpleReadFunc),
		Read:   protoItemGetReadWrapper(itemSimpleReadFunc),
		Update: protoItemGetUpdateWrapper(itemSimpleModFunc, itemSimpleReadFunc),
		Delete: resourceProtoItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, itemPrototypeSchema),
	}
}
func resourceLLDSimple() *schema.Resource {
	return &schema.Resource{
		Create: lldGetCreateWrapper(lldSimpleModFunc, lldSimpleReadFunc),
		Read:   lldGetReadWrapper(lldSimpleReadFunc),
		Update: lldGetUpdateWrapper(lldSimpleModFunc, lldSimpleReadFunc),
		Delete: resourceLLDDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(lldCommonSchema, itemInterfaceSchema),
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
