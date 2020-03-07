package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

func resourceTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceTemplateCreate,
		Read:   resourceTemplateRead,
		Update: resourceTemplateUpdate,
		Delete: resourceTemplateDelete,

		Schema: map[string]*schema.Schema{
			"groups": &schema.Schema{
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required:    true,
				Description: "Zabbix ID",
			},
			"host": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Host",
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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
				Description: "Zabbix ID",
			},
			"host": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Host",
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"macro": macroListSchema,
		},
	}
}

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

func dataTemplateRead(d *schema.ResourceData, m interface{}) error {

	params := zabbix.Params{
		"filter":       map[string]interface{}{},
		"selectMacros": "extend",
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

func resourceTemplateRead(d *schema.ResourceData, m interface{}) error {
	log.Debug("Lookup of template with id %s", d.Id())

	return templateRead(d, m, zabbix.Params{
		"templateids":  d.Id(),
		"selectMacros": "extend",
	})
}

func templateRead(d *schema.ResourceData, m interface{}, params zabbix.Params) error {
	api := m.(*zabbix.API)

	templates, err := api.TemplatesGet(params)

	if err != nil {
		return err
	}

	if len(templates) < 1 {
		return errors.New("no template found")
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
	d.SetId(t.TemplateID)

	return nil
}

func buildTemplateObject(d *schema.ResourceData) *zabbix.Template {
	item := zabbix.Template{
		Description: d.Get("description").(string),
		Name:        d.Get("name").(string),
		Host:        d.Get("host").(string),
		Groups:      buildHostGroupIds(d.Get("groups").(*schema.Set)),
	}

	item.UserMacros = macroGenerate(d)
	return &item
}

func resourceTemplateUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := buildTemplateObject(d)
	item.TemplateID = d.Id()

	items := []zabbix.Template{*item}

	err := api.TemplatesUpdate(items)

	if err != nil {
		return err
	}

	return resourceTemplateRead(d, m)
}

func resourceTemplateDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.TemplatesDeleteByIds([]string{d.Id()})
}
