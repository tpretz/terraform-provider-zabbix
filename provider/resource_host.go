package provider

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"

	"github.com/tpretz/go-zabbix-api"
)

var HSNMP_LOOKUP = map[string]zabbix.ItemType{
	"1": zabbix.SNMPv1Agent,
	"2": zabbix.SNMPv2Agent,
	"3": zabbix.SNMPv3Agent,
}
var HSNMP_LOOKUP_REV = map[zabbix.ItemType]string{}
var HSNMP_LOOKUP_ARR = []string{}

var HSNMP_AUTHPROTO = map[string]string{
	"md5": "0",
	"sha": "1",
}
var HSNMP_AUTHPROTO_REV = map[string]string{}
var HSNMP_AUTHPROTO_ARR = []string{}

var HSNMP_PRIVPROTO = map[string]string{
	"des": "0",
	"aes": "1",
}
var HSNMP_PRIVPROTO_REV = map[string]string{}
var HSNMP_PRIVPROTO_ARR = []string{}

var HSNMP_SECLEVEL = map[string]string{
	"noauthnopriv": "0",
	"authnopriv":   "1",
	"authpriv":     "2",
}
var HSNMP_SECLEVEL_REV = map[string]string{}
var HSNMP_SECLEVEL_ARR = []string{}

// generate the above structures
var _ = func() bool {
	for k, v := range HSNMP_LOOKUP {
		HSNMP_LOOKUP_REV[v] = k
		HSNMP_LOOKUP_ARR = append(HSNMP_LOOKUP_ARR, k)
	}
	for k, v := range HSNMP_AUTHPROTO {
		HSNMP_AUTHPROTO_REV[v] = k
		HSNMP_AUTHPROTO_ARR = append(HSNMP_AUTHPROTO_ARR, k)
	}
	for k, v := range HSNMP_PRIVPROTO {
		HSNMP_PRIVPROTO_REV[v] = k
		HSNMP_PRIVPROTO_ARR = append(HSNMP_PRIVPROTO_ARR, k)
	}
	for k, v := range HSNMP_SECLEVEL {
		HSNMP_SECLEVEL_REV[v] = k
		HSNMP_SECLEVEL_ARR = append(HSNMP_SECLEVEL_ARR, k)
	}
	return false
}()

// interface type conversions
var HOST_IFACE_TYPES = map[string]zabbix.InterfaceType{
	"agent": zabbix.Agent,
	"snmp":  zabbix.SNMP,
	"ipmi":  zabbix.IPMI,
	"jmx":   zabbix.JMX,
}
var HOST_IFACE_TYPES_REV = map[zabbix.InterfaceType]string{
	zabbix.Agent: "agent",
	zabbix.SNMP:  "snmp",
	zabbix.IPMI:  "ipmi",
	zabbix.JMX:   "jmx",
}
var HOST_IFACE_PORTS = map[string]int{
	"agent": 10050,
	"snmp":  161,
	"ipmi":  623,
	"jmx":   8686,
}

