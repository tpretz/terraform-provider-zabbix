package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

var schemaSnmpAgent = map[string]*schema.Schema{
	"snmp_oid": &schema.Schema{
		Type:         schema.TypeString,
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Description:  "SNMP OID",
		Required:     true,
	},
}

// terraform resource handler for item type
func resourceItemSnmpAgent() *schema.Resource {
	return &schema.Resource{
		Create: itemGetCreateWrapper(itemSnmpAgentModFunc, itemSnmpAgentReadFunc),
		Read:   itemGetReadWrapper(itemSnmpAgentReadFunc),
		Update: itemGetUpdateWrapper(itemSnmpAgentModFunc, itemSnmpAgentReadFunc),
		Delete: resourceItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, schemaSnmpAgent),
	}
}
func resourceProtoItemSnmpAgent() *schema.Resource {
	return &schema.Resource{
		Create: protoItemGetCreateWrapper(itemSnmpAgentModFunc, itemSnmpAgentReadFunc),
		Read:   protoItemGetReadWrapper(itemSnmpAgentReadFunc),
		Update: protoItemGetUpdateWrapper(itemSnmpAgentModFunc, itemSnmpAgentReadFunc),
		Delete: resourceProtoItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, itemPrototypeSchema, schemaSnmpAgent),
	}
}

func resourceLLDSnmpAgent() *schema.Resource {
	return &schema.Resource{
		Create: lldGetCreateWrapper(lldSnmpAgentModFunc, lldSnmpAgentReadFunc),
		Read:   lldGetReadWrapper(lldSnmpAgentReadFunc),
		Update: lldGetUpdateWrapper(lldSnmpAgentModFunc, lldSnmpAgentReadFunc),
		Delete: resourceLLDDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: mergeSchemas(lldCommonSchema, lldInterfaceSchema, schemaSnmpAgent),
	}
}

// Custom mod handler for item type
func itemSnmpAgentModFunc(d *schema.ResourceData, item *zabbix.Item) {
	item.Type = zabbix.SNMPAgent
	item.InterfaceID = d.Get("interfaceid").(string)
	item.Delay = d.Get("delay").(string)
	item.SNMPOid = d.Get("snmp_oid").(string)
}

// Also for LLD Discovery SNMP
func lldSnmpAgentModFunc(d *schema.ResourceData, item *zabbix.LLDRule) {
	item.Type = zabbix.SNMPAgent
	item.InterfaceID = d.Get("interfaceid").(string)
	item.SNMPOid = d.Get("snmp_oid").(string)
}

// Custom read handler for item type
func itemSnmpAgentReadFunc(d *schema.ResourceData, item *zabbix.Item) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("delay", item.Delay)
	d.Set("snmp_oid", item.SNMPOid)
}

// Also for LLD Discovery SNMP
func lldSnmpAgentReadFunc(d *schema.ResourceData, item *zabbix.LLDRule) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("snmp_oid", item.SNMPOid)
}
