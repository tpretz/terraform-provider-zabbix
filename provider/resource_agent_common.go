package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

var schemaAgent = map[string]*schema.Schema{
	"active": &schema.Schema{
		Type:        schema.TypeBool,
		Description: "Active zabbix agent Item",
		Optional:    true,
		Default:     false,
	},
}

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

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, schemaAgent),
	}
}
func resourceProtoItemAgent() *schema.Resource {
	return &schema.Resource{
		Create: protoItemGetCreateWrapper(itemAgentModFunc, itemAgentReadFunc),
		Read:   protoItemGetReadWrapper(itemAgentReadFunc),
		Update: protoItemGetUpdateWrapper(itemAgentModFunc, itemAgentReadFunc),
		Delete: resourceProtoItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, itemPrototypeSchema, schemaAgent),
	}
}
func resourceLLDAgent() *schema.Resource {
	return &schema.Resource{
		Create: lldGetCreateWrapper(lldAgentModFunc, lldAgentReadFunc),
		Read:   lldGetReadWrapper(lldAgentReadFunc),
		Update: lldGetUpdateWrapper(lldAgentModFunc, lldAgentReadFunc),
		Delete: resourceLLDDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(lldCommonSchema, lldInterfaceSchema, schemaAgent),
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

func lldAgentModFunc(d *schema.ResourceData, item *zabbix.LLDRule) {
	t := zabbix.ZabbixAgent
	if d.Get("active").(bool) {
		t = zabbix.ZabbixAgentActive
	}
	item.Type = t
	item.InterfaceID = d.Get("interfaceid").(string)
}

func itemAgentReadFunc(d *schema.ResourceData, item *zabbix.Item) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("delay", item.Delay)
	d.Set("active", item.Type == zabbix.ZabbixAgentActive)
}

func lldAgentReadFunc(d *schema.ResourceData, item *zabbix.LLDRule) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("active", item.Type == zabbix.ZabbixAgentActive)
}
