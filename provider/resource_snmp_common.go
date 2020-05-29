package provider

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

var SNMP_LOOKUP = map[string]zabbix.ItemType{
	"1": zabbix.SNMPv1Agent,
	"2": zabbix.SNMPv2Agent,
	"3": zabbix.SNMPv3Agent,
}
var SNMP_LOOKUP_REV = map[zabbix.ItemType]string{}
var SNMP_LOOKUP_ARR = []string{}

var SNMP_AUTHPROTO = map[string]string{
	"md5": "0",
	"sha": "1",
}
var SNMP_AUTHPROTO_REV = map[string]string{}
var SNMP_AUTHPROTO_ARR = []string{}

var SNMP_PRIVPROTO = map[string]string{
	"des": "0",
	"aes": "1",
}
var SNMP_PRIVPROTO_REV = map[string]string{}
var SNMP_PRIVPROTO_ARR = []string{}

var SNMP_SECLEVEL = map[string]string{
	"noauthnopriv": "0",
	"authnopriv":   "1",
	"authpriv":     "2",
}
var SNMP_SECLEVEL_REV = map[string]string{}
var SNMP_SECLEVEL_ARR = []string{}

// generate the above structures
var _ = func() bool {
	for k, v := range SNMP_LOOKUP {
		SNMP_LOOKUP_REV[v] = k
		SNMP_LOOKUP_ARR = append(SNMP_LOOKUP_ARR, k)
	}
	for k, v := range SNMP_AUTHPROTO {
		SNMP_AUTHPROTO_REV[v] = k
		SNMP_AUTHPROTO_ARR = append(SNMP_AUTHPROTO_ARR, k)
	}
	for k, v := range SNMP_PRIVPROTO {
		SNMP_PRIVPROTO_REV[v] = k
		SNMP_PRIVPROTO_ARR = append(SNMP_PRIVPROTO_ARR, k)
	}
	for k, v := range SNMP_SECLEVEL {
		SNMP_SECLEVEL_REV[v] = k
		SNMP_SECLEVEL_ARR = append(SNMP_SECLEVEL_ARR, k)
	}
	return false
}()

var schemaSnmp = map[string]*schema.Schema{
	"snmp_version": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Default:      "2",
		Description:  "SNMP Version, one of: " + strings.Join(SNMP_LOOKUP_ARR, ", "),
		ValidateFunc: validation.StringInSlice(SNMP_LOOKUP_ARR, false),
	},
	"snmp_oid": &schema.Schema{
		Type:         schema.TypeString,
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Description:  "SNMP OID",
		Required:     true,
	},
	"snmp_community": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "SNMP Community (v1/v2 only)",
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
		Description:  "Authentication Protocol (v3 only), one of: " + strings.Join(SNMP_AUTHPROTO_ARR, ", "),
		ValidateFunc: validation.StringInSlice(SNMP_AUTHPROTO_ARR, false),
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
		Description:  "Priv Protocol (v3 only), one of: " + strings.Join(SNMP_PRIVPROTO_ARR, ", "),
		ValidateFunc: validation.StringInSlice(SNMP_PRIVPROTO_ARR, false),
		Default:      "aes",
	},
	"snmp3_securitylevel": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "Security Level (v3 only), one of: " + strings.Join(SNMP_SECLEVEL_ARR, ", "),
		ValidateFunc: validation.StringInSlice(SNMP_SECLEVEL_ARR, false),
		Default:      "authpriv",
	},
	"snmp3_securityname": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "Security Name (v3 only)",
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Default:      "{$SNMP3_SECURITYNAME}",
	},
}

// terraform resource handler for item type
func resourceItemSnmp() *schema.Resource {
	return &schema.Resource{
		Create: itemGetCreateWrapper(itemSnmpModFunc, itemSnmpReadFunc),
		Read:   itemGetReadWrapper(itemSnmpReadFunc),
		Update: itemGetUpdateWrapper(itemSnmpModFunc, itemSnmpReadFunc),
		Delete: resourceItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, schemaSnmp),
	}
}
func resourceProtoItemSnmp() *schema.Resource {
	return &schema.Resource{
		Create: protoItemGetCreateWrapper(itemSnmpModFunc, itemSnmpReadFunc),
		Read:   protoItemGetReadWrapper(itemSnmpReadFunc),
		Update: protoItemGetUpdateWrapper(itemSnmpModFunc, itemSnmpReadFunc),
		Delete: resourceProtoItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, itemPrototypeSchema, schemaSnmp),
	}
}

func resourceLLDSnmp() *schema.Resource {
	return &schema.Resource{
		Create: lldGetCreateWrapper(lldSnmpModFunc, lldSnmpReadFunc),
		Read:   lldGetReadWrapper(lldSnmpReadFunc),
		Update: lldGetUpdateWrapper(lldSnmpModFunc, lldSnmpReadFunc),
		Delete: resourceLLDDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: mergeSchemas(lldCommonSchema, lldInterfaceSchema, schemaSnmp),
	}
}

