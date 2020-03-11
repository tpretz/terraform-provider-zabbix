package provider

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

var HTTP_METHODS = map[string]string{
	"get":  "0",
	"post": "1",
	"put":  "2",
	"head": "3",
}
var HTTP_METHODS_REV = map[string]string{}
var HTTP_METHODS_ARR = []string{}

var HTTP_POSTTYPE = map[string]string{
	"body":    "0",
	"headers": "1",
	"both":    "2",
}
var HTTP_POSTTYPE_REV = map[string]string{}
var HTTP_POSTTYPE_ARR = []string{}

// generate the above structures
var _ = func() bool {
	for k, v := range HTTP_METHODS {
		HTTP_METHODS_REV[v] = k
		HTTP_METHODS_ARR = append(HTTP_METHODS_ARR, k)
	}
	for k, v := range HTTP_POSTTYPE {
		HTTP_POSTTYPE_REV[v] = k
		HTTP_POSTTYPE_ARR = append(HTTP_POSTTYPE_ARR, k)
	}
	return false
}()

// resourceItemHttp Http item resource handler
func resourceItemHttp() *schema.Resource {
	return &schema.Resource{
		Create: itemGetCreateWrapper(itemHttpModFunc, itemHttpReadFunc),
		Read:   itemGetReadWrapper(itemHttpReadFunc),
		Update: itemGetUpdateWrapper(itemHttpModFunc, itemHttpReadFunc),
		Delete: resourceItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, map[string]*schema.Schema{
			"url": &schema.Schema{
				Type:         schema.TypeString,
				Description:  "url to probe",
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
				Required:     true,
			},
			"request_method": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "HTTP request method, one of: " + strings.Join(HTTP_METHODS_ARR, ", "),
				ValidateFunc: validation.StringInSlice(HTTP_METHODS_ARR, false),
				Default:      "get",
			},
			"post_type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "HTTP post type, one of: " + strings.Join(HTTP_POSTTYPE_ARR, ", "),
				ValidateFunc: validation.StringInSlice(HTTP_POSTTYPE_ARR, false),
				Default:      "body",
			},
			"posts": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "POST data to send in request",
			},
			"status_codes": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "200",
				Description: "http status code",
			},
			"timeout": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "http request timeout",
				Default:     "3s",
			},
			"verify_host": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "https verify host",
				Default:     false,
			},
			"verify_peer": &schema.Schema{
				Type:        schema.TypeBool,
				Description: "https verify peer",
				Optional:    true,
				Default:     false,
			},
		}),
	}
}

// http item modify custom function
func itemHttpModFunc(d *schema.ResourceData, item *zabbix.Item) {
	item.InterfaceID = d.Get("interfaceid").(string)
	item.Url = d.Get("url").(string)
	item.Delay = d.Get("delay").(string)
	item.RequestMethod = HTTP_METHODS[d.Get("request_method").(string)]
	item.PostType = HTTP_POSTTYPE[d.Get("post_type").(string)]
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

// http item read custom function
func itemHttpReadFunc(d *schema.ResourceData, item *zabbix.Item) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("url", item.Url)
	d.Set("delay", item.Delay)
	d.Set("request_method", HTTP_METHODS_REV[item.RequestMethod])
	d.Set("post_type", HTTP_POSTTYPE_REV[item.PostType])
	d.Set("posts", item.Posts)
	d.Set("status_codes", item.StatusCodes)
	d.Set("timeout", item.Timeout)
	d.Set("verify_host", item.VerifyHost == "1")
	d.Set("verify_peer", item.VerifyPeer == "1")
}
