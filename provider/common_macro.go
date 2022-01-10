package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/tpretz/go-zabbix-api"
)

// macro list schema
var macroSetSchema = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Macro Name (key)",
			},
			"value": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Macro Value",
			},
		},
	},
}

// macroGenerate build macro structs from terraform inputs
func macroGenerate(d *schema.ResourceData) (macros zabbix.Macros) {
	set := d.Get("macro").(*schema.Set).List()
	macros = make(zabbix.Macros, len(set))

	for i := 0; i < len(set); i++ {
		current := set[i].(map[string]interface{})
		macros[i] = zabbix.Macro{
			MacroName: current["name"].(string),
			Value:     current["value"].(string),
			MacroID:   current["id"].(string),
		}
	}

	return
}

// flattenMacros convert response to terraform input
func flattenMacros(list zabbix.Macros) *schema.Set {
	set := schema.NewSet(func(i interface{}) int {
		m := i.(map[string]interface{})
		return hashcode.String(m["name"].(string))
	}, []interface{}{})

	for i := 0; i < len(list); i++ {
		set.Add(map[string]interface{}{
			"name":  list[i].MacroName,
			"value": list[i].Value,
			"id":    list[i].MacroID,
		})
	}
	return set
}
