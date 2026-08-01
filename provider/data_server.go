package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	zabbix "github.com/tpretz/go-zabbix-api"
)

func dataSourceServer() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceServerRead,
		Schema: map[string]*schema.Schema{
			"version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Zabbix server version string (e.g. \"7.0.1\")",
			},
		},
	}
}

func dataSourceServerRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	d.SetId("zabbix_server")
	d.Set("version", api.Config.VersionString)
	return nil
}
