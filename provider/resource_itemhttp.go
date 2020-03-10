package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

func resourceItemHttp() *schema.Resource {
	return &schema.Resource{
		Create: itemGetCreateWrapper(itemHttpModFunc, itemHttpReadFunc),
		Read:   itemGetReadWrapper(itemHttpReadFunc),
		Update: itemGetUpdateWrapper(itemHttpModFunc, itemHttpReadFunc),
		Delete: resourceItemDelete,

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, map[string]*schema.Schema{
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
		}),
	}
}

func itemHttpModFunc(d *schema.ResourceData, item *zabbix.Item) {
	item.InterfaceID = d.Get("interfaceid").(string)
	item.Url = d.Get("url").(string)
	item.Delay = d.Get("delay").(string)
	item.RequestMethod = d.Get("request_method").(string)
	item.PostType = d.Get("post_type").(string)
	item.Posts = d.Get("posts").(string)
	item.StatusCodes = d.Get("status_codes").(string)
	item.Timeout = d.Get("timeout").(string)
	item.Type = zabbix.HTTPAgent
	item.VerifyHost = "0"
	item.VerifyPeer = "0"

	if d.Get("verify_host").(bool) {
		item.VerifyHost = "1"
	}

	if d.Get("verify_peer").(bool) {
		item.VerifyPeer = "1"
	}
}

func itemHttpReadFunc(d *schema.ResourceData, item *zabbix.Item) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("url", item.Url)
	d.Set("delay", item.Delay)
	d.Set("request_method", item.RequestMethod)
	d.Set("post_type", item.PostType)
	d.Set("posts", item.Posts)
	d.Set("status_codes", item.StatusCodes)
	d.Set("timeout", item.Timeout)
	d.Set("verify_host", item.VerifyHost == "1")
	d.Set("verify_peer", item.VerifyPeer == "1")
}
