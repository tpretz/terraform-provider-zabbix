package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

func resourceItemHttp() *schema.Resource {
	return &schema.Resource{
		Create: resourceItemHttpCreate,
		Read:   resourceItemHttpRead,
		Update: resourceItemHttpUpdate,
		Delete: resourceItemHttpDelete,

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
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"valuetype": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"request_method": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  0,
			},
			"post_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  0,
			},
			"posts": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"status_codes": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"timeout": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"verify_host": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"verify_peer": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func buildItemHttpObject(d *schema.ResourceData) *zabbix.Item {
	item := zabbix.Item{
		Key:       d.Get("key").(string),
		HostID:    d.Get("hostid").(string),
		Name:      d.Get("name").(string),
		Type:      zabbix.HTTPAgent,
		ValueType: zabbix.ValueType(d.Get("valuetype").(int)),

		Url:           d.Get("url").(string),
		RequestMethod: d.Get("request_method").(string),
		PostType:      d.Get("post_type").(string),
		Posts:         d.Get("posts").(string),
		StatusCodes:   d.Get("status_codes").(string),
		Timeout:       d.Get("timeout").(string),
		VerifyHost:    "0",
		VerifyPeer:    "0",
	}

	if d.Get("verify_host").(bool) {
		item.VerifyHost = "1"
	}

	if d.Get("verify_peer").(bool) {
		item.VerifyPeer = "1"
	}

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

	d.Set("url", item.Url)
	d.Set("request_method", item.RequestMethod)
	d.Set("post_type", item.PostType)
	d.Set("posts", item.Posts)
	d.Set("status_codes", item.StatusCodes)
	d.Set("timeout", item.Timeout)
	d.Set("verify_host", item.VerifyHost == "1")
	d.Set("verify_peer", item.VerifyPeer == "1")

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

func resourceItemHttpDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.ItemsDeleteByIds([]string{d.Id()})
}
