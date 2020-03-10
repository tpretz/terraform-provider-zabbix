package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

func resourceItemSimple() *schema.Resource {
	return &schema.Resource{
		Create: itemGetCreateWrapper(itemSimpleModFunc, itemSimpleReadFunc),
		Read:   itemGetReadWrapper(itemSimpleReadFunc),
		Update: itemGetUpdateWrapper(itemSimpleModFunc, itemSimpleReadFunc),
		Delete: resourceItemDelete,

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema),
	}
}

func itemSimpleModFunc(d *schema.ResourceData, item *zabbix.Item) {
	item.Delay = d.Get("delay").(string)
	item.Type = zabbix.SimpleCheck
}

func itemSimpleReadFunc(d *schema.ResourceData, item *zabbix.Item) {
	d.Set("delay", item.Delay)
}
