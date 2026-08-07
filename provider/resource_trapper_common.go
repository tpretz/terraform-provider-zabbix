package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// terraform resource handler for item type
func resourceItemTrapper() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a Zabbix trapper item: a value pushed to the server by zabbix_sender or the API rather than polled.",
		Create:      itemGetCreateWrapper(itemTrapperModFunc, itemTrapperReadFunc),
		Read:        itemGetReadWrapper(itemTrapperReadFunc),
		Update:      itemGetUpdateWrapper(itemTrapperModFunc, itemTrapperReadFunc),
		Delete:      resourceItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: itemCommonSchema,
	}
}
func resourceProtoItemTrapper() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a Zabbix trapper item prototype. One trapper item is created from it for each entity its discovery rule finds.",
		Create:      protoItemGetCreateWrapper(itemTrapperModFunc, itemTrapperReadFunc),
		Read:        protoItemGetReadWrapper(itemTrapperReadFunc),
		Update:      protoItemGetUpdateWrapper(itemTrapperModFunc, itemTrapperReadFunc),
		Delete:      resourceProtoItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemPrototypeSchema),
	}
}
func resourceLLDTrapper() *schema.Resource {
	// trapper rules receive pushed data and are not polled: Zabbix requires
	// delay == 0, so the shared 3600 default would fail on create.
	s := mergeSchemas(lldCommonSchema, lldZeroDelaySchema)
	return &schema.Resource{
		Description: "Manages a Zabbix low-level discovery rule fed by pushed data rather than polling. Zabbix requires `delay` to be \"0\" for this rule type.",
		Create:      lldGetCreateWrapper(lldTrapperModFunc, lldTrapperReadFunc),
		Read:        lldGetReadWrapper(lldTrapperReadFunc),
		Update:      lldGetUpdateWrapper(lldTrapperModFunc, lldTrapperReadFunc),
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
func itemTrapperModFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	item.Type = zabbix.ZabbixTrapper
}
func lldTrapperModFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
	item.Type = zabbix.ZabbixTrapper
}

// Custom read handler for item type
func itemTrapperReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
}
func lldTrapperReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
}
