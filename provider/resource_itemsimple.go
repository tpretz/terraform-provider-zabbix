package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
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

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema),
	}
}

// Custom mod handler for item type
func itemSimpleModFunc(d *schema.ResourceData, item *zabbix.Item) {
	item.Delay = d.Get("delay").(string)
	item.Type = zabbix.SimpleCheck
}

// Custom read handler for item type
func itemSimpleReadFunc(d *schema.ResourceData, item *zabbix.Item) {
	d.Set("delay", item.Delay)
}
