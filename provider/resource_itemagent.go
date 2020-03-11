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

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, map[string]*schema.Schema{
			"active": &schema.Schema{
				Type:        schema.TypeBool,
				Description: "Active zabbix agent Item",
				Optional:    true,
				Default:     false,
			},
		}),
	}
}

func itemAgentModFunc(d *schema.ResourceData, item *zabbix.Item) {
	t := zabbix.ZabbixAgent
	if d.Get("active").(bool) {
		t = zabbix.ZabbixAgentActive
	}
	item.Type = t
	item.InterfaceID = d.Get("interfaceid").(string)
	item.Delay = d.Get("delay").(string)
}

func itemAgentReadFunc(d *schema.ResourceData, item *zabbix.Item) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("delay", item.Delay)
	d.Set("active", item.Type == zabbix.ZabbixAgentActive)
}
