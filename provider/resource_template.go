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

func getHostGroups(api *zabbix.API, s *schema.Set) (groups zabbix.HostGroups, err error) {
	list := s.List()
	strarr := []string{}
	for _, v := range list {
		strarr = append(strarr, v.(string))
	}

	groups, err = api.HostGroupsGet(zabbix.Params{
		"groupids": strarr,
	})

	if err != nil {
		return
	}

	if len(groups) != len(strarr) {
		err = errors.New("incorrect number of host groups, check all ids resolve")
	}

	return
}

func resourceTemplateCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	hostGroups, err := getHostGroups(api, d.Get("groups").(*schema.Set))
	if err != nil {
		return err
	}

	item := zabbix.Template{
		Description: d.Get("description").(string),
		Host:        d.Get("host").(string),
		Name:        d.Get("name").(string),
		Groups:      hostGroups,
	}

	items := []zabbix.Template{item}

	err = api.TemplatesCreate(items)

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

func resourceTemplateUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	hostGroups, err := getHostGroups(api, d.Get("groups").(*schema.Set))
	if err != nil {
		return err
	}

	item := zabbix.Template{
		TemplateID:  d.Id(),
		Description: d.Get("description").(string),
		Name:        d.Get("name").(string),
		Host:        d.Get("host").(string),
		Groups:      hostGroups,
	}

	items := []zabbix.Template{item}

	err = api.TemplatesUpdate(items)

	if err != nil {
		return err
	}

	return resourceTemplateRead(d, m)
}

func resourceTemplateDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.TemplatesDeleteByIds([]string{d.Id()})
}
