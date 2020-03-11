package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

// terraform resource handler for item type
func resourceItemTrapper() *schema.Resource {
	return &schema.Resource{
		Create: itemGetCreateWrapper(itemTrapperModFunc, itemTrapperReadFunc),
		Read:   itemGetReadWrapper(itemTrapperReadFunc),
		Update: itemGetUpdateWrapper(itemTrapperModFunc, itemTrapperReadFunc),
		Delete: resourceItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: itemCommonSchema,
	}
}

// Custom mod handler for item type
func itemTrapperModFunc(d *schema.ResourceData, item *zabbix.Item) {
	item.Type = zabbix.ZabbixTrapper
}

// Custom read handler for item type
func itemTrapperReadFunc(d *schema.ResourceData, item *zabbix.Item) {
}
