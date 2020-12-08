package provider

import (
	"errors"
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

// template resource function
func resourceTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceTemplateCreate,
		Read:   resourceTemplateRead,
		Update: resourceTemplateUpdate,
		Delete: resourceTemplateDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"groups": &schema.Schema{
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9]+$"), "must be a numeric string"),
				},
				Required:    true,
				Description: "Host Group IDs",
			},
			"host": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Template hostname (internal name)",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Template description",
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Template Display Name (defaults to host)",
			},
			"templates": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9]+$"), "must be a numeric string"),
				},
				Description: "linked templates",
			},
			"macro": macroListSchema,
		},
	}
}

func dataTemplate() *schema.Resource {
	return &schema.Resource{
		Read: dataTemplateRead,

		Schema: map[string]*schema.Schema{
			"groups": &schema.Schema{
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "Host Group IDs",
			},
			"host": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Template hostname (internal name)",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Template description",
				Computed:    true,
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Template Display Name (defaults to host)",
			},
			"macro": macroListSchema,
		},
	}
}

// terraform resource create handler
func resourceTemplateCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := buildTemplateObject(d)
	items := []zabbix.Template{*item}

	err := api.TemplatesCreate(items)

	if err != nil {
		return err
	}

	log.Trace("crated template: %+v", items[0])

	d.SetId(items[0].TemplateID)

	return resourceTemplateRead(d, m)
}

// terraform template read handler (data source)
func dataTemplateRead(d *schema.ResourceData, m interface{}) error {

	params := zabbix.Params{
		"filter":                map[string]interface{}{},
		"selectMacros":          "extend",
		"selectParentTemplates": "extend",
		"selectGroups":          "extend",
	}

	if v := d.Get("host").(string); v != "" {
		params["filter"].(map[string]interface{})["host"] = v
	}

	if v := d.Get("name").(string); v != "" {
		params["filter"].(map[string]interface{})["name"] = v
	}

	if len(params["filter"].(map[string]interface{})) < 1 {
		return errors.New("no filter parameters provided")
	}
	log.Debug("Lookup of template with: %#v", params)

	return templateRead(d, m, params)
}

// terraform template read handler (resource)
func resourceTemplateRead(d *schema.ResourceData, m interface{}) error {
	log.Debug("Lookup of template with id %s", d.Id())

	return templateRead(d, m, zabbix.Params{
		"templateids":           d.Id(),
		"selectMacros":          "extend",
		"selectParentTemplates": "extend",
		"selectGroups":          "extend",
	})
}

// generic template read function
func templateRead(d *schema.ResourceData, m interface{}, params zabbix.Params) error {
	api := m.(*zabbix.API)

	templates, err := api.TemplatesGet(params)

	if err != nil {
		return err
	}

	if len(templates) < 1 {
		d.SetId("")
		return nil
	}
	if len(templates) > 1 {
		return errors.New("multiple templates found")
	}
	t := templates[0]

	log.Debug("Got template: %+v", t)

	d.Set("description", t.Description)
	d.Set("host", t.Host)
	d.Set("name", t.Name)
	d.Set("macro", flattenMacros(t.UserMacros))
	d.Set("groups", flattenHostGroupIds(t.Groups))
	d.Set("templates", flattenTemplateIds(t.ParentTemplates))
	d.SetId(t.TemplateID)

	return nil
}

// build a template object from terraform data
func buildTemplateObject(d *schema.ResourceData) *zabbix.Template {
	item := zabbix.Template{
		Description:     d.Get("description").(string),
		Name:            d.Get("name").(string),
		Host:            d.Get("host").(string),
		Groups:          buildHostGroupIds(d.Get("groups").(*schema.Set)),
		LinkedTemplates: buildTemplateIds(d.Get("templates").(*schema.Set)),
	}

	item.UserMacros = macroGenerate(d)
	return &item
}

// terraform update resource handler
func resourceTemplateUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := buildTemplateObject(d)
	item.TemplateID = d.Id()

	// templates may need a bit extra effort
	if d.HasChange("templates") {
		old, new := d.GetChange("templates")
		diff := old.(*schema.Set).Difference(new.(*schema.Set))

		// removals, we need to unlink and clear
		if diff.Len() > 0 {
			item.TemplatesClear = buildTemplateIds(diff)
		}
	}

	items := []zabbix.Template{*item}

	err := api.TemplatesUpdate(items)

	if err != nil {
		return err
	}

	return resourceTemplateRead(d, m)
}

// terraform delete handler
func resourceTemplateDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.TemplatesDeleteByIds([]string{d.Id()})
}
