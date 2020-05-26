package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

// resourceApplication terraform resource handler
func resourceApplication() *schema.Resource {
	return &schema.Resource{
		Create: resourceApplicationCreate,
		Read:   resourceApplicationRead,
		Delete: resourceApplicationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				ForceNew:     true,
				Description:  "Application Name",
				Required:     true,
			},
			"hostid": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				ForceNew:     true,
				Description:  "Host ID",
				Required:     true,
			},
		},
	}
}

// dataApplication terraform data handler
func dataApplication() *schema.Resource {
	return &schema.Resource{
		Read: dataApplicationRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Application Name",
				Required:     true,
			},
			"hostid": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Host ID",
				Optional:     true,
			},
		},
	}
}

// terraform Application create function
func resourceApplicationCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := zabbix.Application{
		Name:   d.Get("name").(string),
		HostID: d.Get("hostid").(string),
	}

	items := []zabbix.Application{item}

	err := api.ApplicationsCreate(items)

	if err != nil {
		return err
	}

	log.Trace("created Application: %+v", items[0])

	d.SetId(items[0].ApplicationID)

	return resourceApplicationRead(d, m)
}

// ApplicationRead terraform Application read function
func ApplicationRead(d *schema.ResourceData, m interface{}, params zabbix.Params) error {
	api := m.(*zabbix.API)

	Applications, err := api.ApplicationsGet(params)

	if err != nil {
		return err
	}

	if len(Applications) < 1 {
		d.SetId("")
		return nil
	}
	if len(Applications) > 1 {
		return errors.New("multiple Applications found")
	}
	t := Applications[0]

	log.Debug("Got Application: %+v", t)

	d.SetId(t.ApplicationID)
	d.Set("name", t.Name)
	d.Set("hostid", t.HostID)

	return nil
}

// dataApplicationRead terraform data resource read handler
func dataApplicationRead(d *schema.ResourceData, m interface{}) error {
	params := zabbix.Params{
		"filter": map[string]interface{}{
			"name": d.Get("name"),
		},
	}

	if v, ok := d.GetOk("hostid"); ok {
		params["filter"].(map[string]interface{})["hostid"] = v
	}
	return ApplicationRead(d, m, params)
}

// resourceApplicationRead terraform resource read handler
func resourceApplicationRead(d *schema.ResourceData, m interface{}) error {
	log.Debug("Lookup of Application with id %s", d.Id())

	return ApplicationRead(d, m, zabbix.Params{
		"applicationids": d.Id(),
	})
}

// resourceApplicationDelete terraform resource delete handler
func resourceApplicationDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.ApplicationsDeleteByIds([]string{d.Id()})
}
