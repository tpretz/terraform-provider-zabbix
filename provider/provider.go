package provider

import (
	"errors"
	logger "log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// Provider definition
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Zabbix API username. Give either username and password, or token. Falls back to $ZABBIX_USER or $ZABBIX_USERNAME",
				ValidateFunc: validation.StringIsNotWhiteSpace,
				DefaultFunc:  schema.MultiEnvDefaultFunc([]string{"ZABBIX_USER", "ZABBIX_USERNAME"}, nil),
			},
			"password": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				Description:  "Zabbix API password. Falls back to $ZABBIX_PASS or $ZABBIX_PASSWORD",
				ValidateFunc: validation.StringIsNotWhiteSpace,
				DefaultFunc:  schema.MultiEnvDefaultFunc([]string{"ZABBIX_PASS", "ZABBIX_PASSWORD"}, nil),
			},
			"token": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				Description:  "Zabbix API token (Zabbix 5.4 and later). Used instead of username and password: the provider sends it directly and skips the login call. Falls back to $ZABBIX_TOKEN",
				ValidateFunc: validation.StringIsNotWhiteSpace,
				DefaultFunc:  schema.MultiEnvDefaultFunc([]string{"ZABBIX_TOKEN"}, nil),
			},
			"url": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Full URL of the Zabbix API endpoint, e.g. https://zabbix.example.com/api_jsonrpc.php. Falls back to $ZABBIX_URL or $ZABBIX_SERVER_URL",
				DefaultFunc:  schema.MultiEnvDefaultFunc([]string{"ZABBIX_URL", "ZABBIX_SERVER_URL"}, nil),
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},
			"tls_insecure": &schema.Schema{
				Type:        schema.TypeBool,
				Description: "Skip TLS certificate verification when talking to the API. For testing only",
				Optional:    true,
				Default:     false,
			},
			"serialize": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				// On by default, and a workaround rather than a feature: Zabbix
				// itself is not safe against concurrent writes. See the long
				// note in internal/zabbix/base.go.
				Default: true,
				Description: "Send mutating API requests one at a time (default `true`). This is a workaround for concurrency bugs in Zabbix, not a tuning knob: " +
					"template inheritance does read-modify-write against shared parent objects, and Terraform's default parallelism of 10 drives exactly that path " +
					"whenever a configuration links several hosts to one template. The observed symptom is a host that ends up with a template's items and none of " +
					"its triggers, silently, surfacing much later as an unrelated `Database error occurred`. Read requests are never serialized, so `plan` and " +
					"`refresh` are unaffected. " +
					"**This only protects a single `terraform apply`** — the lock lives in one provider process, so two concurrent applies, or Terraform racing a " +
					"change made in the Zabbix UI, will still collide. Set to `false` only if you are confident your configuration cannot race and you need the speed.",
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"zabbix_host":          dataHost(),
			"zabbix_proxy":         dataProxy(),
			"zabbix_hostgroup":     dataHostgroup(),
			"zabbix_templategroup": dataTemplategroup(),
			"zabbix_template":      dataTemplate(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"zabbix_trigger":       resourceTrigger(),
			"zabbix_proto_trigger": resourceProtoTrigger(),
			"zabbix_template":      resourceTemplate(),
			"zabbix_hostgroup":     resourceHostgroup(),
			"zabbix_templategroup": resourceTemplategroup(),
			"zabbix_host":          resourceHost(),
			"zabbix_proxy":         resourceProxy(),

			"zabbix_graph":       resourceGraph(),
			"zabbix_proto_graph": resourceProtoGraph(),

			"zabbix_item_trapper":       resourceItemTrapper(),
			"zabbix_proto_item_trapper": resourceProtoItemTrapper(),
			"zabbix_lld_trapper":        resourceLLDTrapper(),

			"zabbix_item_http":       resourceItemHttp(),
			"zabbix_proto_item_http": resourceProtoItemHttp(),
			"zabbix_lld_http":        resourceLLDHttp(),

			"zabbix_item_simple":       resourceItemSimple(),
			"zabbix_proto_item_simple": resourceProtoItemSimple(),
			"zabbix_lld_simple":        resourceLLDSimple(),

			"zabbix_item_external":       resourceItemExternal(),
			"zabbix_proto_item_external": resourceProtoItemExternal(),
			"zabbix_lld_external":        resourceLLDExternal(),

			"zabbix_item_internal":       resourceItemInternal(),
			"zabbix_proto_item_internal": resourceProtoItemInternal(),
			"zabbix_lld_internal":        resourceLLDInternal(),

			"zabbix_item_snmp":       resourceItemSnmp(),
			"zabbix_proto_item_snmp": resourceProtoItemSnmp(),
			"zabbix_lld_snmp":        resourceLLDSnmp(),

			"zabbix_item_snmptrap":       resourceItemSnmpTrap(),
			"zabbix_proto_item_snmptrap": resourceProtoItemSnmpTrap(),

			"zabbix_item_agent":       resourceItemAgent(),
			"zabbix_proto_item_agent": resourceProtoItemAgent(),
			"zabbix_lld_agent":        resourceLLDAgent(),

			"zabbix_item_calculated":       resourceItemCalculated(),
			"zabbix_proto_item_calculated": resourceProtoItemCalculated(),

			"zabbix_item_dependent":       resourceItemDependent(),
			"zabbix_proto_item_dependent": resourceProtoItemDependent(),
			"zabbix_lld_dependent":        resourceLLDDependent(),
		},
		ConfigureFunc: providerConfigure,
	}
}

// providerConfigure configure this provider
func providerConfigure(d *schema.ResourceData) (meta interface{}, err error) {
	log.Trace("Started zabbix provider init")
	l := logger.New(stderr, "[DEBUG] ", logger.LstdFlags)

	// we need one of these options
	if (d.Get("username").(string) == "" ||
		d.Get("password").(string) == "") &&
		d.Get("token").(string) == "" {
		log.Error("credentials required")
		return nil, errors.New("credentials required")
	}

	api, apierr := zabbix.NewAPI(zabbix.Config{
		Url:         d.Get("url").(string),
		TlsNoVerify: d.Get("tls_insecure").(bool),
		Log:         l,
		Serialize:   d.Get("serialize").(bool),
	})
	if apierr != nil {
		return nil, apierr
	}

	if d.Get("token").(string) != "" {
		api.Auth = d.Get("token").(string)
	} else {
		_, err = api.Login(d.Get("username").(string), d.Get("password").(string))
	}
	meta = api
	log.Trace("Started zabbix provider got error: %+v", err)

	return
}
