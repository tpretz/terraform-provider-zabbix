package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/tpretz/go-zabbix-api"
)

// proxySchemaBase base proxy schema
var proxySchemaBase = map[string]*schema.Schema{
	"host": &schema.Schema{
		Type:         schema.TypeString,
		Description:  "FQDN of proxy",
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Required:     true,
	},
}

// dataProxy terraform proxy resource entrypoint
func dataProxy() *schema.Resource {
	return &schema.Resource{
		Read:   dataProxyRead,
		Schema: proxySchemaBase,
	}
}

// dataProxyRead read handler for data resource
func dataProxyRead(d *schema.ResourceData, m interface{}) error {
	params := zabbix.Params{
		"selectInterface": "extend",
		"filter":          map[string]interface{}{},
	}

	lookups := []string{"host"}
	for _, k := range lookups {
		if v, ok := d.GetOk(k); ok {
			params["filter"].(map[string]interface{})[k] = v
		}
	}

	if len(params["filter"].(map[string]interface{})) < 1 {
		return errors.New("no proxy lookup attribute")
	}
	log.Debug("performing data lookup with params: %#v", params)

	return proxyRead(d, m, params)
}

// proxyRead common proxy read function
func proxyRead(d *schema.ResourceData, m interface{}, params zabbix.Params) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of proxy with params %#v", params)

	proxys, err := api.ProxiesGet(params)

	if err != nil {
		return err
	}

	if len(proxys) < 1 {
		d.SetId("")
		return nil
	}
	if len(proxys) > 1 {
		return errors.New("multiple proxys found")
	}
	proxy := proxys[0]

	log.Debug("Got proxy: %+v", proxy)

	d.SetId(proxy.ProxyID)
	d.Set("host", proxy.Host)

	return nil
}