// hostSchemaBase base host schema
var hostSchemaBase = map[string]*schema.Schema{
	"name": &schema.Schema{
		Type:        schema.TypeString,
		Required:    false,
		Optional:    true,
		Computed:    true,
		Description: "Zabbix host displayname, defaults to the value of \"host\"",
	},
	"host": &schema.Schema{
		Type:         schema.TypeString,
		Description:  "FQDN of host",
		ValidateFunc: validation.StringIsNotWhiteSpace,
	},
	"proxyid": &schema.Schema{
		Type:        schema.TypeString,
		Description: "ID of proxy to monitor this host",
	},
	"enabled": &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     true,
		Description: "Enable host for monitoring",
	},
	"interface": &schema.Schema{
		Type:        schema.TypeList,
		Description: "Host interfaces",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": &schema.Schema{
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Interface ID (internally generated)",
				},
				"dns": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Interface DNS name",
				},
				"ip": &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: validation.IsIPAddress,
					Description:  "Interface IP address",
				},
				"main": &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     true,
					Description: "Primary interface of this type",
				},
				"port": &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					Computed:     true,
					ValidateFunc: validation.IntBetween(0, 65535),
					Description:  "Destination Port",
				},
				"type": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  "agent",
					ValidateFunc: validation.StringInSlice([]string{
						"agent",
						"snmp",
						"ipmi",
						"jmx",
					}, false),
					Description: "Interface type",
				},
				"snmp_version": &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Default:      "2",
					Description:  "SNMP Version, one of: " + strings.Join(HSNMP_LOOKUP_ARR, ", "),
					ValidateFunc: validation.StringInSlice(HSNMP_LOOKUP_ARR, false),
				},
				"snmp_bulk": &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     true,
					Description: "SNMP Bulk",
				},
				"snmp_community": &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "HSNMP Community (v1/v2 only)",
					ValidateFunc: validation.StringIsNotWhiteSpace,
					Default:      "{$SNMP_COMMUNITY}",
				},
				"snmp3_authpassphrase": &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "Authentication Passphrase (v3 only)",
					ValidateFunc: validation.StringIsNotWhiteSpace,
					Default:      "{$SNMP3_AUTHPASSPHRASE}",
				},
				"snmp3_authprotocol": &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "Authentication Protocol (v3 only), one of: " + strings.Join(HSNMP_AUTHPROTO_ARR, ", "),
					ValidateFunc: validation.StringInSlice(HSNMP_AUTHPROTO_ARR, false),
					Default:      "sha",
				},
				"snmp3_contextname": &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "Context Name (v3 only)",
					ValidateFunc: validation.StringIsNotWhiteSpace,
					Default:      "{$SNMP3_CONTEXTNAME}",
				},
				"snmp3_privpassphrase": &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "Priv Passphrase (v3 only)",
					ValidateFunc: validation.StringIsNotWhiteSpace,
					Default:      "{$SNMP3_PRIVPASSPHRASE}",
				},
				"snmp3_privprotocol": &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "Priv Protocol (v3 only), one of: " + strings.Join(HSNMP_PRIVPROTO_ARR, ", "),
					ValidateFunc: validation.StringInSlice(HSNMP_PRIVPROTO_ARR, false),
					Default:      "aes",
				},
				"snmp3_securitylevel": &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "Security Level (v3 only), one of: " + strings.Join(HSNMP_SECLEVEL_ARR, ", "),
					ValidateFunc: validation.StringInSlice(HSNMP_SECLEVEL_ARR, false),
					Default:      "authpriv",
				},
				"snmp3_securityname": &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "Security Name (v3 only)",
					ValidateFunc: validation.StringIsNotWhiteSpace,
					Default:      "{$SNMP3_SECURITYNAME}",
				},
			},
		},
	},
	"groups": &schema.Schema{
		Type:        schema.TypeSet,
		Description: "Hostgroup IDs to associate this host with",
		Elem: &schema.Schema{
			Type:         schema.TypeString,
			ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9]+$"), "must be a numeric string"),
		},
	},
	"templates": &schema.Schema{
		Type:        schema.TypeSet,
		Description: "Template IDs to attach to this host",
		Elem: &schema.Schema{
			Type:         schema.TypeString,
			ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9]+$"), "must be a numeric string"),
		},
	},
	"macro": macroListSchema,
}

