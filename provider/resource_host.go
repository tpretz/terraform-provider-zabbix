package provider

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// HSNMP_LOOKUP_ARR lists the valid values of the host interface "snmp_version"
// field (1/2/3), which Zabbix's interface "details" object still takes as of
// 6.0+ even though the item-level v1/v2/v3 type split was collapsed at 5.0.
var HSNMP_LOOKUP_ARR = []string{"1", "2", "3"}

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

// HOST_TLS_LOOKUP maps the Terraform tls_connect / tls_accept values onto the
// API's numeric encryption codes. Deliberately the same vocabulary as
// zabbix_proxy's tls_connect / tls_accept.
var HOST_TLS_LOOKUP = map[string]zabbix.TLSMode{
	"unencrypted": zabbix.TLSUnencrypted,
	"psk":         zabbix.TLSPSKMode,
	"cert":        zabbix.TLSCertificate,
}
var HOST_TLS_LOOKUP_REV = map[zabbix.TLSMode]string{}
var HOST_TLS_LOOKUP_ARR = []string{}

// HOST_IPMI_AUTHTYPE maps the Terraform ipmi_authtype values onto the API's
// numeric codes. "default" (-1) is Zabbix's own default.
var HOST_IPMI_AUTHTYPE = map[string]string{
	"default":  "-1",
	"none":     "0",
	"md2":      "1",
	"md5":      "2",
	"straight": "4",
	"oem":      "5",
	"rmcp+":    "6",
}
var HOST_IPMI_AUTHTYPE_REV = map[string]string{}
var HOST_IPMI_AUTHTYPE_ARR = []string{}

// HOST_IPMI_PRIVILEGE maps the Terraform ipmi_privilege values onto the API's
// numeric codes. "user" (2) is Zabbix's own default.
var HOST_IPMI_PRIVILEGE = map[string]string{
	"callback": "1",
	"user":     "2",
	"operator": "3",
	"admin":    "4",
	"oem":      "5",
}
var HOST_IPMI_PRIVILEGE_REV = map[string]string{}
var HOST_IPMI_PRIVILEGE_ARR = []string{}

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

// HOST_IFACE_TYPES_ARR exists for the description only. Unlike every other
// enum, interface "type" validates against an inline []string rather than
// against this list -- see TestSchemaHostInterfaceTypeMatchesLookup, which
// exists to catch the two drifting apart and would be trivially true if the
// validator read from here.
var HOST_IFACE_TYPES_ARR = []string{}
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
	Description: "Host inventory fields. A single block, not a collection — the list type " +
		"is how a lone optional nested block is expressed in SDKv2. Requires " +
		"`inventory_mode` to be \"manual\" or \"automatic\"; under \"automatic\" Zabbix " +
		"overwrites any field populated by an item, so managing those here will fight the " +
		"server.",
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{},
	},
}

