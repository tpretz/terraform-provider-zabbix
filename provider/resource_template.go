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

func resourceTemplateRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of trigger with id %s", d.Id())

	templates, err := api.TemplatesGet(zabbix.Params{
		"templateids": d.Id(),
	})

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

	return nil
}

func buildTemplateObject(d *schema.ResourceData) *zabbix.Template {
	item := zabbix.Template{
		Description: d.Get("description").(string),
		Name:        d.Get("name").(string),
		Host:        d.Get("host").(string),
		Groups:      buildHostGroupIds(d.Get("groups").(*schema.Set)),
	}
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
