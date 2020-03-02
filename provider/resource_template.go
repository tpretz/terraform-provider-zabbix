package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
	"errors"
)

func resourceTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceTemplateCreate,
		Read:   resourceTemplateRead,
		Update: resourceTemplateUpdate,
		Delete: resourceTemplateDelete,

		Schema: map[string]*schema.Schema{
			"templateid": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "Zabbix ID",
			},
			"groups": &schema.Schema{
				Type:        schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required: true,
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

        list := d.Get("groups").(*schema.Set).List()
	strarr := []string{}
        for _, v := range list {
		strarr = append(strarr, v.(string))
        }

        hostGroups, err := api.HostGroupsGet(zabbix.Params{
		"groupids": strarr,
	})

	if err != nil {
		return err
	}

	item := zabbix.Template{
		Description: d.Get("description").(string),
		Host:  d.Get("host").(string),
		Name:    d.Get("name").(string),
		Groups: hostGroups,
	}

	items := []zabbix.Template{item}

	err = api.TemplatesCreate(items)

	if err != nil {
		return err
	}

	log.Trace("crated template: %+v", items[0])

	d.Set("templateid", items[0].TemplateID)
	d.SetId(items[0].TemplateID)

	return resourceTemplateRead(d, m)
}

func resourceTemplateRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	id := d.Get("templateid").(string)

	log.Debug("Lookup of trigger with id %s", id)

	templates, err := api.TemplatesGet(zabbix.Params{
		"templateids": id,
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

	d.Set("templateid", t.TemplateID)
	d.Set("description", t.Description)
	d.Set("host", t.Host)
	d.Set("name", t.Name)

	return nil
}

func resourceTemplateUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := zabbix.Template{
		TemplateID:   d.Id(),
		Description: d.Get("description").(string),
		Name:  d.Get("name").(string),
		Host:    d.Get("host").(string),
	}

	items := []zabbix.Template{item}

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
