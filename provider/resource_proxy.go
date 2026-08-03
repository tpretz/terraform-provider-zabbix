package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
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
	api := m.(*zabbix.API)

	params := zabbix.Params{
		"filter": map[string]interface{}{},
	}

	// Zabbix 7.0 folded the proxy interface into the proxy object itself, so
	// "selectInterface" no longer exists, and renamed the technical name
	// property from "host" to "name". Both are rejected outright on 7.0+.
	nameKey := "host"
	if api.Config.Version >= zabbix.V70 {
		nameKey = "name"
	} else {
		params["selectInterface"] = "extend"
	}

	if v, ok := d.GetOk("host"); ok {
		params["filter"].(map[string]interface{})[nameKey] = v
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
