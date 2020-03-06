package provider

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

func resourceItemSnmp() *schema.Resource {
	return &schema.Resource{
		Create: resourceItemSnmpCreate,
		Read:   resourceItemSnmpRead,
		Update: resourceItemSnmpUpdate,
		Delete: resourceItemDelete,

		Schema: map[string]*schema.Schema{
			"hostid": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Host ID",
			},
			"interfaceid": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Host Interface ID",
				Default:     "0",
			},
			"key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"valuetype": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"delay": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "1m",
			},
			"snmp_version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "2",
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(string)
					switch val.(string) {
					case "1", "2", "3":
						return
					}

					errs = append(errs, fmt.Errorf("%q must be 1, 2 or 3, got: %d", key, v))
					return
				},
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
			"preprocessor": itemPreprocessorSchema,
		},
	}
}

var SNMP_LOOKUP = map[string]zabbix.ItemType{
	"1": zabbix.SNMPv1Agent,
	"2": zabbix.SNMPv2Agent,
	"3": zabbix.SNMPv3Agent,
}

func buildItemSnmpObject(d *schema.ResourceData) *zabbix.Item {

	item := zabbix.Item{
		Key:         d.Get("key").(string),
		HostID:      d.Get("hostid").(string),
		Name:        d.Get("name").(string),
		Type:        SNMP_LOOKUP[d.Get("snmp_version").(string)],
		ValueType:   zabbix.ValueType(d.Get("valuetype").(int)),
		Delay:       d.Get("delay").(string),
		InterfaceID: d.Get("interfaceid").(string),

		SNMPOid: d.Get("snmp_oid").(string),
	}

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

	item.Preprocessors = itemGeneratePreprocessors(d)

	return &item
}

func resourceItemSnmpCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := buildItemSnmpObject(d)
	items := []zabbix.Item{*item}

	err := api.ItemsCreate(items)

	if err != nil {
		return err
	}

	log.Trace("created item: %+v", items[0])

	d.SetId(items[0].ItemID)

	return resourceItemSnmpRead(d, m)
}

func resourceItemSnmpRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of item with id %s", d.Id())

	items, err := api.ItemsGet(zabbix.Params{
		"itemids":             []string{d.Id()},
		"selectPreprocessing": "extend",
	})

	if err != nil {
		return err
	}

	if len(items) < 1 {
		return errors.New("no item found")
	}
	if len(items) > 1 {
		return errors.New("multiple items found")
	}
	item := items[0]

	log.Debug("Got item: %+v", item)

	d.SetId(item.ItemID)
	d.Set("hostid", item.HostID)
	d.Set("interfaceid", item.InterfaceID)
	d.Set("key", item.Key)
	d.Set("name", item.Name)
	d.Set("valuetype", item.ValueType)
	d.Set("delay", item.Delay)

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

	d.Set("preprocessor", flattenItemPreprocessors(item))

	return nil
}

func resourceItemSnmpUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := buildItemSnmpObject(d)
	item.ItemID = d.Id()

	items := []zabbix.Item{*item}

	err := api.ItemsUpdate(items)

	if err != nil {
		return err
	}

	return resourceItemSnmpRead(d, m)
}
