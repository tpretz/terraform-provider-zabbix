package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

func dataHost() *schema.Resource {
	return &schema.Resource{
		Read: dataHostRead,
		Schema: map[string]*schema.Schema{
			"hostid": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Host ID",
			},
			"host": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Host FQDN",
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Host name",
			},
			"status": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Status",
			},

			// Fields below used only when creating hosts
			//	GroupIds    HostGroupIDs   `json:"groups,omitempty"`
			//	Interfaces  HostInterfaces `json:"interfaces,omitempty"`
			//	TemplateIDs TemplateIDs    `json:"templates,omitempty"`

		},
	}
}

func dataHostRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	params := zabbix.Params{}

	lookups := []string{"host", "hostid", "name"}
	for _, k := range lookups {
		if v, ok := d.GetOk(k); ok {
			if _, ok := params["filter"]; !ok {
				params["filter"] = map[string]interface{}{}
			}
			params["filter"].(map[string]interface{})[k] = v
		}
	}

	if len(params) < 1 {
		return errors.New("no host lookup attribute")
	}

	hosts, err := api.HostsGet(params)

	if err != nil {
		return err
	}

	if len(hosts) > 1 {
		return errors.New("multiple hosts matching filter, please refine")
	}

	if len(hosts) < 1 {
		return errors.New("no host found matching filter")
	}
	log.Debug("Got host: %+v", hosts[0])

	d.SetId(hosts[0].HostID)
	d.Set("hostid", hosts[0].HostID)
	d.Set("host", hosts[0].Host)
	d.Set("name", hosts[0].Name)
	d.Set("status", hosts[0].Status)

	return nil
}