// resourceHost terraform host resource entrypoint
func resourceHost() *schema.Resource {
	return &schema.Resource{
		Create: resourceHostCreate,
		Read:   resourceHostRead,
		Update: resourceHostUpdate,
		Delete: resourceHostDelete,
		Schema: hostResourceSchema(hostSchemaBase),
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

// dataHost terraform host resource entrypoint
func dataHost() *schema.Resource {
	return &schema.Resource{
		Read:   dataHostRead,
		Schema: hostDataSchema(hostSchemaBase),
	}
}

// hostResourceSchema adjust a base schema for resource usage
func hostResourceSchema(m map[string]*schema.Schema) (o map[string]*schema.Schema) {
	o = map[string]*schema.Schema{}
	for k, v := range m {
		schema := *v

		// required
		switch k {
		case "host", "interface", "groups":
			schema.Required = true
		case "templates", "proxyid":
			schema.Optional = true
		}

		o[k] = &schema
	}

	o["proxyid"].ValidateFunc = validation.StringIsNotWhiteSpace
	o["proxyid"].Default = "0"
	return o
}

// hostDataSchema adjust a base schema for data usage
func hostDataSchema(m map[string]*schema.Schema) (o map[string]*schema.Schema) {
	o = map[string]*schema.Schema{}
	for k, v := range m {
		schema := *v

		// computed
		switch k {
		case "host", "templates":
			schema.Optional = true
			fallthrough
		case "interface", "groups", "macro", "proxyid":
			schema.Computed = true
		}

		o[k] = &schema
	}

	// lookup vars
	o["hostid"] = &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	}

	return o
}

// hostGenerateInterfaces generate interface object array
func hostGenerateInterfaces(d *schema.ResourceData, m interface{}) (interfaces zabbix.HostInterfaces, err error) {
	api := m.(*zabbix.API)
	interfaceCount := d.Get("interface.#").(int)
	interfaces = make(zabbix.HostInterfaces, interfaceCount)

	for i := 0; i < interfaceCount; i++ {
		prefix := fmt.Sprintf("interface.%d.", i)
		typeId := HOST_IFACE_TYPES[d.Get(prefix+"type").(string)]

		interfaces[i] = zabbix.HostInterface{
			IP:    d.Get(prefix + "ip").(string),
			DNS:   d.Get(prefix + "dns").(string),
			Main:  "0",
			Type:  typeId,
			UseIP: "0",
		}
		if interfaces[i].IP == "" && interfaces[i].DNS == "" {
			err = errors.New("interface requires either an IP or DNS entry")
			return
		}

		if interfaces[i].IP != "" {
			interfaces[i].UseIP = "1"
		}

		if d.Get(prefix + "main").(bool) {
			interfaces[i].Main = "1"
		}

		// if no port set, set the default for the type
		if v, ok := d.GetOk(prefix + "port"); ok {
			interfaces[i].Port = strconv.FormatInt(int64(v.(int)), 10)
		} else {
			v := HOST_IFACE_PORTS[d.Get(prefix+"type").(string)]
			d.Set(prefix+"port", v)
			interfaces[i].Port = strconv.FormatInt(int64(v), 10)
		}

		// if we have an id (i.e an update)
		if str := d.Get(prefix + "id").(string); str != "" {
			interfaces[i].InterfaceID = str
		}

		log.Debug("interface config abc: %+v", api.Config)
		// version 5 and snmp
		if api.Config.Version >= 5 && typeId == zabbix.SNMP {
			details := zabbix.HostInterfaceDetail{}
			details.Version = d.Get(prefix + "snmp_version").(string)
			details.Bulk = "0"
			if d.Get(prefix + "snmp_bulk").(bool) {
				details.Bulk = "1"
			}

			// only pull relevent params
			//if details.Version == "3" {
			details.SecurityName = d.Get(prefix + "snmp3_securityname").(string)
			details.SecurityLevel = HSNMP_SECLEVEL[d.Get(prefix+"snmp3_securitylevel").(string)]
			details.AuthPassphrase = d.Get(prefix + "snmp3_authpassphrase").(string)
			details.PrivPassphrase = d.Get(prefix + "snmp3_privpassphrase").(string)
			details.AuthProtocol = HSNMP_AUTHPROTO[d.Get(prefix+"snmp3_authprotocol").(string)]
			details.PrivProtocol = HSNMP_PRIVPROTO[d.Get(prefix+"snmp3_privprotocol").(string)]
			details.ContextName = d.Get(prefix + "snmp3_contextname").(string)
			//} else {
			details.Community = d.Get(prefix + "snmp_community").(string)
			//}
			//interfaces[i].Details = zabbix.HostInterfaceDetails{details}
			interfaces[i].Details = &details
		}
	}

	return
}

// buildHostObject create host struct
func buildHostObject(d *schema.ResourceData, m interface{}) (*zabbix.Host, error) {
	item := zabbix.Host{
		Host:    d.Get("host").(string),
		Name:    d.Get("name").(string),
		ProxyID: d.Get("proxyid").(string),
		Status:  0,
	}

	if !d.Get("enabled").(bool) {
		item.Status = 1
	}

	item.GroupIds = buildHostGroupIds(d.Get("groups").(*schema.Set))
	item.TemplateIDs = buildTemplateIds(d.Get("templates").(*schema.Set))

	interfaces, err := hostGenerateInterfaces(d, m)

	if err != nil {
		return nil, err
	}

	item.Interfaces = interfaces
	item.UserMacros = macroGenerate(d)

	log.Trace("build host object: %#v", item)

	return &item, nil
}

// resourceHostCreate terraform create handler
func resourceHostCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item, err := buildHostObject(d, m)

	if err != nil {
		return err
	}

	items := []zabbix.Host{*item}

	err = api.HostsCreate(items)

	if err != nil {
		return err
	}

	log.Trace("created host: %+v", items[0])

	d.SetId(items[0].HostID)

	return resourceHostRead(d, m)
}

// dataHostRead read handler for data resource
func dataHostRead(d *schema.ResourceData, m interface{}) error {
	params := zabbix.Params{
		"selectInterfaces":      "extend",
		"selectParentTemplates": "extend",
		"selectGroups":          "extend",
		"selectMacros":          "extend",
		"filter":                map[string]interface{}{},
	}

	lookups := []string{"host", "hostid", "name"}
	for _, k := range lookups {
		if v, ok := d.GetOk(k); ok {
			params["filter"].(map[string]interface{})[k] = v
		}
	}

	if len(params["filter"].(map[string]interface{})) < 1 {
		return errors.New("no host lookup attribute")
	}
	log.Debug("performing data lookup with params: %#v", params)

	return hostRead(d, m, params)
}