// generate the above structures
var _ = func() bool {
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
	for k, v := range HOST_TLS_LOOKUP {
		HOST_TLS_LOOKUP_REV[v] = k
		HOST_TLS_LOOKUP_ARR = append(HOST_TLS_LOOKUP_ARR, k)
	}
	for k, v := range HOST_IPMI_AUTHTYPE {
		HOST_IPMI_AUTHTYPE_REV[v] = k
		HOST_IPMI_AUTHTYPE_ARR = append(HOST_IPMI_AUTHTYPE_ARR, k)
	}
	for k, v := range HOST_IPMI_PRIVILEGE {
		HOST_IPMI_PRIVILEGE_REV[v] = k
		HOST_IPMI_PRIVILEGE_ARR = append(HOST_IPMI_PRIVILEGE_ARR, k)
	}
	for k := range HOST_IFACE_TYPES {
		HOST_IFACE_TYPES_ARR = append(HOST_IFACE_TYPES_ARR, k)
	}
	for _, v := range INVENTORY_KEYS {
		inventorySchema.Elem.(*schema.Resource).Schema[v] = &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Zabbix host inventory field `" + v + "`",
		}
	}
	// map iteration order is random; sort so that the generated documentation
	// and validation messages are stable between builds
	for _, a := range [][]string{
		HINV_LOOKUP_ARR, HSNMP_AUTHPROTO_ARR, HSNMP_PRIVPROTO_ARR, HSNMP_SECLEVEL_ARR,
		HOST_TLS_LOOKUP_ARR, HOST_IPMI_AUTHTYPE_ARR, HOST_IPMI_PRIVILEGE_ARR,
		HOST_IFACE_TYPES_ARR,
	} {
		sort.Strings(a)
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
		Description:  "Technical host name, unique across the Zabbix server. Usually the FQDN",
		ValidateFunc: validation.StringIsNotWhiteSpace,
	},
	"proxyid": &schema.Schema{
		Type:        schema.TypeString,
		Description: "ID of the proxy monitoring this host, or \"0\" for the Zabbix server itself. Sets `monitored_by` accordingly on Zabbix 7.0 and later",
	},
	"enabled": &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     true,
		Description: "Whether the host is monitored. A disabled host keeps its configuration but is not polled",
	},
	"inventory": inventorySchema,
	"inventory_mode": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Default:      "disabled",
		Description:  "How the host inventory is populated, one of: " + strings.Join(HINV_LOOKUP_ARR, ", ") + ". The `inventory` block requires \"manual\" or \"automatic\"",
		ValidateFunc: validation.StringInSlice(HINV_LOOKUP_ARR, false),
	},
	"interface": &schema.Schema{
		Type:        schema.TypeSet,
		Set:         hostInterfaceHash,
		Description: "Host interfaces (unordered). A set, not a list: `interface[0]` does not parse — use `one(...)` or a `for` expression to pick one out",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": &schema.Schema{
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Interface ID, assigned by Zabbix",
				},
				"dns": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "DNS name Zabbix connects to. Used when `ip` is empty",
				},
				"ip": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "IP address Zabbix connects to. Takes precedence over `dns` when both are set",
				},
				"main": &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     true,
					Description: "Whether this is the default interface for its type. Exactly one interface of each type must be the primary",
				},
				"port": &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					Computed:     true,
					ValidateFunc: validation.IntBetween(0, 65535),
					Description:  "Port Zabbix connects to. Defaults to the standard port for the interface type",
				},
				"type": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  "agent",
					// deliberately an inline list rather than
					// HOST_IFACE_TYPES_ARR: see the comment on that variable
					ValidateFunc: validation.StringInSlice([]string{
						"agent",
						"snmp",
						"ipmi",
						"jmx",
					}, false),
					Description: "Interface type, one of: " + strings.Join(HOST_IFACE_TYPES_ARR, ", ") +
						". Determines the default port (agent 10050, snmp 161, ipmi 623, jmx 8686) " +
						"and which of the snmp* attributes below apply",
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
					Description: "Use SNMP bulk requests (GETBULK) where the version supports them",
				},
				"snmp_community": &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "SNMP community string (v1/v2 only). Usually a user macro reference such as `{$SNMP_COMMUNITY}`",
					ValidateFunc: validation.StringIsNotWhiteSpace,
					Default:      "{$SNMP_COMMUNITY}",
				},
				"snmp3_authpassphrase": &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "SNMPv3 authentication passphrase (v3 only)",
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
					Description:  "SNMPv3 context name (v3 only)",
					ValidateFunc: validation.StringIsNotWhiteSpace,
					Default:      "{$SNMP3_CONTEXTNAME}",
				},
				"snmp3_privpassphrase": &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "SNMPv3 privacy passphrase (v3 only)",
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
					Description:  "SNMPv3 security name (v3 only)",
					ValidateFunc: validation.StringIsNotWhiteSpace,
					Default:      "{$SNMP3_SECURITYNAME}",
				},
			},
		},
	},
	"groups": &schema.Schema{
		Type:        schema.TypeSet,
		Description: "Host group IDs this host belongs to. At least one is required",
		Elem: &schema.Schema{
			Type:         schema.TypeString,
			ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9]+$"), "must be a numeric string"),
		},
	},
	"templates": &schema.Schema{
		Type:        schema.TypeSet,
		Description: "Template IDs linked to this host. Removing a template here unlinks and clears it",
		Elem: &schema.Schema{
			Type:         schema.TypeString,
			ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9]+$"), "must be a numeric string"),
		},
	},
	"ipmi_authtype": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Default:      "default",
		Description:  "IPMI authentication algorithm, one of: " + strings.Join(HOST_IPMI_AUTHTYPE_ARR, ", "),
		ValidateFunc: validation.StringInSlice(HOST_IPMI_AUTHTYPE_ARR, false),
	},
	"ipmi_privilege": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Default:      "user",
		Description:  "IPMI privilege level, one of: " + strings.Join(HOST_IPMI_PRIVILEGE_ARR, ", "),
		ValidateFunc: validation.StringInSlice(HOST_IPMI_PRIVILEGE_ARR, false),
	},
	"ipmi_username": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Username used for IPMI checks against this host",
	},
	"ipmi_password": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "Password used for IPMI checks against this host",
	},
	"tls_connect": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		Default:  "unencrypted",
		Description: "Encryption used for outgoing connections to the host, one of: " +
			strings.Join(HOST_TLS_LOOKUP_ARR, ", "),
		ValidateFunc: validation.StringInSlice(HOST_TLS_LOOKUP_ARR, false),
	},
	"tls_accept": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		Default:  "unencrypted",
		Description: "Encryption accepted for incoming connections from the host, one of: " +
			strings.Join(HOST_TLS_LOOKUP_ARR, ", ") +
			". Zabbix stores this as a bitmask and the frontend allows combinations; only a single mode is expressible here",
		ValidateFunc: validation.StringInSlice(HOST_TLS_LOOKUP_ARR, false),
	},
	"tls_issuer": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Certificate issuer, requires cert encryption",
	},
	"tls_subject": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Certificate subject, requires cert encryption",
	},
	"tls_psk_identity": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "PSK identity, requires psk encryption. Write only: host.get never returns it, so it cannot be read back or imported",
	},
	"tls_psk": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "Pre-shared key, at least 32 hex digits, requires psk encryption. Write only: host.get never returns it, so it cannot be read back or imported",
	},
	"macro": macroSetSchema,
	"tag":   tagSetSchema,
}

