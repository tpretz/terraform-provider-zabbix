package provider

import (
        "github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

func Provider() *schema.Provider {
        return &schema.Provider{
                ResourcesMap: map[string]*schema.Resource{},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (meta interface{}, err error) {
	api := zabbix.NewAPI(d.Get("url").(string))
	_, err = api.Login(d.Get("user").(string), d.Get("password").(string))
        meta = api

	return
}
