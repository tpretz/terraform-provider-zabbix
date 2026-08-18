package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

var schemaSnmp = map[string]*schema.Schema{
	"snmp_oid": &schema.Schema{
		Type:         schema.TypeString,
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Description:  "SNMP object identifier to query, e.g. `1.3.6.1.2.1.1.3.0` or `ifHCInOctets[\"index\",\"eth0\"]`",
		Required:     true,
	},
}

// terraform resource handler for item type
func resourceItemSnmp() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Zabbix SNMP item: a value read from an SNMP OID over the host's SNMP interface.",
		Create:        itemGetCreateWrapper(itemSnmpModFunc, itemSnmpReadFunc),
		Read:          itemGetReadWrapper(itemSnmpReadFunc),
		Update:        itemGetUpdateWrapper(itemSnmpModFunc, itemSnmpReadFunc),
		Delete:        resourceItemDelete,
		CustomizeDiff: itemTrendsCustomizeDiff,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, schemaSnmp),
	}
}
func resourceProtoItemSnmp() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Zabbix SNMP item prototype. One SNMP item is created from it for each entity its discovery rule finds.",
		Create:        protoItemGetCreateWrapper(itemSnmpModFunc, itemSnmpReadFunc),
		Read:          protoItemGetReadWrapper(itemSnmpReadFunc),
		Update:        protoItemGetUpdateWrapper(itemSnmpModFunc, itemSnmpReadFunc),
		Delete:        resourceProtoItemDelete,
		CustomizeDiff: itemTrendsCustomizeDiff,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, itemPrototypeSchema, schemaSnmp),
	}
}

func resourceLLDSnmp() *schema.Resource {
	s := mergeSchemas(lldCommonSchema, lldDelaySchema, lldInterfaceSchema, schemaSnmp)
	return &schema.Resource{
		Description: "Manages a Zabbix low-level discovery rule backed by an SNMP item, typically walking a table such as ifTable. Prototypes attached to it are instantiated for every entity it discovers.",
		Create:      lldGetCreateWrapper(lldSnmpModFunc, lldSnmpReadFunc),
		Read:        lldGetReadWrapper(lldSnmpReadFunc),
		Update:      lldGetUpdateWrapper(lldSnmpModFunc, lldSnmpReadFunc),
		Delete:      resourceLLDDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		// v0 -> v1: "condition" became a TypeSet. See typeSetStateUpgradeV0.
		SchemaVersion:  1,
		StateUpgraders: lldStateUpgraders(s),

		Schema: s,
	}
}

// Custom mod handler for item type
func itemSnmpModFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	item.InterfaceID = d.Get("interfaceid").(string)
	item.Delay = d.Get("delay").(string)
	item.SNMPOid = d.Get("snmp_oid").(string)
	item.Type = zabbix.SNMPAgent
}

// Also for LLD Discovery SNMP
func lldSnmpModFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
	item.InterfaceID = d.Get("interfaceid").(string)
	item.SNMPOid = d.Get("snmp_oid").(string)
	item.Type = zabbix.SNMPAgent
}

// Custom read handler for item type
func itemSnmpReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("delay", item.Delay)
	d.Set("snmp_oid", item.SNMPOid)
}

// Also for LLD Discovery SNMP
func lldSnmpReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("snmp_oid", item.SNMPOid)
}
