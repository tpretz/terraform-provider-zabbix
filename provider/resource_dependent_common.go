package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// itemTypesDependent is the Zabbix item type the dependent resources
// represent; a read rejects anything else. See item_backend.go.
var itemTypesDependent = itemTypeSet{zabbix.Dependent}

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
		Description:   "Manages a Zabbix dependent item: a value derived from a master item's raw value by preprocessing, so one collection feeds many items.",
		Create:        itemGetCreateWrapper(itemDependentModFunc, itemDependentReadFunc, itemTypesDependent),
		Read:          itemGetReadWrapper(itemDependentReadFunc, itemTypesDependent),
		Update:        itemGetUpdateWrapper(itemDependentModFunc, itemDependentReadFunc, itemTypesDependent),
		Delete:        resourceItemDelete,
		CustomizeDiff: itemTrendsCustomizeDiff,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: mergeSchemas(itemCommonSchema, schemaDependent),
	}
}
func resourceProtoItemDependent() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Zabbix dependent item prototype. One dependent item is created from it for each entity its discovery rule finds.",
		Create:        protoItemGetCreateWrapper(itemDependentModFunc, itemDependentReadFunc, itemTypesDependent),
		Read:          protoItemGetReadWrapper(itemDependentReadFunc, itemTypesDependent),
		Update:        protoItemGetUpdateWrapper(itemDependentModFunc, itemDependentReadFunc, itemTypesDependent),
		Delete:        resourceProtoItemDelete,
		CustomizeDiff: itemTrendsCustomizeDiff,
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
		Create:      lldGetCreateWrapper(lldDependentModFunc, lldDependentReadFunc, itemTypesDependent),
		Read:        lldGetReadWrapper(lldDependentReadFunc, itemTypesDependent),
		Update:      lldGetUpdateWrapper(lldDependentModFunc, lldDependentReadFunc, itemTypesDependent),
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
