package provider

import (
	logger "log"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

// this changes and no longer works if accessed later
var stderr = os.Stderr

type Log struct{}

func (Log) Trace(msg string, args ...interface{}) {
	logger.Printf("[TRACE] "+msg, args...)
}
func (Log) Debug(msg string, args ...interface{}) {
	logger.Printf("[DEBUG] "+msg, args...)
}
func (Log) Info(msg string, args ...interface{}) {
	logger.Printf("[INFO] "+msg, args...)
}
func (Log) Warn(msg string, args ...interface{}) {
	logger.Printf("[WARN] "+msg, args...)
}
func (Log) Error(msg string, args ...interface{}) {
	logger.Printf("[ERROR] "+msg, args...)
}

var log = &Log{}

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
			"zabbix_item_trapper": resourceItemTrapper(),
			"zabbix_item_http":    resourceItemHttp(),
			"zabbix_item_simple":  resourceItemSimple(),
			"zabbix_item_snmp":    resourceItemSnmp(),
			"zabbix_trigger":      resourceTrigger(),
			"zabbix_template":     resourceTemplate(),
			"zabbix_hostgroup":    resourceHostgroup(),
			"zabbix_host":         resourceHost(),
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

func buildHostGroupIds(s *schema.Set) zabbix.HostGroupIDs {
	list := s.List()

	groups := make(zabbix.HostGroupIDs, len(list))

	for i := 0; i < len(list); i++ {
		groups[i] = zabbix.HostGroupID{
			GroupID: list[i].(string),
		}
	}

	return groups
}

func buildTriggerIds(s *schema.Set) zabbix.TriggerIDs {
	list := s.List()

	groups := make(zabbix.TriggerIDs, len(list))

	for i := 0; i < len(list); i++ {
		groups[i] = zabbix.TriggerID{
			TriggerID: list[i].(string),
		}
	}

	return groups
}

func buildTemplateIds(s *schema.Set) zabbix.TemplateIDs {
	list := s.List()

	groups := make(zabbix.TemplateIDs, len(list))

	for i := 0; i < len(list); i++ {
		groups[i] = zabbix.TemplateID{
			TemplateID: list[i].(string),
		}
	}

	return groups
}

// mergeSchemas, take a varadic list of schemas and merge, latter overwrites former
func mergeSchemas(schemas ...map[string]*schema.Schema) map[string]*schema.Schema {
	n := map[string]*schema.Schema{}

	for _, s := range schemas {
		for k, v := range s {
			n[k] = v
		}
	}

	return n
}
