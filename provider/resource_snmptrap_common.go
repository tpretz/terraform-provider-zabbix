package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tomasherout/go-zabbix-api"
)

// terraform resource handler for item type
func resourceItemSnmpTrap() *schema.Resource {
	return &schema.Resource{
		Create: itemGetCreateWrapper(itemSnmpTrapModFunc, itemSnmpTrapReadFunc),
		Read:   itemGetReadWrapper(itemSnmpTrapReadFunc),
		Update: itemGetUpdateWrapper(itemSnmpTrapModFunc, itemSnmpTrapReadFunc),
		Delete: resourceItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: itemCommonSchema,
	}
}
func resourceProtoItemSnmpTrap() *schema.Resource {
	return &schema.Resource{
		Create: protoItemGetCreateWrapper(itemSnmpTrapModFunc, itemSnmpTrapReadFunc),
		Read:   protoItemGetReadWrapper(itemSnmpTrapReadFunc),
		Update: protoItemGetUpdateWrapper(itemSnmpTrapModFunc, itemSnmpTrapReadFunc),
		Delete: resourceProtoItemDelete,
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
