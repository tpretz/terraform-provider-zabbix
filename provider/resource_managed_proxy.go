package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

var proxyOperatingModeMap = map[string]string{
	"active":  "0",
	"passive": "1",
}
var proxyOperatingModeReverseMap = map[string]string{
	"0": "active",
	"1": "passive",
}
var proxyOperatingModeArr = []string{"active", "passive"}

var proxyTLSModeMap = map[string]string{
	"unencrypted": "1",
	"psk":         "2",
	"certificate": "4",
}
var proxyTLSModeReverseMap = map[string]string{
	"1": "unencrypted",
	"2": "psk",
	"4": "certificate",
}
var proxyTLSConnectArr = []string{"unencrypted", "psk", "certificate"}

// tls_accept is a bitmask for active proxies: can combine psk+cert
// For simplicity, we use the same set but also allow combined values
var proxyTLSAcceptArr = []string{"unencrypted", "psk", "certificate"}

var proxyCustomTimeoutsMap = map[string]string{
	"disabled": "0",
	"enabled":  "1",
}
var proxyCustomTimeoutsReverseMap = map[string]string{
	"0": "disabled",
	"1": "enabled",
}
var proxyCustomTimeoutsArr = []string{"disabled", "enabled"}

func resourceManagedProxy() *schema.Resource {
	return &schema.Resource{
		Create: resourceManagedProxyCreate,
		Read:   resourceManagedProxyRead,
		Update: resourceManagedProxyUpdate,
		Delete: resourceManagedProxyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Technical name of the proxy",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"operating_mode": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Proxy operating mode: active or passive",
				ValidateFunc: validation.StringInSlice(proxyOperatingModeArr, false),
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Description of the proxy",
			},
			"allowed_addresses": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Comma-delimited IP addresses or DNS names of active proxy",
			},
			"address": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "IP address or DNS name to connect to for passive proxy. Defaults to 127.0.0.1 for active.",
			},
			"port": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Port number to connect to for passive proxy. Defaults to 10051.",
			},
			"tls_connect": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "unencrypted",
				Description:  "How the server connects to the proxy: unencrypted, psk, certificate",
				ValidateFunc: validation.StringInSlice(proxyTLSConnectArr, false),
			},
			"tls_accept": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "unencrypted",
				Description:  "What type of connections to accept from the proxy: unencrypted, psk, certificate",
				ValidateFunc: validation.StringInSlice(proxyTLSAcceptArr, false),
			},
			"tls_issuer": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Certificate issuer",
			},
			"tls_subject": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Certificate subject",
			},
			"tls_psk_identity": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Sensitive:   true,
				Description: "PSK identity string",
			},
			"tls_psk": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Sensitive:   true,
				Description: "PSK value (hex string, min 32 chars)",
			},
			"custom_timeouts": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "disabled",
				Description:  "Whether to use custom timeout values: disabled, enabled",
				ValidateFunc: validation.StringInSlice(proxyCustomTimeoutsArr, false),
			},
			"timeout_zabbix_agent": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Timeout for Zabbix agent checks (with suffix, e.g. 3s)",
			},
			"timeout_simple_check": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Timeout for simple checks",
			},
			"timeout_snmp_agent": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Timeout for SNMP agent checks",
			},
			"timeout_external_check": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Timeout for external checks",
			},
			"timeout_db_monitor": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Timeout for database monitor checks",
			},
			"timeout_http_agent": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Timeout for HTTP agent checks",
			},
			"timeout_ssh_agent": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Timeout for SSH agent checks",
			},
			"timeout_telnet_agent": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Timeout for telnet agent checks",
			},
			"timeout_script": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Timeout for script checks",
			},
			"timeout_browser": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Timeout for browser checks",
			},
			"proxy_groupid": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "0",
				Description: "ID of the proxy group the proxy belongs to, 0 for none",
			},
			"local_address": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Local address for the proxy within a proxy group",
			},
			"local_port": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Local port for the proxy within a proxy group",
			},
		},
	}
}

