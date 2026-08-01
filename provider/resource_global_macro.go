package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

var GLOBAL_MACRO_TYPE_LOOKUP = map[string]string{
	"text":   "0",
	"secret": "1",
	"vault":  "2",
}

var GLOBAL_MACRO_TYPE_LOOKUP_REV = map[string]string{
	"0": "text",
	"1": "secret",
	"2": "vault",
}

var GLOBAL_MACRO_TYPE_ARR = []string{
	"text",
	"secret",
	"vault",
}

func resourceGlobalMacro() *schema.Resource {
	return &schema.Resource{
		Create: resourceGlobalMacroCreate,
		Read:   resourceGlobalMacroRead,
		Update: resourceGlobalMacroUpdate,
		Delete: resourceGlobalMacroDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"macro": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Macro name including braces, e.g. {$MY_MACRO}",
			},
			"value": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Sensitive:   true,
				Description: "Macro value. For secret macros, this value is write-only.",
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "text",
				ValidateFunc: validation.StringInSlice(GLOBAL_MACRO_TYPE_ARR, false),
				Description:  "Type of macro: text, secret, or vault",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Description of the macro",
			},
		},
	}
}

func globalMacroBuildObject(d *schema.ResourceData) zabbix.GlobalMacro {
	return zabbix.GlobalMacro{
		Macro:       d.Get("macro").(string),
		Value:       d.Get("value").(string),
		Type:        zabbix.GlobalMacroType(GLOBAL_MACRO_TYPE_LOOKUP[d.Get("type").(string)]),
		Description: d.Get("description").(string),
	}
}

func resourceGlobalMacroCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	macro := globalMacroBuildObject(d)
	macros := zabbix.GlobalMacros{macro}

	err := api.GlobalMacrosCreate(macros)
	if err != nil {
		return err
	}

	log.Trace("created global macro: %+v", macros[0])
	d.SetId(macros[0].GlobalMacroID)

	return resourceGlobalMacroRead(d, m)
}

func resourceGlobalMacroRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of global macro with id %s", d.Id())

	macro, err := api.GlobalMacroGetByID(d.Id())
	if err != nil {
		return err
	}
	if macro == nil {
		d.SetId("")
		return nil
	}

	d.SetId(macro.GlobalMacroID)
	d.Set("macro", macro.Macro)
	d.Set("type", GLOBAL_MACRO_TYPE_LOOKUP_REV[string(macro.Type)])
	d.Set("description", macro.Description)

	// Secret macros (type=1) do not return a value from the API.
	// Do not clobber the configured value with empty string, which would cause
	// a permanent diff on every plan.
	if string(macro.Type) != "1" {
		d.Set("value", macro.Value)
	}

	return nil
}

func resourceGlobalMacroUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	macro := globalMacroBuildObject(d)
	macro.GlobalMacroID = d.Id()

	err := api.GlobalMacrosUpdate(zabbix.GlobalMacros{macro})
	if err != nil {
		return err
	}

	return resourceGlobalMacroRead(d, m)
}

func resourceGlobalMacroDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.GlobalMacrosDeleteByIds([]string{d.Id()})
}