// resourceHostRead read handler for resource
func resourceHostRead(d *schema.ResourceData, m interface{}) error {
	log.Debug("Lookup of hostgroup with id %s", d.Id())

	return hostRead(d, m, zabbix.Params{
		"selectInterfaces":      "extend",
		"selectParentTemplates": "extend",
		"selectGroups":          "extend",
		"selectMacros":          "extend",
		"hostids":               d.Id(),
	})
}

// hostRead common host read function
func hostRead(d *schema.ResourceData, m interface{}, params zabbix.Params) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of host with params %#v", params)

	hosts, err := api.HostsGet(params)

	if err != nil {
		return err
	}

	if len(hosts) < 1 {
		d.SetId("")
		return nil
	}
	if len(hosts) > 1 {
		return errors.New("multiple hosts found")
	}
	host := hosts[0]

	log.Debug("Got host: %+v", host)

	d.SetId(host.HostID)
	d.Set("name", host.Name)
	d.Set("host", host.Host)
	d.Set("proxyid", host.ProxyID)
	d.Set("enabled", host.Status == 0)

	d.Set("interface", flattenHostInterfaces(host, d, m))
	d.Set("templates", flattenTemplateIds(host.ParentTemplateIDs))
	d.Set("groups", flattenHostGroupIds(host.GroupIds))
	d.Set("macro", flattenMacros(host.UserMacros))

	return nil
}

// flattenHostInterfaces convert API response into terraform structs
func flattenHostInterfaces(host zabbix.Host, d *schema.ResourceData, m interface{}) []interface{} {
	api := m.(*zabbix.API)
	val := make([]interface{}, len(host.Interfaces))
	for i := 0; i < len(host.Interfaces); i++ {
		port, _ := strconv.ParseInt(host.Interfaces[i].Port, 10, 64)
		params := map[string]interface{}{
			"id":   host.Interfaces[i].InterfaceID,
			"ip":   host.Interfaces[i].IP,
			"dns":  host.Interfaces[i].DNS,
			"main": host.Interfaces[i].Main == "1",
			"port": port,
			"type": HOST_IFACE_TYPES_REV[host.Interfaces[i].Type],
		}

		// need to handle detail
		details := host.Interfaces[i].Details
		log.Debug("got details: %+v", details)
		if api.Config.Version >= 5 && params["type"] == "snmp" && details != nil {
			log.Debug("interface new logic")
			d := details
			params["snmp_version"] = d.Version
			params["snmp_bulk"] = d.Bulk == "1"

			params["snmp_community"] = d.Community

			params["snmp_securityname"] = d.SecurityName
			params["snmp_securitylevel"] = HSNMP_SECLEVEL_REV[d.SecurityLevel]
			params["snmp_authpassphrase"] = d.AuthPassphrase
			params["snmp_privpassphrase"] = d.PrivPassphrase
			params["snmp_authprotocol"] = HSNMP_AUTHPROTO_REV[d.AuthProtocol]
			params["snmp_privprotocol"] = HSNMP_PRIVPROTO_REV[d.PrivProtocol]
			params["snmp_contextname"] = d.ContextName
		} else { // echo back current values, keep state happy
			log.Debug("interface old logic")
			arr := []string{
				"snmp_version",
				"snmp_community",
				"snmp3_authpassphrase",
				"snmp3_authprotocol",
				"snmp3_contextname",
				"snmp3_privpassphrase",
				"snmp3_privprotocol",
				"snmp3_securitylevel",
				"snmp3_securityname",
				"snmp_bulk",
			}

			for _, v := range arr {
				params[v] = hostSchemaBase["interface"].Elem.(*schema.Resource).Schema[v].Default
			}
		}
		log.Debug("Got host interface: %+v", params)
		val[i] = params
	}
	return val
}

// resourceHostUpdate terraform update resource handler
func resourceHostUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item, err := buildHostObject(d, m)

	if err != nil {
		return err
	}

	// templates may need a bit extra effort
	if d.HasChange("templates") {
		old, new := d.GetChange("templates")
		diff := old.(*schema.Set).Difference(new.(*schema.Set))

		// removals, we need to unlink and clear
		if diff.Len() > 0 {
			item.TemplateIDsClear = buildTemplateIds(diff)
		}
	}

	item.HostID = d.Id()

	items := []zabbix.Host{*item}

	err = api.HostsUpdate(items)

	if err != nil {
		return err
	}

	return resourceHostRead(d, m)
}

// resourceHostDelete terraform delete resource handler
func resourceHostDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.HostsDeleteByIds([]string{d.Id()})
}