func proxyBuildObject(d *schema.ResourceData) zabbix.ManagedProxy {
	p := zabbix.ManagedProxy{
		Name:             d.Get("name").(string),
		OperatingMode:    proxyOperatingModeMap[d.Get("operating_mode").(string)],
		Description:      d.Get("description").(string),
		AllowedAddresses: d.Get("allowed_addresses").(string),
		TLSConnect:       proxyTLSModeMap[d.Get("tls_connect").(string)],
		TLSAccept:        proxyTLSModeMap[d.Get("tls_accept").(string)],
		TLSIssuer:        d.Get("tls_issuer").(string),
		TLSSubject:       d.Get("tls_subject").(string),
		TLSPSKIdentity:   d.Get("tls_psk_identity").(string),
		TLSPSK:           d.Get("tls_psk").(string),
		CustomTimeouts:   proxyCustomTimeoutsMap[d.Get("custom_timeouts").(string)],
		ProxyGroupID:     d.Get("proxy_groupid").(string),
		LocalAddress:     d.Get("local_address").(string),
	}

	// address and port are only valid for passive proxy
	if d.Get("operating_mode").(string) == "passive" {
		p.Address = d.Get("address").(string)
		p.Port = d.Get("port").(string)
	}

	// local_port
	if v := d.Get("local_port").(string); v != "" {
		p.LocalPort = v
	}

	// timeout fields - only send when custom_timeouts is enabled
	if d.Get("custom_timeouts").(string) == "enabled" {
		p.TimeoutZabbixAgent = d.Get("timeout_zabbix_agent").(string)
		p.TimeoutSimpleCheck = d.Get("timeout_simple_check").(string)
		p.TimeoutSNMPAgent = d.Get("timeout_snmp_agent").(string)
		p.TimeoutExternalCheck = d.Get("timeout_external_check").(string)
		p.TimeoutDBMonitor = d.Get("timeout_db_monitor").(string)
		p.TimeoutHTTPAgent = d.Get("timeout_http_agent").(string)
		p.TimeoutSSHAgent = d.Get("timeout_ssh_agent").(string)
		p.TimeoutTelnetAgent = d.Get("timeout_telnet_agent").(string)
		p.TimeoutScript = d.Get("timeout_script").(string)
		p.TimeoutBrowser = d.Get("timeout_browser").(string)
	}

	return p
}

func resourceManagedProxyCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := proxyBuildObject(d)
	items := zabbix.ManagedProxies{item}

	err := api.ManagedProxiesCreate(items)
	if err != nil {
		return err
	}

	log.Trace("created proxy: %+v", items[0])
	d.SetId(items[0].ProxyID)

	return resourceManagedProxyRead(d, m)
}

func resourceManagedProxyRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of proxy with id %s", d.Id())

	proxies, err := api.ManagedProxiesGet(zabbix.Params{
		"proxyids": d.Id(),
	})
	if err != nil {
		return err
	}

	if len(proxies) < 1 {
		d.SetId("")
		return nil
	}
	if len(proxies) > 1 {
		return errors.New("multiple proxies found")
	}

	p := proxies[0]

	d.SetId(p.ProxyID)
	d.Set("name", p.Name)
	d.Set("operating_mode", proxyOperatingModeReverseMap[p.OperatingMode])
	d.Set("description", p.Description)
	d.Set("allowed_addresses", p.AllowedAddresses)
	d.Set("address", p.Address)
	d.Set("port", p.Port)
	d.Set("tls_connect", proxyTLSModeReverseMap[p.TLSConnect])
	d.Set("tls_accept", proxyTLSModeReverseMap[p.TLSAccept])
	d.Set("tls_issuer", p.TLSIssuer)
	d.Set("tls_subject", p.TLSSubject)
	// tls_psk and tls_psk_identity are NOT returned by the API - do not overwrite state
	d.Set("custom_timeouts", proxyCustomTimeoutsReverseMap[p.CustomTimeouts])
	d.Set("timeout_zabbix_agent", p.TimeoutZabbixAgent)
	d.Set("timeout_simple_check", p.TimeoutSimpleCheck)
	d.Set("timeout_snmp_agent", p.TimeoutSNMPAgent)
	d.Set("timeout_external_check", p.TimeoutExternalCheck)
	d.Set("timeout_db_monitor", p.TimeoutDBMonitor)
	d.Set("timeout_http_agent", p.TimeoutHTTPAgent)
	d.Set("timeout_ssh_agent", p.TimeoutSSHAgent)
	d.Set("timeout_telnet_agent", p.TimeoutTelnetAgent)
	d.Set("timeout_script", p.TimeoutScript)
	d.Set("timeout_browser", p.TimeoutBrowser)
	d.Set("proxy_groupid", p.ProxyGroupID)
	d.Set("local_address", p.LocalAddress)
	d.Set("local_port", p.LocalPort)

	return nil
}

func resourceManagedProxyUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := proxyBuildObject(d)
	item.ProxyID = d.Id()

	err := api.ManagedProxiesUpdate(zabbix.ManagedProxies{item})
	if err != nil {
		return err
	}

	return resourceManagedProxyRead(d, m)
}

func resourceManagedProxyDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.ManagedProxiesDeleteByIds([]string{d.Id()})
}
