package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// macroHash hashes a user macro over everything the user writes -- name and
// value -- and deliberately not over `id`.
//
// `id` is the server-assigned hostmacroid. A configuration never carries one,
// so including it would make every state element hash differently from its
// config counterpart and produce a diff on every plan. Everything else has to
// be in the hash or changes to it are silently discarded; see
// hashElementExcept.
func macroHash(v interface{}) int {
	return hashElementExcept(v, "id")
}

// macro list schema
var macroSetSchema = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	Set:      macroHash,
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
	// the same hash the schema uses, so the set written here indexes its
	// elements exactly as one read back out of state does
	set := schema.NewSet(macroHash, []interface{}{})

	for i := 0; i < len(list); i++ {
		set.Add(map[string]interface{}{
			"name":  list[i].MacroName,
			"value": list[i].Value,
			"id":    list[i].MacroID,
		})
	}
	return set
}
