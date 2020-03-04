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
		Delete: resourceItemHttpDelete,

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
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"request_method": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "0",
			},
			"post_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "0",
			},
			"posts": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"status_codes": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "200",
			},
			"timeout": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "3s",
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
			"preprocessor": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"params": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"error_handler": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "0",
						},
						"error_handler_params": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
					},
				},
			},
		},
	}
}

func itemGeneratePreprocessors(d *schema.ResourceData) (preprocessors zabbix.Preprocessors) {
	preprocessorCount := d.Get("preprocessor.#").(int)
	preprocessors = make(zabbix.Preprocessors, preprocessorCount)

	for i := 0; i < preprocessorCount; i++ {
		prefix := fmt.Sprintf("preprocessor.%d.", i)

		preprocessors[i] = zabbix.Preprocessor{
			Type:               d.Get(prefix + "type").(string),
			Params:             d.Get(prefix + "params").(string),
			ErrorHandler:       d.Get(prefix + "error_handler").(string),
			ErrorHandlerParams: d.Get(prefix + "error_handler_params").(string),
		}
	}

	return
}

func buildItemHttpObject(d *schema.ResourceData) *zabbix.Item {
	item := zabbix.Item{
		Key:         d.Get("key").(string),
		HostID:      d.Get("hostid").(string),
		Name:        d.Get("name").(string),
		Type:        zabbix.HTTPAgent,
		ValueType:   zabbix.ValueType(d.Get("valuetype").(int)),
		Delay:       d.Get("delay").(string),
		InterfaceID: d.Get("interfaceid").(string),

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

	val := make([]interface{}, len(item.Preprocessors))
	for i := 0; i < len(item.Preprocessors); i++ {
		current := map[string]interface{}{}
		//current["id"] = host.Interfaces[i].InterfaceID
		current["type"] = item.Preprocessors[i].Type
		current["params"] = item.Preprocessors[i].Params
		current["error_handler"] = item.Preprocessors[i].ErrorHandler
		current["error_handler_params"] = item.Preprocessors[i].ErrorHandlerParams
		val[i] = current
	}
	d.Set("preprocessor", val)

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