// resourceHost terraform host resource entrypoint
func resourceHost() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a Zabbix host: the monitored object that items, triggers and graphs belong to. A host needs at least one group. Interfaces are optional: a host carrying only calculated, dependent, trapper or internal items does not need one.",
		Create:      resourceHostCreate,
		Read:        resourceHostRead,
		Update:      resourceHostUpdate,
		Delete:      resourceHostDelete,
		Schema:      hostResourceSchema(hostSchemaBase),
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		// v0 -> v1: "interface" became a TypeSet. See typeSetStateUpgradeV0.
		SchemaVersion:  1,
		StateUpgraders: hostStateUpgraders(),
	}
}

// dataHost terraform host resource entrypoint
func dataHost() *schema.Resource {
	return &schema.Resource{
		Description: "Looks up an existing Zabbix host by id, technical name or display name.",
		Read:        dataHostRead,
		Schema:      hostDataSchema(hostSchemaBase),
	}
}

// hostResourceSchema adjust a base schema for resource usage
func hostResourceSchema(m map[string]*schema.Schema) (o map[string]*schema.Schema) {
	o = map[string]*schema.Schema{}
	for k, v := range m {
		schema := *v

		// required
		switch k {
		case "host", "groups":
			schema.Required = true
		case "interface":
			// Not required. Zabbix accepts a host with no interfaces at all on
			// every supported version - one carrying only calculated, dependent,
			// trapper or internal items, or existing purely to hold templates,
			// has nothing to attach an interface to.
			schema.Optional = true
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
		case "tls_psk_identity", "tls_psk":
			// write-only in the API: there is nothing to look up
			continue
		case "host", "templates":
			schema.Optional = true
			fallthrough
		case "interface", "groups", "macro", "proxyid", "inventory":
			schema.Computed = true
		case "ipmi_authtype", "ipmi_privilege", "ipmi_username", "ipmi_password",
			"tls_connect", "tls_accept", "tls_issuer", "tls_subject":
			// read-only here; the SDK rejects a default or a validator on a
			// purely computed attribute
			schema.Computed = true
			schema.Optional = false
			schema.Default = nil
			schema.ValidateFunc = nil
		}

		o[k] = &schema
	}

	// lookup vars
	o["hostid"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Look the host up by its Zabbix ID. Give one of `hostid`, `host` or `name`",
	}

	return o
}

