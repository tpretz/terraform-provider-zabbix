package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

func resourceItemSnmp() *schema.Resource {
	return &schema.Resource{
		Create: itemGetCreateWrapper(itemSnmpModFunc, itemSnmpReadFunc),
		Read:   itemGetReadWrapper(itemSnmpReadFunc),
		Update: itemGetUpdateWrapper(itemSnmpModFunc, itemSnmpReadFunc),
		Delete: resourceItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, map[string]*schema.Schema{
			"snmp_version": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "2",
				ValidateFunc: validation.StringInSlice([]string{"1", "2", "3"}, false),
			},
			"snmp_oid": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"snmp_community": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "${SNMP_COMMUNITY}",
			},
			"snmp3_authpassphrase": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "${SNMP3_AUTHPASSPHRASE}",
			},
			"snmp3_authprotocol": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "1",
			},
			"snmp3_contextname": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "${SNMP3_CONTEXTNAME}",
			},
			"snmp3_privpassphrase": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "${SNMP3_PRIVPASSPHRASE}",
			},
			"snmp3_privprotocol": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "1",
			},
			"snmp3_securitylevel": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "2",
			},
			"snmp3_securityname": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "${SNMP3_SECURITYNAME}",
			},
		}),
	}
}

var SNMP_LOOKUP = map[string]zabbix.ItemType{
	"1": zabbix.SNMPv1Agent,
	"2": zabbix.SNMPv2Agent,
	"3": zabbix.SNMPv3Agent,
}
var SNMP_LOOKUP_REV = map[zabbix.ItemType]string{
	zabbix.SNMPv1Agent: "1",
	zabbix.SNMPv2Agent: "2",
	zabbix.SNMPv3Agent: "3",
}

func itemSnmpModFunc(d *schema.ResourceData, item *zabbix.Item) {
	item.Type = SNMP_LOOKUP[d.Get("snmp_version").(string)]
	item.InterfaceID = d.Get("interfaceid").(string)
	item.Delay = d.Get("delay").(string)

	item.SNMPOid = d.Get("snmp_oid").(string)

	switch item.Type {
	case zabbix.SNMPv1Agent, zabbix.SNMPv2Agent:
		item.SNMPCommunity = d.Get("snmp_community").(string)
	case zabbix.SNMPv3Agent:
		item.SNMPv3AuthPassphrase = d.Get("snmp3_authpassphrase").(string)
		item.SNMPv3AuthProtocol = d.Get("snmp3_authprotocol").(string)
		item.SNMPv3ContextName = d.Get("snmp3_contextname").(string)
		item.SNMPv3PrivPasshrase = d.Get("snmp3_privpassphrase").(string)
		item.SNMPv3PrivProtocol = d.Get("snmp3_privprotocol").(string)
		item.SNMPv3SecurityLevel = d.Get("snmp3_securitylevel").(string)
		item.SNMPv3SecurityName = d.Get("snmp3_securityname").(string)
	}
}

func itemSnmpReadFunc(d *schema.ResourceData, item *zabbix.Item) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("delay", item.Delay)
	d.Set("type", SNMP_LOOKUP_REV[item.Type]) // may be null, check

	d.Set("snmp_oid", item.SNMPOid)

	switch item.Type {
	case zabbix.SNMPv1Agent, zabbix.SNMPv2Agent:
		d.Set("snmp_community", item.SNMPCommunity)
	case zabbix.SNMPv3Agent:
		d.Set("snmp3_authpassphrase", item.SNMPv3AuthPassphrase)
		d.Set("snmp3_authprotocol", item.SNMPv3AuthProtocol)
		d.Set("snmp3_contextname", item.SNMPv3ContextName)
		d.Set("snmp3_privpassphrase", item.SNMPv3PrivPasshrase)
		d.Set("snmp3_privprotocol", item.SNMPv3PrivProtocol)
		d.Set("snmp3_securitylevel", item.SNMPv3SecurityLevel)
		d.Set("snmp3_securityname", item.SNMPv3SecurityName)
	}
}
