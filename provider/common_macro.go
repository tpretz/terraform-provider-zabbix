package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

// macro list schema
var macroListSchema = &schema.Schema{
	Type:     schema.TypeList,
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
	macroCount := d.Get("macro.#").(int)
	macros = make(zabbix.Macros, macroCount)

	for i := 0; i < macroCount; i++ {
		prefix := fmt.Sprintf("macro.%d.", i)

		macros[i] = zabbix.Macro{
			MacroName: d.Get(prefix + "name").(string),
			Value:     d.Get(prefix + "value").(string),
			MacroID:   d.Get(prefix + "id").(string),
		}
	}

	return
}

// flattenMacros convert response to terraform input
func flattenMacros(list zabbix.Macros) []interface{} {
	val := make([]interface{}, len(list))
	for i := 0; i < len(list); i++ {
		val[i] = map[string]interface{}{
			"name":  list[i].MacroName,
			"value": list[i].Value,
			"id":    list[i].MacroID,
		}
	}
	return val
}