// hostInterfacePort resolves the port of an interface element: the value the
// user gave, or the Zabbix default for the interface type. `port` is
// Optional+Computed, so an element read from config carries 0 where the same
// element read back from state carries the real port. Both have to reduce to
// the same number or the two would not hash alike and the interface would look
// replaced on every plan.
func hostInterfacePort(m map[string]interface{}) int {
	var port int
	switch v := m["port"].(type) {
	case int:
		port = v
	case int64:
		port = int(v)
	case float64:
		port = int(v)
	}
	if port != 0 {
		return port
	}
	t, _ := m["type"].(string)
	return HOST_IFACE_PORTS[t]
}

// hostInterfaceHash hashes a host interface over everything the user writes.
//
// A host's interfaces are an unordered collection - host.get returns them in
// whatever order it likes - so they are a TypeSet. Everything but the
// server-assigned `id` goes into the hash, because an attribute outside it can
// never be seen to change (see hashElementExcept); `port` is normalised first,
// so that an omitted port and an explicitly written default port are the same
// interface rather than two.
//
// One consequence needs handling rather than documenting: editing any field of
// an interface reads as a delete plus an add, so the replacement element
// arrives with no id. hostReuseInterfaceIDs puts the id back.
func hostInterfaceHash(v interface{}) int {
	m, ok := v.(map[string]interface{})
	if !ok {
		return 0
	}

	normalised := make(map[string]interface{}, len(m))
	for k, val := range m {
		normalised[k] = val
	}
	normalised["port"] = hostInterfacePort(m)

	return hashElementExcept(normalised, "id")
}

// hostReuseInterfaceIDs hands an interface that Terraform sees as new the id of
// a prior interface of the same type, where one is going spare.
//
// Without it, editing (say) an SNMP community would send Zabbix an interface
// with no interfaceid: the server would create a new interface and delete the
// old one, which it refuses outright once items are bound to that interface.
// An edit has to stay an edit. Matching the leftovers up by type is what the
// old TypeList achieved by position, only less arbitrarily - Zabbix does not
// allow an interface to change type in any case.
func hostReuseInterfaceIDs(d *schema.ResourceData, interfaces zabbix.HostInterfaces) {
	if d.Id() == "" {
		// create: there is no prior state to reuse anything from
		return
	}

	prior, _ := d.GetChange("interface")
	set, ok := prior.(*schema.Set)
	if !ok {
		return
	}

	// ids already carried over by an interface that did not change
	claimed := map[string]bool{}
	for _, iface := range interfaces {
		if iface.InterfaceID != "" {
			claimed[iface.InterfaceID] = true
		}
	}

	spare := map[zabbix.InterfaceType][]string{}
	for _, raw := range set.List() {
		m, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := m["id"].(string)
		if id == "" || claimed[id] {
			continue
		}
		t, _ := m["type"].(string)
		typeId := HOST_IFACE_TYPES[t]
		spare[typeId] = append(spare[typeId], id)
	}

	for i := range interfaces {
		if interfaces[i].InterfaceID != "" {
			continue
		}
		ids := spare[interfaces[i].Type]
		if len(ids) == 0 {
			continue
		}
		log.Debug("reusing interface id %s for edited %s interface", ids[0], HOST_IFACE_TYPES_REV[interfaces[i].Type])
		interfaces[i].InterfaceID = ids[0]
		spare[interfaces[i].Type] = ids[1:]
	}
}

