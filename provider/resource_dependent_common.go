package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// lldDependentDelaySchema pins delay to "0": a dependent LLD rule is driven by
// its master item rather than polled, and Zabbix rejects any other value. It is
// kept in the schema (rather than omitted like itemDelaySchema is for dependent
// items) so the shared LLD read path can always populate it, which import needs.
var lldDependentDelaySchema = map[string]*schema.Schema{
	"delay": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Default:      "0",
		ValidateFunc: validation.StringInSlice([]string{"0"}, false),
		Description:  "LLD Delay period, must be 0 for dependent discovery rules",
	},
}

var schemaDependent = map[string]*schema.Schema{
	"master_itemid": &schema.Schema{
		Type:         schema.TypeString,
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Description:  "Master Item ID",
		Required:     true,
	},
}

// resourceItemDependent terraform resource for agent items
func resourceItemDependent() *schema.Resource {
	return &schema.Resource{
		Create: itemGetCreateWrapper(itemDependentModFunc, itemDependentReadFunc),
		Read:   itemGetReadWrapper(itemDependentReadFunc),
		Update: itemGetUpdateWrapper(itemDependentModFunc, itemDependentReadFunc),
		Delete: resourceItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: mergeSchemas(itemCommonSchema, schemaDependent),
	}
}
func resourceProtoItemDependent() *schema.Resource {
	return &schema.Resource{
		Create: protoItemGetCreateWrapper(itemDependentModFunc, itemDependentReadFunc),
		Read:   protoItemGetReadWrapper(itemDependentReadFunc),
		Update: protoItemGetUpdateWrapper(itemDependentModFunc, itemDependentReadFunc),
		Delete: resourceProtoItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: mergeSchemas(itemCommonSchema, itemPrototypeSchema, schemaDependent),
	}
}
func resourceLLDDependent() *schema.Resource {
	s := mergeSchemas(lldCommonSchema, lldDependentDelaySchema, schemaDependent)
	return &schema.Resource{
		Create: lldGetCreateWrapper(lldDependentModFunc, lldDependentReadFunc),
		Read:   lldGetReadWrapper(lldDependentReadFunc),
		Update: lldGetUpdateWrapper(lldDependentModFunc, lldDependentReadFunc),
		Delete: resourceLLDDelete,
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
