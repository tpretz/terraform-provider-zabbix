package provider

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/tpretz/go-zabbix-api"
)

var HSNMP_LOOKUP = map[string]zabbix.ItemType{
	"1": zabbix.SNMPv1Agent,
	"2": zabbix.SNMPv2Agent,
	"3": zabbix.SNMPv3Agent,
}
var HSNMP_LOOKUP_REV = map[zabbix.ItemType]string{}
var HSNMP_LOOKUP_ARR = []string{}

var HINV_LOOKUP = map[string]zabbix.InventoryMode{
	"disabled":  zabbix.InventoryDisabled,
	"manual":    zabbix.InventoryManual,
	"automatic": zabbix.InventoryAutomatic,
}
var HINV_LOOKUP_REV = map[zabbix.InventoryMode]string{}
var HINV_LOOKUP_ARR = []string{}

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

var INVENTORY_KEYS = []string{
	"alias",
	"asset_tag",
	"chassis",
	"contact",
	"contract_number",
	"date_hw_decomm",
	"date_hw_expiry",
	"date_hw_install",
	"date_hw_purchase",
	"deployment_status",
	"hardware",
	"hardware_full",
	"host_netmask",
	"host_networks",
	"host_router",
	"hw_arch",
	"installer_name",
	"location",
	"location_lat",
	"location_lon",
	"macaddress_a",
	"macaddress_b",
	"model",
	"name",
	"notes",
	"oob_ip",
	"oob_netmask",
	"oob_router",
	"os",
	"os_full",
	"os_short",
	"poc_1_cell",
	"poc_1_email",
	"poc_1_name",
	"poc_1_notes",
	"poc_1_phone_a",
	"poc_1_phone_b",
	"poc_1_screen",
	"poc_2_cell",
	"poc_2_email",
	"poc_2_name",
	"poc_2_notes",
	"poc_2_phone_a",
	"poc_2_phone_b",
	"poc_2_screen",
	"serialno_a",
	"serialno_b",
	"site_address_a",
	"site_address_b",
	"site_address_c",
	"site_city",
	"site_country",
	"site_notes",
	"site_rack",
	"site_state",
	"site_zip",
	"software",
	"software_app_a",
	"software_app_b",
	"software_app_c",
	"software_app_d",
	"software_app_e",
	"software_full",
	"tag",
	"type",
	"type_full",
	"url_a",
	"url_b",
	"url_c",
	"vendor",
}

var inventorySchema = &schema.Schema{
	Type: schema.TypeList,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{},
	},
}

