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
			"delay": &schema.Schema{
				Type:        schema.TypeString,
				Required:    false,
				Optional:    true,
				Description: "Item Delay",
			},
			"hostid": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Host ID",
			},
			"interfaceid": &schema.Schema{
				Type:        schema.TypeString,
				Required:    false,
				Optional:    true,
				Description: "Interface ID",
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
		HostId:    d.Get("hostid").(string),
		Name:      d.Get("name").(string),
		ValueType: d.Get("valuetype").(zabbix.ValueType),
	}

	if v, ok := d.GetOk("delay"); ok {
		item.Delay = v.(string)
	}

	if v, ok := d.GetOk("interfaceid"); ok {
		item.InterfaceId = v.(string)
	}

	items := []zabbix.Item{item}

	err := api.ItemsCreate(items)

	if err != nil {
		return err
	}

	log.Trace("crated item: %+v", items[0])

	d.Set("itemid", items[0].ItemId)
	d.SetId(items[0].ItemId)

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

	d.Set("delay", item.Delay)
	d.Set("hostid", item.HostID)
	d.Set("interfaceid", item.InterfaceID)
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
