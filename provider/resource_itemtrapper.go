package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

func resourceItemTrapper() *schema.Resource {
	return &schema.Resource{
		Create: resourceItemTrapperCreate,
		Read:   resourceItemTrapperRead,
		Update: resourceItemTrapperUpdate,
		Delete: resourceItemTrapperDelete,

		Schema: map[string]*schema.Schema{
			"itemid": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "Zabbix ID",
			},
			"hostid": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Host ID",
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
		},
	}
}

func resourceItemTrapperCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := zabbix.Item{
		Key:       d.Get("key").(string),
		HostID:    d.Get("hostid").(string),
		Name:      d.Get("name").(string),
		Type:      zabbix.ZabbixTrapper,
		ValueType: zabbix.ValueType(d.Get("valuetype").(int)),
	}

	items := []zabbix.Item{item}

	err := api.ItemsCreate(items)

	if err != nil {
		return err
	}

	log.Trace("crated item: %+v", items[0])

	d.Set("itemid", items[0].ItemID)
	d.SetId(items[0].ItemID)

	return resourceItemTrapperRead(d, m)
}

func resourceItemTrapperRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	id := d.Get("itemid").(string)

	log.Debug("Lookup of item with id %s", id)

	item, err := api.ItemGetByID(id)

	if err != nil {
		return err
	}

	log.Debug("Got item: %+v", item)

	d.Set("hostid", item.HostID)
	d.Set("key", item.Key)
	d.Set("name", item.Name)
	d.Set("valuetype", item.ValueType)

	return nil
}

func resourceItemTrapperUpdate(d *schema.ResourceData, m interface{}) error {
	return resourceItemTrapperRead(d, m)
}

func resourceItemTrapperDelete(d *schema.ResourceData, m interface{}) error {
	return nil
}
