package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

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

		Schema: mergeSchemas(itemCommonSchema, map[string]*schema.Schema{
			"master_itemid": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Master Item ID",
				Required:     true,
			},
		}),
	}
}

func itemDependentModFunc(d *schema.ResourceData, item *zabbix.Item) {
	t := zabbix.Dependent
	item.Type = t
	item.MasterItemID = d.Get("master_itemid").(string)
}

func itemDependentReadFunc(d *schema.ResourceData, item *zabbix.Item) {
	d.Set("master_itemid", item.MasterItemID)
}
