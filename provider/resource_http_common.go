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

var HTTP_RETRIEVEMODE = map[string]string{
	"body":    "0",
	"headers": "1",
	"both":    "2",
}
var HTTP_RETRIEVEMODE_REV = map[string]string{}
var HTTP_RETRIEVEMODE_ARR = []string{}

var HTTP_POSTTYPE = map[string]string{
	"raw":  "0",
	"json": "2",
	"xml":  "3",
}
var HTTP_POSTTYPE_REV = map[string]string{}
var HTTP_POSTTYPE_ARR = []string{}

var HTTP_AUTHTYPE = map[string]string{
	"none":     "0",
	"basic":    "1",
	"ntlm":     "2",
	"kerberos": "3",
}
var HTTP_AUTHTYPE_REV = map[string]string{}
var HTTP_AUTHTYPE_ARR = []string{}

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
	for k, v := range HTTP_RETRIEVEMODE {
		HTTP_RETRIEVEMODE_REV[v] = k
		HTTP_RETRIEVEMODE_ARR = append(HTTP_RETRIEVEMODE_ARR, k)
	}
	for k, v := range HTTP_AUTHTYPE {
		HTTP_AUTHTYPE_REV[v] = k
		HTTP_AUTHTYPE_ARR = append(HTTP_AUTHTYPE_ARR, k)
	}
	return false
}()

var schemaHttpHeader = &schema.Schema{
	Type:     schema.TypeMap,
	Optional: true,
	Elem: &schema.Schema{
		Type:         schema.TypeString,
		Description:  "Header Value",
		ValidateFunc: validation.StringIsNotWhiteSpace,
	},
}

var schemaHttp = map[string]*schema.Schema{
	"url": &schema.Schema{
		Type:         schema.TypeString,
		Description:  "url to probe",
		ValidateFunc: validation.StringIsNotWhiteSpace,
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
	"retrieve_mode": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "HTTP retrieve mode, one of: " + strings.Join(HTTP_RETRIEVEMODE_ARR, ", "),
		ValidateFunc: validation.StringInSlice(HTTP_RETRIEVEMODE_ARR, false),
		Default:      "body",
	},
	"auth_type": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "HTTP auth type, one of: " + strings.Join(HTTP_AUTHTYPE_ARR, ", "),
		ValidateFunc: validation.StringInSlice(HTTP_AUTHTYPE_ARR, false),
		Default:      "none",
	},
	"username": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Authentication Username",
	},
	"proxy": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "HTTP proxy connection string",
	},
	"password": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "Authentication Password",
	},
	"headers": schemaHttpHeader,
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
		Default:     true,
	},
	"verify_peer": &schema.Schema{
		Type:        schema.TypeBool,
		Description: "https verify peer",
		Optional:    true,
		Default:     true,
	},
}

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

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, schemaHttp),
	}
}
func resourceProtoItemHttp() *schema.Resource {
	return &schema.Resource{
		Create: protoItemGetCreateWrapper(itemHttpModFunc, itemHttpReadFunc),
		Read:   protoItemGetReadWrapper(itemHttpReadFunc),
		Update: protoItemGetUpdateWrapper(itemHttpModFunc, itemHttpReadFunc),
		Delete: resourceProtoItemDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, itemPrototypeSchema, schemaHttp),
	}
}
func resourceLLDHttp() *schema.Resource {
	return &schema.Resource{
		Create: lldGetCreateWrapper(lldHttpModFunc, lldHttpReadFunc),
		Read:   lldGetReadWrapper(lldHttpReadFunc),
		Update: lldGetUpdateWrapper(lldHttpModFunc, lldHttpReadFunc),
		Delete: resourceLLDDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: mergeSchemas(lldCommonSchema, itemInterfaceSchema, schemaHttp),
	}
}

func httpGenerateHeaders(d *schema.ResourceData) (headers zabbix.HttpHeaders) {
	m := d.Get("headers").(map[string]interface{})
	headers = zabbix.HttpHeaders{}

	for k, v := range m {
		headers[k] = v.(string)
	}

	return
}

func httpFlattenHeaders(headers zabbix.HttpHeaders) (ret map[string]interface{}) {
	ret = map[string]interface{}{}
	for k, v := range headers {
		ret[k] = v
	}
	return
}

// http item modify custom function
func itemHttpModFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	item.InterfaceID = d.Get("interfaceid").(string)
	item.Url = d.Get("url").(string)
	item.Delay = d.Get("delay").(string)
	item.RequestMethod = HTTP_METHODS[d.Get("request_method").(string)]
	item.PostType = HTTP_POSTTYPE[d.Get("post_type").(string)]
	item.RetrieveMode = HTTP_RETRIEVEMODE[d.Get("retrieve_mode").(string)]
	item.AuthType = HTTP_AUTHTYPE[d.Get("auth_type").(string)]
	item.Username = d.Get("username").(string)
	item.Proxy = d.Get("proxy").(string)
	item.Password = d.Get("password").(string)
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
	item.Headers = httpGenerateHeaders(d)
}
func lldHttpModFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
	item.InterfaceID = d.Get("interfaceid").(string)
	item.Url = d.Get("url").(string)
	item.RequestMethod = HTTP_METHODS[d.Get("request_method").(string)]
	item.PostType = HTTP_POSTTYPE[d.Get("post_type").(string)]
	item.RetrieveMode = HTTP_RETRIEVEMODE[d.Get("retrieve_mode").(string)]
	item.AuthType = HTTP_AUTHTYPE[d.Get("auth_type").(string)]
	item.Username = d.Get("username").(string)
	item.Proxy = d.Get("proxy").(string)
	item.Password = d.Get("password").(string)
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
	item.Headers = httpGenerateHeaders(d)
}

// http item read custom function
func itemHttpReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("url", item.Url)
	d.Set("delay", item.Delay)
	d.Set("request_method", HTTP_METHODS_REV[item.RequestMethod])
	d.Set("post_type", HTTP_POSTTYPE_REV[item.PostType])
	d.Set("retrieve_mode", HTTP_RETRIEVEMODE_REV[item.RetrieveMode])
	d.Set("auth_type", HTTP_AUTHTYPE_REV[item.AuthType])
	d.Set("username", item.Username)
	d.Set("proxy", item.Proxy)
	d.Set("password", item.Password)
	d.Set("posts", item.Posts)
	d.Set("status_codes", item.StatusCodes)
	d.Set("timeout", item.Timeout)
	d.Set("verify_host", item.VerifyHost == "1")
	d.Set("verify_peer", item.VerifyPeer == "1")
	d.Set("headers", httpFlattenHeaders(item.Headers))
}
func lldHttpReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.LLDRule) {
	d.Set("interfaceid", item.InterfaceID)
	d.Set("url", item.Url)
	d.Set("request_method", HTTP_METHODS_REV[item.RequestMethod])
	d.Set("post_type", HTTP_POSTTYPE_REV[item.PostType])
	d.Set("retrieve_mode", HTTP_RETRIEVEMODE_REV[item.RetrieveMode])
	d.Set("auth_type", HTTP_AUTHTYPE_REV[item.AuthType])
	d.Set("username", item.Username)
	d.Set("proxy", item.Proxy)
	d.Set("password", item.Password)
	d.Set("posts", item.Posts)
	d.Set("status_codes", item.StatusCodes)
	d.Set("timeout", item.Timeout)
	d.Set("verify_host", item.VerifyHost == "1")
	d.Set("verify_peer", item.VerifyPeer == "1")
	d.Set("headers", httpFlattenHeaders(item.Headers))
}