// generate the above structures
var _ = func() bool {
	for k, v := range HSNMP_LOOKUP {
		HSNMP_LOOKUP_REV[v] = k
		HSNMP_LOOKUP_ARR = append(HSNMP_LOOKUP_ARR, k)
	}
	for k, v := range HINV_LOOKUP {
		HINV_LOOKUP_REV[v] = k
		HINV_LOOKUP_ARR = append(HINV_LOOKUP_ARR, k)
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
	for _, v := range INVENTORY_KEYS {
		inventorySchema.Elem.(*schema.Resource).Schema[v] = &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Inventory " + v,
		}
	}
	return false
}()

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
	"inventory": inventorySchema,
	"inventory_mode": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Default:      "disabled",
		Description:  "Inventory Mode, one of: " + strings.Join(HINV_LOOKUP_ARR, ", "),
		ValidateFunc: validation.StringInSlice(HINV_LOOKUP_ARR, false),
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
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Interface IP address",
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
	"tag": &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"key": &schema.Schema{
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validation.StringIsNotWhiteSpace,
					Description:  "Tag Key",
				},
				"value": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Tag Value",
				},
			},
		},
	},
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
		case "templates", "proxyid", "inventory":
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
		case "interface", "groups", "macro", "proxyid", "inventory":
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
			//d.Set(prefix+"port", v)
			interfaces[i].Port = strconv.FormatInt(int64(v), 10)
		}

		// if we have an id (i.e an update)
		if str := d.Get(prefix + "id").(string); str != "" {
			interfaces[i].InterfaceID = str
		}

		log.Debug("interface config abc: %+v", api.Config)
		// version 5 and snmp
		if api.Config.Version >= 50000 && typeId == zabbix.SNMP {
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

func hostGenerateInventory(d *schema.ResourceData) (zabbix.Inventory, error) {

	inventoryCount := d.Get("inventory.#").(int)
	if inventoryCount > 1 {
		return nil, errors.New("must be 0 or 1 instances of inventory block")
	}
	if inventoryCount < 1 {
		return nil, nil
	}

	inventory := zabbix.Inventory{}
	for i := 0; i < inventoryCount; i++ {
		prefix := fmt.Sprintf("inventory.%d.", i)

		for _, k := range INVENTORY_KEYS {
			if val, ok := d.GetOk(prefix + k); ok {
				inventory[k] = val.(string)
			}
		}
	}

	return inventory, nil
}

// buildHostObject create host struct
func buildHostObject(d *schema.ResourceData, m interface{}) (*zabbix.Host, error) {
	item := zabbix.Host{
		Host:          d.Get("host").(string),
		Name:          d.Get("name").(string),
		ProxyID:       d.Get("proxyid").(string),
		InventoryMode: HINV_LOOKUP[d.Get("inventory_mode").(string)],
		Status:        0,
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
	item.Tags = tagGenerate(d)
	item.Inventory, err = hostGenerateInventory(d)

	if err != nil {
		return nil, err
	}

	// adjust inventory mode if block is included
	if item.Inventory != nil && item.InventoryMode == zabbix.InventoryDisabled {
		return nil, errors.New("inventory_mode must be enabled for inventory to be used")
	}

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
		"selectTags":            "extend",
		"selectInventory":       "extend",
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
		"selectTags":            "extend",
		"selectInventory":       "extend",
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
	d.Set("inventory_mode", HINV_LOOKUP_REV[host.InventoryMode])

	d.Set("interface", flattenHostInterfaces(host, d, m))
	d.Set("templates", flattenTemplateIds(host.ParentTemplateIDs))
	d.Set("inventory", flattenInventory(host))
	d.Set("groups", flattenHostGroupIds(host.GroupIds))
	d.Set("macro", flattenMacros(host.UserMacros))
	d.Set("tag", flattenTags(host.Tags))

	return nil
}

// flattenInventory converts API response into terraform structs
func flattenInventory(host zabbix.Host) []interface{} {
	if host.Inventory == nil {
		return []interface{}{}
	}
	obj := map[string]interface{}{}
	for k, v := range host.Inventory {
		// handle legacy zabbix v4 values that may be in here
		if k == "hostid" || k == "inventory_mode" {
			continue
		}
		obj[k] = v
	}
	if len(obj) == 0 {
		return []interface{}{}
	}
	return []interface{}{obj}
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

		// Set defaults, as these may or may not be bounced back
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

		// need to handle detail
		details := host.Interfaces[i].Details
		log.Debug("got details: %+v", details)
		if api.Config.Version >= 50000 && params["type"] == "snmp" && details != nil {
			log.Debug("interface new logic")
			params["snmp_version"] = details.Version
			params["snmp_bulk"] = details.Bulk == "1"

			if params["snmp_version"] != "3" {
				params["snmp_community"] = details.Community
			} else {
				params["snmp3_securityname"] = details.SecurityName
				params["snmp3_securitylevel"] = HSNMP_SECLEVEL_REV[details.SecurityLevel]
				params["snmp3_authpassphrase"] = details.AuthPassphrase
				params["snmp3_privpassphrase"] = details.PrivPassphrase
				params["snmp3_authprotocol"] = HSNMP_AUTHPROTO_REV[details.AuthProtocol]
				params["snmp3_privprotocol"] = HSNMP_PRIVPROTO_REV[details.PrivProtocol]
				params["snmp3_contextname"] = details.ContextName
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

	// if we had tags, and now we don't, send empty list
	if d.HasChange("tag") {
		_, new := d.GetChange("tag")
		newS := new.(*schema.Set)

		// change from something, to nothing, need to send "nothing"
		fmt.Printf("tag change")
		if newS.Len() == 0 {
			fmt.Print("setting")
			item.Tags = zabbix.Tags{}
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
