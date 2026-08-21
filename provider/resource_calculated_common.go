package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// itemTypesCalculated is the Zabbix item type the calculated resources
// represent; a read rejects anything else. See item_backend.go.
var itemTypesCalculated = itemTypeSet{zabbix.Calculated}

var schemaCalculated = map[string]*schema.Schema{
	"formula": &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		Description:  "Calculated item formula, e.g. `last(//net.if.in)+last(//net.if.out)`",
		ValidateFunc: validation.StringIsNotWhiteSpace,
	},
}

// terraform resource handler for item type
func resourceItemCalculated() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Zabbix calculated item: a value computed from other items by a formula, rather than collected from the host.",
		Create:        itemGetCreateWrapper(itemCalculatedModFunc, itemCalculatedReadFunc, itemTypesCalculated),
		Read:          itemGetReadWrapper(itemCalculatedReadFunc, itemTypesCalculated),
		Update:        itemGetUpdateWrapper(itemCalculatedModFunc, itemCalculatedReadFunc, itemTypesCalculated),
		Delete:        resourceItemDelete,
		CustomizeDiff: itemTrendsCustomizeDiff,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, schemaCalculated),
	}
}
func resourceProtoItemCalculated() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Zabbix calculated item prototype. One calculated item is created from it for each entity its discovery rule finds.",
		Create:        protoItemGetCreateWrapper(itemCalculatedModFunc, itemCalculatedReadFunc, itemTypesCalculated),
		Read:          protoItemGetReadWrapper(itemCalculatedReadFunc, itemTypesCalculated),
		Update:        protoItemGetUpdateWrapper(itemCalculatedModFunc, itemCalculatedReadFunc, itemTypesCalculated),
		Delete:        resourceProtoItemDelete,
		CustomizeDiff: itemTrendsCustomizeDiff,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemPrototypeSchema, schemaCalculated),
	}
}

// Custom mod handler for item type
func itemCalculatedModFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	item.Type = zabbix.Calculated
	item.Delay = d.Get("delay").(string)
	item.Params = d.Get("formula").(string)
}

// Custom read handler for item type
func itemCalculatedReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	d.Set("delay", item.Delay)
	d.Set("formula", item.Params)
}
