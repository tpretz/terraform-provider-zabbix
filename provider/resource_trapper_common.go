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
func resourceProtoItemTrapper() *schema.Resource {
	return &schema.Resource{
		Create: protoItemGetCreateWrapper(itemTrapperModFunc, itemTrapperReadFunc),
		Read:   protoItemGetReadWrapper(itemTrapperReadFunc),
		Update: protoItemGetUpdateWrapper(itemTrapperModFunc, itemTrapperReadFunc),
		Delete: resourceProtoItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: itemCommonSchema,
	}
}
func resourceLLDTrapper() *schema.Resource {
	return &schema.Resource{
		Create: lldGetCreateWrapper(lldTrapperModFunc, lldTrapperReadFunc),
		Read:   lldGetReadWrapper(lldTrapperReadFunc),
		Update: lldGetUpdateWrapper(lldTrapperModFunc, lldTrapperReadFunc),
		Delete: resourceLLDDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: lldCommonSchema,
	}
}

// Custom mod handler for item type
func itemTrapperModFunc(d *schema.ResourceData, item *zabbix.Item) {
	item.Type = zabbix.ZabbixTrapper
}
func lldTrapperModFunc(d *schema.ResourceData, item *zabbix.LLDRule) {
	item.Type = zabbix.ZabbixTrapper
}

// Custom read handler for item type
func itemTrapperReadFunc(d *schema.ResourceData, item *zabbix.Item) {
}
func lldTrapperReadFunc(d *schema.ResourceData, item *zabbix.LLDRule) {
}
