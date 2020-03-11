package provider

import (
	logger "log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"ZABBIX_USER", "ZABBIX_USERNAME"}, nil),
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"ZABBIX_PASS", "ZABBIX_PASSWORD"}, nil),
			},
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"ZABBIX_URL", "ZABBIX_SERVER_URL"}, nil),
			},
			"tls_insecure": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"serialize": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Serialize API requests",
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"zabbix_host":      dataHost(),
			"zabbix_hostgroup": dataHostgroup(),
			"zabbix_template":  dataTemplate(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"zabbix_item_trapper":   resourceItemTrapper(),
			"zabbix_item_http":      resourceItemHttp(),
			"zabbix_item_simple":    resourceItemSimple(),
			"zabbix_item_internal":  resourceItemInternal(),
			"zabbix_item_snmp":      resourceItemSnmp(),
			"zabbix_item_agent":     resourceItemAgent(),
			"zabbix_item_aggregate": resourceItemAggregate(),
			"zabbix_trigger":        resourceTrigger(),
			"zabbix_template":       resourceTemplate(),
			"zabbix_hostgroup":      resourceHostgroup(),
			"zabbix_host":           resourceHost(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (meta interface{}, err error) {
	log.Trace("Started zabbix provider init")
	l := logger.New(stderr, "[DEBUG] ", logger.LstdFlags)

	api := zabbix.NewAPI(zabbix.Config{
		Url:         d.Get("url").(string),
		TlsNoVerify: d.Get("tls_insecure").(bool),
		Log:         l,
		Serialize:   d.Get("serialize").(bool),
	})

	_, err = api.Login(d.Get("username").(string), d.Get("password").(string))
	meta = api
	log.Trace("Started zabbix provider got error: %+v", err)

	return
}
