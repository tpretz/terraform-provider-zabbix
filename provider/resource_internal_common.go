package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

// terraform resource handler for item type
func resourceItemInternal() *schema.Resource {
	return &schema.Resource{
		Create: itemGetCreateWrapper(itemInternalModFunc, itemInternalReadFunc),
		Read:   itemGetReadWrapper(itemInternalReadFunc),
		Update: itemGetUpdateWrapper(itemInternalModFunc, itemInternalReadFunc),
		Delete: resourceItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema),
	}
}
func resourceProtoItemInternal() *schema.Resource {
	return &schema.Resource{
		Create: protoItemGetCreateWrapper(itemInternalModFunc, itemInternalReadFunc),
		Read:   protoItemGetReadWrapper(itemInternalReadFunc),
		Update: protoItemGetUpdateWrapper(itemInternalModFunc, itemInternalReadFunc),
		Delete: resourceProtoItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, itemPrototypeSchema),
	}
}
func resourceLLDInternal() *schema.Resource {
	return &schema.Resource{
		Create: lldGetCreateWrapper(lldInternalModFunc, lldInternalReadFunc),
		Read:   lldGetReadWrapper(lldInternalReadFunc),
		Update: lldGetUpdateWrapper(lldInternalModFunc, lldInternalReadFunc),
		Delete: resourceLLDDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(lldCommonSchema, itemInterfaceSchema),
	}
}

// Custom mod handler for item type
func itemInternalModFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	item.Type = zabbix.ZabbixInternal
	item.InterfaceID = d.Get("interfaceid").(string)
	item.Delay = d.Get("delay").(string)
}
func lldInternalModFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
	item.Type = zabbix.ZabbixInternal
	item.InterfaceID = d.Get("interfaceid").(string)
}

// Custom read handler for item type
func itemInternalReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("delay", item.Delay)
}
func lldInternalReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
	d.Set("interfaceid", item.InterfaceID)
}
