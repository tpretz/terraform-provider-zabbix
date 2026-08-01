package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

func resourceProxygroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceProxygroupCreate,
		Read:   resourceProxygroupRead,
		Update: resourceProxygroupUpdate,
		Delete: resourceProxygroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the proxy group",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Description of the proxy group",
			},
			"failover_delay": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "60",
				Description: "Failover delay in seconds",
			},
			"min_online": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "1",
				Description: "Minimum number of online proxies required for the group to be online",
			},
		},
	}
}

func proxygroupBuildObject(d *schema.ResourceData) zabbix.ProxyGroup {
	return zabbix.ProxyGroup{
		Name:          d.Get("name").(string),
		Description:   d.Get("description").(string),
		FailoverDelay: d.Get("failover_delay").(string),
		MinOnline:     d.Get("min_online").(string),
	}
}

func resourceProxygroupCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := proxygroupBuildObject(d)
	items := zabbix.ProxyGroups{item}

	err := api.ProxyGroupsCreate(items)
	if err != nil {
		return err
	}

	log.Trace("created proxygroup: %+v", items[0])
	d.SetId(items[0].ProxyGroupID)

	return resourceProxygroupRead(d, m)
}

func resourceProxygroupRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of proxygroup with id %s", d.Id())

	proxyGroups, err := api.ProxyGroupsGet(zabbix.Params{
		"proxy_groupids": d.Id(),
	})
	if err != nil {
		return err
	}

	if len(proxyGroups) < 1 {
		d.SetId("")
		return nil
	}
	if len(proxyGroups) > 1 {
		return errors.New("multiple proxy groups found")
	}

	pg := proxyGroups[0]

	d.SetId(pg.ProxyGroupID)
	d.Set("name", pg.Name)
	d.Set("description", pg.Description)
	d.Set("failover_delay", pg.FailoverDelay)
	d.Set("min_online", pg.MinOnline)

	return nil
}

func resourceProxygroupUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := proxygroupBuildObject(d)
	item.ProxyGroupID = d.Id()

	err := api.ProxyGroupsUpdate(zabbix.ProxyGroups{item})
	if err != nil {
		return err
	}

	return resourceProxygroupRead(d, m)
}

func resourceProxygroupDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.ProxyGroupsDeleteByIds([]string{d.Id()})
}
