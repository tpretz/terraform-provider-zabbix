package provider

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// PROXY_MODE_LOOKUP maps the Terraform "operating_mode" value onto the
// version-neutral mode understood by the API client. Zabbix spells the same
// concept as "status" 5/6 before 7.0 and "operating_mode" 0/1 from 7.0; the
// Terraform name follows the current model and does not change with the
// server.
var PROXY_MODE_LOOKUP = map[string]zabbix.ProxyMode{
	"active":  zabbix.ProxyModeActive,
	"passive": zabbix.ProxyModePassive,
}
var PROXY_MODE_LOOKUP_REV = map[zabbix.ProxyMode]string{}
var PROXY_MODE_LOOKUP_ARR = []string{}

// PROXY_TLS_LOOKUP maps the Terraform tls_connect / tls_accept values onto the
// API's numeric encryption codes.
var PROXY_TLS_LOOKUP = map[string]zabbix.TLSMode{
	"unencrypted": zabbix.TLSUnencrypted,
	"psk":         zabbix.TLSPSKMode,
	"cert":        zabbix.TLSCertificate,
}
var PROXY_TLS_LOOKUP_REV = map[zabbix.TLSMode]string{}
var PROXY_TLS_LOOKUP_ARR = []string{}

// generate the reverse maps and value lists
var _ = func() bool {
	for k, v := range PROXY_MODE_LOOKUP {
		PROXY_MODE_LOOKUP_REV[v] = k
		PROXY_MODE_LOOKUP_ARR = append(PROXY_MODE_LOOKUP_ARR, k)
	}
	for k, v := range PROXY_TLS_LOOKUP {
		PROXY_TLS_LOOKUP_REV[v] = k
		PROXY_TLS_LOOKUP_ARR = append(PROXY_TLS_LOOKUP_ARR, k)
	}
	return true
}()

// proxySchemaBase describes every attribute a proxy has, in the
// version-neutral naming the provider exposes. resourceProxy and dataProxy
// each take a copy and adjust required/optional/computed.
var proxySchemaBase = map[string]*schema.Schema{
	"name": &schema.Schema{
		Type:         schema.TypeString,
		Description:  "Technical name of the proxy",
		ValidateFunc: validation.StringIsNotWhiteSpace,
	},
	"operating_mode": &schema.Schema{
		Type: schema.TypeString,
		Description: "How the proxy and server connect, one of: " +
			strings.Join(PROXY_MODE_LOOKUP_ARR, ", ") +
			" (Zabbix calls this \"status\" before 7.0)",
		ValidateFunc: validation.StringInSlice(PROXY_MODE_LOOKUP_ARR, false),
	},
	"address": &schema.Schema{
		Type:        schema.TypeString,
		Description: "Address (IP or DNS name) the server connects to, passive proxies only",
	},
	"port": &schema.Schema{
		Type:        schema.TypeString,
		Description: "Port the server connects to, passive proxies only",
	},
	"allowed_addresses": &schema.Schema{
		Type:        schema.TypeString,
		Description: "Comma delimited list of addresses the proxy may connect from, active proxies only",
	},
	"description": &schema.Schema{
		Type:        schema.TypeString,
		Description: "Description of the proxy",
	},
	"tls_connect": &schema.Schema{
		Type: schema.TypeString,
		Description: "Encryption used when the server connects to a passive proxy, one of: " +
			strings.Join(PROXY_TLS_LOOKUP_ARR, ", "),
		ValidateFunc: validation.StringInSlice(PROXY_TLS_LOOKUP_ARR, false),
	},
	"tls_accept": &schema.Schema{
		Type: schema.TypeString,
		Description: "Encryption accepted from an active proxy, one of: " +
			strings.Join(PROXY_TLS_LOOKUP_ARR, ", "),
		ValidateFunc: validation.StringInSlice(PROXY_TLS_LOOKUP_ARR, false),
	},
	"tls_issuer": &schema.Schema{
		Type:        schema.TypeString,
		Description: "Certificate issuer, requires certificate encryption",
	},
	"tls_subject": &schema.Schema{
		Type:        schema.TypeString,
		Description: "Certificate subject, requires certificate encryption",
	},
	"tls_psk_identity": &schema.Schema{
		Type:        schema.TypeString,
		Description: "PSK identity, requires psk encryption. Write only: Zabbix never returns it, so it cannot be read back or imported",
		Sensitive:   true,
	},
	"tls_psk": &schema.Schema{
		Type:        schema.TypeString,
		Description: "Pre-shared key, at least 32 hex digits, requires psk encryption. Write only: Zabbix never returns it, so it cannot be read back or imported",
		Sensitive:   true,
	},
}

