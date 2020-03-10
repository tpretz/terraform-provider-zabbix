package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

// resourceItemAgent terraform resource for agent items
func resourceItemAgent() *schema.Resource {
	return &schema.Resource{
		Create: itemGetCreateWrapper(itemAgentModFunc, itemAgentReadFunc),
		Read:   itemGetReadWrapper(itemAgentReadFunc),
		Update: itemGetUpdateWrapper(itemAgentModFunc, itemAgentReadFunc),
		Delete: resourceItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema),
	}
}

func itemAgentModFunc(d *schema.ResourceData, item *zabbix.Item) {
	item.Type = zabbix.ZabbixAgent
	item.InterfaceID = d.Get("interfaceid").(string)
	item.Delay = d.Get("delay").(string)
}

func itemAgentReadFunc(d *schema.ResourceData, item *zabbix.Item) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("delay", item.Delay)
}
