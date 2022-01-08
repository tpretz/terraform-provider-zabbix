package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

var schemaCalculated = map[string]*schema.Schema{
	"formula": &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		Description:  "Formula",
		ValidateFunc: validation.StringIsNotWhiteSpace,
	},
}

// terraform resource handler for item type
func resourceItemCalculated() *schema.Resource {
	return &schema.Resource{
		Create: itemGetCreateWrapper(itemCalculatedModFunc, itemCalculatedReadFunc),
		Read:   itemGetReadWrapper(itemCalculatedReadFunc),
		Update: itemGetUpdateWrapper(itemCalculatedModFunc, itemCalculatedReadFunc),
		Delete: resourceItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, schemaCalculated),
	}
}
func resourceProtoItemCalculated() *schema.Resource {
	return &schema.Resource{
		Create: protoItemGetCreateWrapper(itemCalculatedModFunc, itemCalculatedReadFunc),
		Read:   protoItemGetReadWrapper(itemCalculatedReadFunc),
		Update: protoItemGetUpdateWrapper(itemCalculatedModFunc, itemCalculatedReadFunc),
		Delete: resourceProtoItemDelete,
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