// proxyResourceSchema adjusts the base schema for resource usage.
func proxyResourceSchema(m map[string]*schema.Schema) (o map[string]*schema.Schema) {
	o = map[string]*schema.Schema{}
	for k, v := range m {
		s := *v

		switch k {
		case "name":
			s.Required = true
		default:
			s.Optional = true
		}

		o[k] = &s
	}

	o["operating_mode"].Default = PROXY_MODE_LOOKUP_REV[zabbix.ProxyModeActive]
	o["tls_connect"].Default = PROXY_TLS_LOOKUP_REV[zabbix.TLSUnencrypted]
	o["tls_accept"].Default = PROXY_TLS_LOOKUP_REV[zabbix.TLSUnencrypted]

	// Zabbix reports these defaults back for a proxy that has no endpoint of
	// its own, on every version, so defaulting them here keeps an active
	// proxy free of a permanent diff.
	o["address"].Default = zabbix.ProxyDefaultAddress
	o["port"].Default = zabbix.ProxyDefaultPort

	return o
}

// proxyDataSchema adjusts the base schema for data source usage.
func proxyDataSchema(m map[string]*schema.Schema) (o map[string]*schema.Schema) {
	o = map[string]*schema.Schema{}
	for k, v := range m {
		s := *v

		switch k {
		case "name":
			s.Optional = true
			s.Computed = true
		case "tls_psk_identity", "tls_psk":
			// write-only in the API: there is nothing to look up
			continue
		default:
			s.Computed = true
			// nothing to validate on a computed-only attribute, and the SDK
			// rejects a schema that tries
			s.ValidateFunc = nil
		}

		o[k] = &s
	}

	// "host" was this data source's only argument before Zabbix 7.0 renamed
	// the property; it is kept working, and reported alongside "name", so
	// existing configurations do not break.
	o["host"] = &schema.Schema{
		Type:         schema.TypeString,
		Description:  "Technical name of the proxy",
		Optional:     true,
		Computed:     true,
		Deprecated:   "use name instead, matching Zabbix 7.0 and later",
		ValidateFunc: validation.StringIsNotWhiteSpace,
		ExactlyOneOf: []string{"host", "name"},
	}
	o["name"].ExactlyOneOf = []string{"host", "name"}

	return o
}

// resourceProxy terraform proxy resource entrypoint
func resourceProxy() *schema.Resource {
	return &schema.Resource{
		Create: resourceProxyCreate,
		Read:   resourceProxyRead,
		Update: resourceProxyUpdate,
		Delete: resourceProxyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: proxyResourceSchema(proxySchemaBase),
	}
}

// dataProxy terraform proxy data source entrypoint
func dataProxy() *schema.Resource {
	return &schema.Resource{
		Read:   dataProxyRead,
		Schema: proxyDataSchema(proxySchemaBase),
	}
}

// buildProxyObject turns resource data into the API client's neutral proxy
// struct, rejecting the combinations Zabbix would reject anyway but with a
// message that says which mode the offending attribute belongs to.
func buildProxyObject(d *schema.ResourceData) (*zabbix.Proxy, error) {
	mode := PROXY_MODE_LOOKUP[d.Get("operating_mode").(string)]

	proxy := zabbix.Proxy{
		Name:           d.Get("name").(string),
		Mode:           mode,
		Description:    d.Get("description").(string),
		TLSConnect:     PROXY_TLS_LOOKUP[d.Get("tls_connect").(string)],
		TLSAccept:      PROXY_TLS_LOOKUP[d.Get("tls_accept").(string)],
		TLSIssuer:      d.Get("tls_issuer").(string),
		TLSSubject:     d.Get("tls_subject").(string),
		TLSPSKIdentity: d.Get("tls_psk_identity").(string),
		TLSPSK:         d.Get("tls_psk").(string),
	}

	address := d.Get("address").(string)
	port := d.Get("port").(string)
	allowed := d.Get("allowed_addresses").(string)

	if mode == zabbix.ProxyModePassive {
		if allowed != "" {
			return nil, errors.New("allowed_addresses applies to active proxies only")
		}
		proxy.Address = address
		proxy.Port = port
		return &proxy, nil
	}

	// active
	for _, check := range []struct{ attr, value, def string }{
		{"address", address, zabbix.ProxyDefaultAddress},
		{"port", port, zabbix.ProxyDefaultPort},
	} {
		if check.value != "" && check.value != check.def {
			return nil, fmt.Errorf("%s applies to passive proxies only (an active proxy connects to the server, so Zabbix requires the default %q)", check.attr, check.def)
		}
	}
	proxy.AllowedAddresses = allowed

	return &proxy, nil
}

