package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

func resourceItemSimple() *schema.Resource {
	return &schema.Resource{
		Create: resourceItemSimpleCreate,
		Read:   resourceItemSimpleRead,
		Update: resourceItemSimpleUpdate,
		Delete: resourceItemSimpleDelete,

		Schema: map[string]*schema.Schema{
			"hostid": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Host ID",
			},
			"key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"delay": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "1m",
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"valuetype": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
		},
	}
}

func buildItemSimpleObject(d *schema.ResourceData) *zabbix.Item {
	item := zabbix.Item{
		Key:       d.Get("key").(string),
		HostID:    d.Get("hostid").(string),
		Name:      d.Get("name").(string),
		Type:      zabbix.SimpleCheck,
		ValueType: zabbix.ValueType(d.Get("valuetype").(int)),
		Delay:     d.Get("delay").(string),
	}

	return &item
}

func resourceItemSimpleCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := buildItemSimpleObject(d)
	items := []zabbix.Item{*item}

	err := api.ItemsCreate(items)

	if err != nil {
		return err
	}

	log.Trace("crated item: %+v", items[0])

	d.SetId(items[0].ItemID)

	return resourceItemSimpleRead(d, m)
}

func resourceItemSimpleRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of item with id %s", d.Id())

	item, err := api.ItemGetByID(d.Id())

	if err != nil {
		return err
	}

	log.Debug("Got item: %+v", item)

	d.SetId(item.ItemID)
	d.Set("hostid", item.HostID)
	d.Set("key", item.Key)
	d.Set("name", item.Name)
	d.Set("valuetype", item.ValueType)
	d.Set("delay", item.Delay)

	return nil
}

func resourceItemSimpleUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := buildItemSimpleObject(d)
	item.ItemID = d.Id()

	items := []zabbix.Item{*item}

	err := api.ItemsUpdate(items)

	if err != nil {
		return err
	}

	return resourceItemSimpleRead(d, m)
}

func resourceItemSimpleDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.ItemsDeleteByIds([]string{d.Id()})
}
