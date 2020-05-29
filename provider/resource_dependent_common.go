package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

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
	return &schema.Resource{
		Create: lldGetCreateWrapper(lldDependentModFunc, lldDependentReadFunc),
		Read:   lldGetReadWrapper(lldDependentReadFunc),
		Update: lldGetUpdateWrapper(lldDependentModFunc, lldDependentReadFunc),
		Delete: resourceLLDDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: mergeSchemas(lldCommonSchema, schemaDependent),
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