// resourceProxyCreate terraform create handler
func resourceProxyCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	proxy, err := buildProxyObject(d)
	if err != nil {
		return err
	}

	items := zabbix.Proxies{*proxy}

	if err := api.ProxiesCreate(items); err != nil {
		return err
	}

	log.Trace("created proxy: %+v", items[0])

	d.SetId(items[0].ProxyID)

	return resourceProxyRead(d, m)
}

// resourceProxyRead terraform read handler
func resourceProxyRead(d *schema.ResourceData, m interface{}) error {
	log.Debug("Lookup of proxy with id %s", d.Id())

	return proxyRead(d, m, zabbix.Params{"proxyids": d.Id()})
}

// dataProxyRead read handler for the data source
func dataProxyRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	name := d.Get("name").(string)
	if name == "" {
		name = d.Get("host").(string)
	}
	if name == "" {
		return errors.New("no proxy lookup attribute")
	}

	// The property the name lives under is version dependent, and filter
	// parameters go to the API verbatim.
	params := zabbix.Params{
		"filter": map[string]interface{}{
			api.ProxyNameProperty(): name,
		},
	}
	log.Debug("performing data lookup with params: %#v", params)

	if err := proxyRead(d, m, params); err != nil {
		return err
	}

	// keep the deprecated pre-7.0 alias in step with the canonical attribute
	if d.Id() != "" {
		d.Set("host", d.Get("name"))
	}
	return nil
}

// proxyRead common proxy read function
func proxyRead(d *schema.ResourceData, m interface{}, params zabbix.Params) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of proxy with params %#v", params)

	proxies, err := api.ProxiesGet(params)

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
	proxy := proxies[0]

	log.Debug("Got proxy: %+v", proxy)

	d.SetId(proxy.ProxyID)
	d.Set("name", proxy.Name)
	d.Set("operating_mode", PROXY_MODE_LOOKUP_REV[proxy.Mode])
	d.Set("allowed_addresses", proxy.AllowedAddresses)
	d.Set("description", proxy.Description)
	d.Set("tls_connect", PROXY_TLS_LOOKUP_REV[proxy.TLSConnect])
	d.Set("tls_accept", PROXY_TLS_LOOKUP_REV[proxy.TLSAccept])
	d.Set("tls_issuer", proxy.TLSIssuer)
	d.Set("tls_subject", proxy.TLSSubject)

	// An active proxy has no endpoint of its own; report the same defaults the
	// schema applies rather than whatever the server happens to keep in those
	// columns, so that active proxies look identical on every version.
	address, port := proxy.Address, proxy.Port
	if address == "" {
		address = zabbix.ProxyDefaultAddress
	}
	if port == "" {
		port = zabbix.ProxyDefaultPort
	}
	d.Set("address", address)
	d.Set("port", port)

	// tls_psk_identity / tls_psk are write-only in the API and deliberately
	// not set here: there is no value to read, and writing an empty string
	// would destroy what the configuration put in state.
	// The data source's deprecated "host" alias is set by dataProxyRead; it is
	// not part of the resource schema, so it cannot be set from here.

	return nil
}

// resourceProxyUpdate terraform update handler
func resourceProxyUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	proxy, err := buildProxyObject(d)
	if err != nil {
		return err
	}
	proxy.ProxyID = d.Id()

	if err := api.ProxiesUpdate(zabbix.Proxies{*proxy}); err != nil {
		return err
	}

	return resourceProxyRead(d, m)
}

// resourceProxyDelete terraform delete handler
func resourceProxyDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.ProxiesDeleteByIds([]string{d.Id()})
}