// Custom mod handler for item type
func itemSnmpModFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	api := m.(*zabbix.API)
	item.InterfaceID = d.Get("interfaceid").(string)
	item.Delay = d.Get("delay").(string)

	item.SNMPOid = d.Get("snmp_oid").(string)

	// new mode
	if api.Config.Version >= 5 {
		item.Type = zabbix.SNMPAgent
	} else { // old mode
		item.Type = SNMP_LOOKUP[d.Get("snmp_version").(string)]
		switch item.Type {
		case zabbix.SNMPv1Agent, zabbix.SNMPv2Agent:
			item.SNMPCommunity = d.Get("snmp_community").(string)
		case zabbix.SNMPv3Agent:
			item.SNMPv3AuthPassphrase = d.Get("snmp3_authpassphrase").(string)
			item.SNMPv3AuthProtocol = SNMP_AUTHPROTO[d.Get("snmp3_authprotocol").(string)]
			item.SNMPv3ContextName = d.Get("snmp3_contextname").(string)
			item.SNMPv3PrivPasshrase = d.Get("snmp3_privpassphrase").(string)
			item.SNMPv3PrivProtocol = SNMP_PRIVPROTO[d.Get("snmp3_privprotocol").(string)]
			item.SNMPv3SecurityLevel = SNMP_SECLEVEL[d.Get("snmp3_securitylevel").(string)]
			item.SNMPv3SecurityName = d.Get("snmp3_securityname").(string)
		}
	}
}

// Also for LLD Discovery SNMP
func lldSnmpModFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
	api := m.(*zabbix.API)
	item.InterfaceID = d.Get("interfaceid").(string)

	item.SNMPOid = d.Get("snmp_oid").(string)

	if api.Config.Version >= 5 {
		item.Type = zabbix.SNMPAgent
	} else { // old mode
		item.Type = SNMP_LOOKUP[d.Get("snmp_version").(string)]
		switch item.Type {
		case zabbix.SNMPv1Agent, zabbix.SNMPv2Agent:
			item.SNMPCommunity = d.Get("snmp_community").(string)
		case zabbix.SNMPv3Agent:
			item.SNMPv3AuthPassphrase = d.Get("snmp3_authpassphrase").(string)
			item.SNMPv3AuthProtocol = SNMP_AUTHPROTO[d.Get("snmp3_authprotocol").(string)]
			item.SNMPv3ContextName = d.Get("snmp3_contextname").(string)
			item.SNMPv3PrivPasshrase = d.Get("snmp3_privpassphrase").(string)
			item.SNMPv3PrivProtocol = SNMP_PRIVPROTO[d.Get("snmp3_privprotocol").(string)]
			item.SNMPv3SecurityLevel = SNMP_SECLEVEL[d.Get("snmp3_securitylevel").(string)]
			item.SNMPv3SecurityName = d.Get("snmp3_securityname").(string)
		}
	}
}

// Custom read handler for item type
func itemSnmpReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	api := m.(*zabbix.API)
	d.Set("interfaceid", item.InterfaceID)
	d.Set("delay", item.Delay)

	d.Set("snmp_oid", item.SNMPOid)

	if api.Config.Version < 5 {
		d.Set("type", SNMP_LOOKUP_REV[item.Type]) // may be null, check
		switch item.Type {
		case zabbix.SNMPv1Agent, zabbix.SNMPv2Agent:
			d.Set("snmp_community", item.SNMPCommunity)
		case zabbix.SNMPv3Agent:
			d.Set("snmp3_authpassphrase", item.SNMPv3AuthPassphrase)
			d.Set("snmp3_authprotocol", SNMP_AUTHPROTO_REV[item.SNMPv3AuthProtocol])
			d.Set("snmp3_contextname", item.SNMPv3ContextName)
			d.Set("snmp3_privpassphrase", item.SNMPv3PrivPasshrase)
			d.Set("snmp3_privprotocol", SNMP_PRIVPROTO_REV[item.SNMPv3PrivProtocol])
			d.Set("snmp3_securitylevel", SNMP_SECLEVEL_REV[item.SNMPv3SecurityLevel])
			d.Set("snmp3_securityname", item.SNMPv3SecurityName)
		}
	}
}

// Also for LLD Discovery SNMP
func lldSnmpReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
	api := m.(*zabbix.API)
	d.Set("interfaceid", item.InterfaceID)

	d.Set("snmp_oid", item.SNMPOid)

	if api.Config.Version < 5 {
		d.Set("type", SNMP_LOOKUP_REV[item.Type]) // may be null, check
		switch item.Type {
		case zabbix.SNMPv1Agent, zabbix.SNMPv2Agent:
			d.Set("snmp_community", item.SNMPCommunity)
		case zabbix.SNMPv3Agent:
			d.Set("snmp3_authpassphrase", item.SNMPv3AuthPassphrase)
			d.Set("snmp3_authprotocol", SNMP_AUTHPROTO_REV[item.SNMPv3AuthProtocol])
			d.Set("snmp3_contextname", item.SNMPv3ContextName)
			d.Set("snmp3_privpassphrase", item.SNMPv3PrivPasshrase)
			d.Set("snmp3_privprotocol", SNMP_PRIVPROTO_REV[item.SNMPv3PrivProtocol])
			d.Set("snmp3_securitylevel", SNMP_SECLEVEL_REV[item.SNMPv3SecurityLevel])
			d.Set("snmp3_securityname", item.SNMPv3SecurityName)
		}
	}
}
