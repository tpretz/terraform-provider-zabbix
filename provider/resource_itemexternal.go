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

// Custom mod handler for item type
func itemExternalModFunc(d *schema.ResourceData, item *zabbix.Item) {
	item.Type = zabbix.ExternalCheck
	item.InterfaceID = d.Get("interfaceid").(string)
	item.Delay = d.Get("delay").(string)
}

// Custom read handler for item type
func itemExternalReadFunc(d *schema.ResourceData, item *zabbix.Item) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("delay", item.Delay)
}
