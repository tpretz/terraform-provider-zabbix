package provider

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

func resourceItemHttp() *schema.Resource {
	return &schema.Resource{
		Create: resourceItemHttpCreate,
		Read:   resourceItemHttpRead,
		Update: resourceItemHttpUpdate,
		Delete: resourceItemDelete,

		Schema: map[string]*schema.Schema{
			"hostid": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Host ID",
			},
			"interfaceid": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Host Interface ID",
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
				Required: true,
				Default:  "1",
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
				Default:  "0",
			},
			"snmp_community": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "0",
			},
			"snmp3_authpassphrase": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"snmp3_authprotocol": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"snmp3_contextname": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"snmp3_privpassphrase": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"snmp3_privprotocol": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"snmp3_securitylevel": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"snmp3_securityname": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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

func buildItemHttpObject(d *schema.ResourceData) *zabbix.Item {

	item := zabbix.Item{
		Key:         d.Get("key").(string),
		HostID:      d.Get("hostid").(string),
		Name:        d.Get("name").(string),
		Type:        SNMP_LOOKUP[d.Get("snmp_version").(string)],
		ValueType:   zabbix.ValueType(d.Get("valuetype").(int)),
		Delay:       d.Get("delay").(string),
		InterfaceID: d.Get("interfaceid").(string),

		
	}

	item.Preprocessors = itemGeneratePreprocessors(d)

	return &item
}

func resourceItemHttpCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := buildItemHttpObject(d)
	items := []zabbix.Item{*item}

	err := api.ItemsCreate(items)

	if err != nil {
		return err
	}

	log.Trace("created item: %+v", items[0])

	d.SetId(items[0].ItemID)

	return resourceItemHttpRead(d, m)
}

func resourceItemHttpRead(d *schema.ResourceData, m interface{}) error {
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

	d.Set("url", item.Url)
	d.Set("request_method", item.RequestMethod)
	d.Set("post_type", item.PostType)
	d.Set("posts", item.Posts)
	d.Set("status_codes", item.StatusCodes)
	d.Set("timeout", item.Timeout)
	d.Set("verify_host", item.VerifyHost == "1")
	d.Set("verify_peer", item.VerifyPeer == "1")

	d.Set("preprocessor", flattenItemPreprocessors(item))

	return nil
}

func resourceItemHttpUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := buildItemHttpObject(d)
	item.ItemID = d.Id()

	items := []zabbix.Item{*item}

	err := api.ItemsUpdate(items)

	if err != nil {
		return err
	}

	return resourceItemHttpRead(d, m)
}
