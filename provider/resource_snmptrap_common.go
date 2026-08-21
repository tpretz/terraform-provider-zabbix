package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// itemTypesSnmpTrap is the Zabbix item type the SNMP-trap resources
// represent; a read rejects anything else. Note that it is not the same
// type as the SNMP agent items in resource_snmp_common.go, which are polled.
var itemTypesSnmpTrap = itemTypeSet{zabbix.SNMPTrap}

// terraform resource handler for item type
func resourceItemSnmpTrap() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Zabbix SNMP trap item: a value taken from SNMP traps received for the host, rather than polled.",
		Create:        itemGetCreateWrapper(itemSnmpTrapModFunc, itemSnmpTrapReadFunc, itemTypesSnmpTrap),
		Read:          itemGetReadWrapper(itemSnmpTrapReadFunc, itemTypesSnmpTrap),
		Update:        itemGetUpdateWrapper(itemSnmpTrapModFunc, itemSnmpTrapReadFunc, itemTypesSnmpTrap),
		Delete:        resourceItemDelete,
		CustomizeDiff: itemTrendsCustomizeDiff,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: itemCommonSchema,
	}
}
func resourceProtoItemSnmpTrap() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Zabbix SNMP trap item prototype. One SNMP trap item is created from it for each entity its discovery rule finds.",
		Create:        protoItemGetCreateWrapper(itemSnmpTrapModFunc, itemSnmpTrapReadFunc, itemTypesSnmpTrap),
		Read:          protoItemGetReadWrapper(itemSnmpTrapReadFunc, itemTypesSnmpTrap),
		Update:        protoItemGetUpdateWrapper(itemSnmpTrapModFunc, itemSnmpTrapReadFunc, itemTypesSnmpTrap),
		Delete:        resourceProtoItemDelete,
		CustomizeDiff: itemTrendsCustomizeDiff,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemPrototypeSchema),
	}
}

// Custom mod handler for item type
func itemSnmpTrapModFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	item.Type = zabbix.SNMPTrap
}

// Custom read handler for item type
func itemSnmpTrapReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
}
