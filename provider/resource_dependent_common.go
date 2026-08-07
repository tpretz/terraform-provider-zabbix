package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

var schemaDependent = map[string]*schema.Schema{
	"master_itemid": &schema.Schema{
		Type:         schema.TypeString,
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Description:  "ID of the item this one derives its value from. The master item must be on the same host",
		Required:     true,
	},
}

// resourceItemDependent terraform resource for agent items
func resourceItemDependent() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a Zabbix dependent item: a value derived from a master item's raw value by preprocessing, so one collection feeds many items.",
		Create:      itemGetCreateWrapper(itemDependentModFunc, itemDependentReadFunc),
		Read:        itemGetReadWrapper(itemDependentReadFunc),
		Update:      itemGetUpdateWrapper(itemDependentModFunc, itemDependentReadFunc),
		Delete:      resourceItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: mergeSchemas(itemCommonSchema, schemaDependent),
	}
}
func resourceProtoItemDependent() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a Zabbix dependent item prototype. One dependent item is created from it for each entity its discovery rule finds.",
		Create:      protoItemGetCreateWrapper(itemDependentModFunc, itemDependentReadFunc),
		Read:        protoItemGetReadWrapper(itemDependentReadFunc),
		Update:      protoItemGetUpdateWrapper(itemDependentModFunc, itemDependentReadFunc),
		Delete:      resourceProtoItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: mergeSchemas(itemCommonSchema, itemPrototypeSchema, schemaDependent),
	}
}
func resourceLLDDependent() *schema.Resource {
	s := mergeSchemas(lldCommonSchema, lldZeroDelaySchema, schemaDependent)
	return &schema.Resource{
		Description: "Manages a Zabbix low-level discovery rule driven by a master item rather than polled. Zabbix requires `delay` to be \"0\" for this rule type.",
		Create:      lldGetCreateWrapper(lldDependentModFunc, lldDependentReadFunc),
		Read:        lldGetReadWrapper(lldDependentReadFunc),
		Update:      lldGetUpdateWrapper(lldDependentModFunc, lldDependentReadFunc),
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

func itemDependentModFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	item.Type = zabbix.Dependent
	item.MasterItemID = d.Get("master_itemid").(string)
}
func lldDependentModFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
	item.Type = zabbix.Dependent
	item.MasterItemID = d.Get("master_itemid").(string)
}

func itemDependentReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	d.Set("master_itemid", item.MasterItemID)
}
func lldDependentReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
	d.Set("master_itemid", item.MasterItemID)
}
