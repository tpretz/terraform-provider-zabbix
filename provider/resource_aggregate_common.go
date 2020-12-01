package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tomasherout/go-zabbix-api"
)

// terraform resource handler for item type
func resourceItemAggregate() *schema.Resource {
	return &schema.Resource{
		Create: itemGetCreateWrapper(itemAggregateModFunc, itemAggregateReadFunc),
		Read:   itemGetReadWrapper(itemAggregateReadFunc),
		Update: itemGetUpdateWrapper(itemAggregateModFunc, itemAggregateReadFunc),
		Delete: resourceItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema),
	}
}
func resourceProtoItemAggregate() *schema.Resource {
	return &schema.Resource{
		Create: protoItemGetCreateWrapper(itemAggregateModFunc, itemAggregateReadFunc),
		Read:   protoItemGetReadWrapper(itemAggregateReadFunc),
		Update: protoItemGetUpdateWrapper(itemAggregateModFunc, itemAggregateReadFunc),
		Delete: resourceProtoItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemPrototypeSchema),
	}
}

// Custom mod handler for item type
func itemAggregateModFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	item.Type = zabbix.ZabbixAggregate
	item.Delay = d.Get("delay").(string)
}

// Custom read handler for item type
func itemAggregateReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	d.Set("delay", item.Delay)
}