// hostGenerateInterfaces generate interface object array
func hostGenerateInterfaces(d *schema.ResourceData, m interface{}) (interfaces zabbix.HostInterfaces, err error) {
	api := m.(*zabbix.API)
	set := d.Get("interface").(*schema.Set).List()
	interfaces = make(zabbix.HostInterfaces, len(set))

	for i, raw := range set {
		iface := raw.(map[string]interface{})
		typeName, _ := iface["type"].(string)
		typeId := HOST_IFACE_TYPES[typeName]

		interfaces[i] = zabbix.HostInterface{
			IP:    iface["ip"].(string),
			DNS:   iface["dns"].(string),
			Main:  "0",
			Type:  typeId,
			UseIP: "0",
			Port:  strconv.FormatInt(int64(hostInterfacePort(iface)), 10),
		}
		if interfaces[i].IP == "" && interfaces[i].DNS == "" {
			err = errors.New("interface requires either an IP or DNS entry")
			return
		}

		if interfaces[i].IP != "" {
			interfaces[i].UseIP = "1"
		}

		if iface["main"].(bool) {
			interfaces[i].Main = "1"
		}

		// if we have an id (i.e an update)
		if str, _ := iface["id"].(string); str != "" {
			interfaces[i].InterfaceID = str
		}

		log.Debug("interface config abc: %+v", api.Config)
		if typeId == zabbix.SNMP {
			details := zabbix.HostInterfaceDetail{}
			details.Version = iface["snmp_version"].(string)
			details.Bulk = "0"
			if iface["snmp_bulk"].(bool) {
				details.Bulk = "1"
			}

			// only pull relevent params
			//if details.Version == "3" {
			details.SecurityName = iface["snmp3_securityname"].(string)
			details.SecurityLevel = HSNMP_SECLEVEL[iface["snmp3_securitylevel"].(string)]
			details.AuthPassphrase = iface["snmp3_authpassphrase"].(string)
			details.PrivPassphrase = iface["snmp3_privpassphrase"].(string)
			details.AuthProtocol = HSNMP_AUTHPROTO[iface["snmp3_authprotocol"].(string)]
			details.PrivProtocol = HSNMP_PRIVPROTO[iface["snmp3_privprotocol"].(string)]
			details.ContextName = iface["snmp3_contextname"].(string)
			//} else {
			details.Community = iface["snmp_community"].(string)
			//}
			//interfaces[i].Details = zabbix.HostInterfaceDetails{details}
			interfaces[i].Details = &details
		}
	}

	hostReuseInterfaceIDs(d, interfaces)

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

		IPMIAuthType:  HOST_IPMI_AUTHTYPE[d.Get("ipmi_authtype").(string)],
		IPMIPrivilege: HOST_IPMI_PRIVILEGE[d.Get("ipmi_privilege").(string)],
		IPMIUsername:  d.Get("ipmi_username").(string),
		IPMIPassword:  d.Get("ipmi_password").(string),

		TLSConnect:     HOST_TLS_LOOKUP[d.Get("tls_connect").(string)],
		TLSAccept:      HOST_TLS_LOOKUP[d.Get("tls_accept").(string)],
		TLSIssuer:      d.Get("tls_issuer").(string),
		TLSSubject:     d.Get("tls_subject").(string),
		TLSPSKIdentity: d.Get("tls_psk_identity").(string),
		TLSPSK:         d.Get("tls_psk").(string),
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

// hostGroupSelectParam names the host.get parameter that returns host group
// membership. Zabbix 7.2 removed "selectGroups" (deprecated since 6.2) in
// favour of "selectHostGroups"; on 7.2+ the old name is silently ignored, which
// reads back as a host with no groups rather than as an error.
func hostGroupSelectParam(m interface{}) string {
	if m.(*zabbix.API).Config.Version >= zabbix.V72 {
		return "selectHostGroups"
	}
	return "selectGroups"
}

// dataHostRead read handler for data resource
func dataHostRead(d *schema.ResourceData, m interface{}) error {
	params := zabbix.Params{
		"selectInterfaces":      "extend",
		"selectParentTemplates": "extend",
		hostGroupSelectParam(m): "extend",
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

	if err := hostRead(d, m, params); err != nil {
		return err
	}
	return dataSourceFound(d, "host", lookups...)
}

// resourceHostRead read handler for resource
func resourceHostRead(d *schema.ResourceData, m interface{}) error {
	log.Debug("Lookup of hostgroup with id %s", d.Id())

	return hostRead(d, m, zabbix.Params{
		"selectInterfaces":      "extend",
		"selectParentTemplates": "extend",
		hostGroupSelectParam(m): "extend",
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

	d.Set("ipmi_authtype", HOST_IPMI_AUTHTYPE_REV[host.IPMIAuthType])
	d.Set("ipmi_privilege", HOST_IPMI_PRIVILEGE_REV[host.IPMIPrivilege])
	d.Set("ipmi_username", host.IPMIUsername)
	d.Set("ipmi_password", host.IPMIPassword)

	d.Set("tls_connect", HOST_TLS_LOOKUP_REV[host.TLSConnect])
	d.Set("tls_accept", HOST_TLS_LOOKUP_REV[host.TLSAccept])
	d.Set("tls_issuer", host.TLSIssuer)
	d.Set("tls_subject", host.TLSSubject)
	// tls_psk_identity / tls_psk are write-only in the API and deliberately
	// not set here: host.get never returns them, so writing back what it did
	// not send would destroy what the configuration put in state. The data
	// source does not declare them at all.

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
		if params["type"] == "snmp" && details != nil {
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

// existingTemplateIds filters a set of template ids down to those the server
// still knows about.
//
// Terraform will happily destroy a zabbix_template in the same apply that
// removes it from a host's `templates`, and does not always order the host
// update first. Deleting a template unlinks it from its hosts anyway, so
// naming it in `templates_clear` is redundant - but Zabbix 7.0 made unknown
// object ids a hard error, so the redundant entry fails the whole host.update.
func existingTemplateIds(api *zabbix.API, s *schema.Set) (zabbix.TemplateIDs, error) {
	ids := buildTemplateIds(s)
	if len(ids) == 0 {
		return nil, nil
	}

	lookup := make([]string, len(ids))
	for i, t := range ids {
		lookup[i] = t.TemplateID
	}

	found, err := api.TemplatesGet(zabbix.Params{
		"templateids": lookup,
		"output":      []string{"templateid"},
	})
	if err != nil {
		return nil, err
	}

	alive := map[string]bool{}
	for _, t := range found {
		alive[t.TemplateID] = true
	}

	out := make(zabbix.TemplateIDs, 0, len(ids))
	for _, t := range ids {
		if alive[t.TemplateID] {
			out = append(out, t)
		} else {
			log.Debug("template %s no longer exists, dropping from templates_clear", t.TemplateID)
		}
	}
	return out, nil
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
			toClear, err := existingTemplateIds(api, diff)
			if err != nil {
				return err
			}
			if len(toClear) > 0 {
				item.TemplateIDsClear = toClear
			}
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
