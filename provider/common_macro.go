package provider

import (
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

var MACRO_TYPE_LOOKUP = map[string]zabbix.MacroType{
	"text":   zabbix.MacroText,
	"secret": zabbix.MacroSecret,
	"vault":  zabbix.MacroVault,
}
var MACRO_TYPE_LOOKUP_REV = map[zabbix.MacroType]string{}
var MACRO_TYPE_LOOKUP_ARR = []string{}

var _ = func() bool {
	for k, v := range MACRO_TYPE_LOOKUP {
		MACRO_TYPE_LOOKUP_REV[v] = k
		MACRO_TYPE_LOOKUP_ARR = append(MACRO_TYPE_LOOKUP_ARR, k)
	}
	sort.Strings(MACRO_TYPE_LOOKUP_ARR)
	return false
}()

// macroHash hashes a user macro over everything the user writes -- name,
// value, type and description -- and deliberately not over `id`.
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
	Description: "User macros defined on this object (unordered). Zabbix replaces the whole " +
		"macro collection on update, so omitting a macro removes it.",
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Macro ID (hostmacroid), assigned by Zabbix",
			},
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Macro name, in Zabbix's `{$NAME}` form",
			},
			"value": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				// Deliberately NOT Sensitive, though a secret macro holds a
				// credential. `macro` is a set, and Terraform cannot keep a mark
				// on one attribute of a set element: it marks the whole set, so
				// every expression over it -- `m.name`, `m.id` -- turns
				// sensitive, outputs need `sensitive = true` and for_each over
				// it fails. Measured on Terraform 1.15. That would land on every
				// configuration, including ones with no secret at all.
				//
				// A value taken from a sensitive source -- a `sensitive = true`
				// variable, or a data source such as azurerm_key_vault_secret
				// whose attribute is marked -- carries its mark into the macro
				// and is masked in the plan the same way, at the cost of the
				// taint only where it was asked for.
				//
				// No StringIsNotWhiteSpace. Zabbix accepts and stores a macro
				// with an empty value — verified on 7.4 — and placeholder macros
				// left empty are common in shipped templates. The validator made
				// `value = ""` unwritable, which also made any host or template
				// carrying such a macro impossible to import: flattenMacros put
				// "" into state and no configuration could match it.
				// See CLAUDE.md § S9, "a ValidateFunc can make a legal value
				// unreachable".
				Description: "Macro value. For `type = \"vault\"` this is the vault path, " +
					"`path/to/secret:key`. A `secret` value is write-only in the Zabbix API: it is " +
					"kept from configuration, a change made to it outside Terraform is not detected, " +
					"and an imported secret macro reads back with an empty value. The attribute is not " +
					"marked sensitive; pass a secret from a sensitive variable or data source and " +
					"Terraform masks it in plan output",
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "text",
				ValidateFunc: validation.StringInSlice(MACRO_TYPE_LOOKUP_ARR, false),
				Description:  "Macro type, one of: " + strings.Join(MACRO_TYPE_LOOKUP_ARR, ", "),
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Macro description",
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
			MacroName:   current["name"].(string),
			Value:       current["value"].(string),
			MacroID:     current["id"].(string),
			Type:        MACRO_TYPE_LOOKUP[current["type"].(string)],
			Description: current["description"].(string),
		}
	}

	return
}

// flattenMacros convert response to terraform input
//
// prior is the macro set already in d -- state on a refresh, the planned set
// after a create or update, empty on import and in a data source. It is
// consulted for one thing only: Zabbix never returns the value of a secret
// macro, so a secret's value is carried over from the prior element with the
// same name that was also a secret. Writing back the "" the server sent would
// change the element's hash and plan a rewrite on every run.
//
// Matching on the type as well as the name is what keeps a type changed
// outside Terraform visible: a text macro turned secret in the frontend has no
// secret counterpart in prior, reads back with an empty value, and plans a
// correction instead of quietly adopting the old plaintext.
func flattenMacros(list zabbix.Macros, prior *schema.Set) *schema.Set {
	secrets := map[string]string{}
	if prior != nil {
		for _, raw := range prior.List() {
			m := raw.(map[string]interface{})
			if t, _ := m["type"].(string); t == MACRO_TYPE_LOOKUP_REV[zabbix.MacroSecret] {
				secrets[m["name"].(string)] = m["value"].(string)
			}
		}
	}

	// the same hash the schema uses, so the set written here indexes its
	// elements exactly as one read back out of state does
	set := schema.NewSet(macroHash, []interface{}{})

	for i := 0; i < len(list); i++ {
		value := list[i].Value
		if list[i].Type == zabbix.MacroSecret {
			value = secrets[list[i].MacroName]
		}
		set.Add(map[string]interface{}{
			"name":        list[i].MacroName,
			"value":       value,
			"type":        MACRO_TYPE_LOOKUP_REV[list[i].Type],
			"description": list[i].Description,
			"id":          list[i].MacroID,
		})
	}
	return set
}
