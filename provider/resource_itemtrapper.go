package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

func resourceItemTrapper() *schema.Resource {
	return &schema.Resource{
		Create: itemGetCreateWrapper(itemTrapperModFunc, itemTrapperReadFunc),
		Read:   itemGetReadWrapper(itemTrapperReadFunc),
		Update: itemGetUpdateWrapper(itemTrapperModFunc, itemTrapperReadFunc),
		Delete: resourceItemDelete,

		Schema: itemCommonSchema,
	}
}

func itemTrapperModFunc(d *schema.ResourceData, item *zabbix.Item) {
	item.Type = zabbix.ZabbixTrapper
}

func itemTrapperReadFunc(d *schema.ResourceData, item *zabbix.Item) {
}
