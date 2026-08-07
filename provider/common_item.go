package provider

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// Item Type Conversion and lookup tables
var ITEM_VALUE_TYPES = map[string]zabbix.ValueType{
	"float":     zabbix.Float,
	"character": zabbix.Character,
	"log":       zabbix.Log,
	"unsigned":  zabbix.Unsigned,
	"text":      zabbix.Text,
}
var ITEM_VALUE_TYPES_REV = map[zabbix.ValueType]string{
	zabbix.Float:     "float",
	zabbix.Character: "character",
	zabbix.Log:       "log",
	zabbix.Unsigned:  "unsigned",
	zabbix.Text:      "text",
}
var ITEM_VALUE_TYPES_ARR = []string{
	"float",
	"character",
	"log",
	"unsigned",
	"text",
}

// common schema elements for all item types
var itemCommonSchema = map[string]*schema.Schema{
	"hostid": &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		Description:  "ID of the host or template this item belongs to. Changing it replaces the item",
		ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9]+$"), "must be numeric"),
	},
	"key": &schema.Schema{
		Type:         schema.TypeString,
		Description:  "Item key, unique per host, e.g. `system.cpu.load[all,avg1]`",
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Required:     true,
	},
	"name": &schema.Schema{
		Type:         schema.TypeString,
		Description:  "Item name, as shown in the frontend",
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Required:     true,
	},
	"history": &schema.Schema{
		Type:         schema.TypeString,
		Description:  "How long Zabbix keeps raw history for this item, as a time suffix string, e.g. \"90d\". \"0\" disables history",
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Default:      "90d",
		Optional:     true,
	},
	"trends": &schema.Schema{
		Type:         schema.TypeString,
		Description:  "How long Zabbix keeps hourly trends for this item, e.g. \"365d\". \"0\" disables trends, and is the only valid value for text and log items. Defaults to \"365d\", or \"0\" for text and log",
		ValidateFunc: validation.StringIsNotWhiteSpace,
		//Default:      "365d",
		Optional: true,
		Computed: true,
	},
	"valuetype": &schema.Schema{
		Type:         schema.TypeString,
		ValidateFunc: validation.StringInSlice(ITEM_VALUE_TYPES_ARR, false),
		Description:  "Type of the value Zabbix stores, one of: " + strings.Join(ITEM_VALUE_TYPES_ARR, ", ") + ". Changing it after data has been collected leaves the old history behind",
		Required:     true,
	},
	"preprocessor": itemPreprocessorSchema,
	"tag":          tagSetSchema,
}

// Delay schema
var itemDelaySchema = map[string]*schema.Schema{
	"delay": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Default:      "1m",
		Description:  "How often the item is polled, as a time suffix string, e.g. \"1m\". Supports Zabbix's flexible and scheduling interval syntax",
	},
}

// Interface schema
var itemInterfaceSchema = map[string]*schema.Schema{
	"interfaceid": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "ID of the host interface to poll through. \"0\", the default, lets Zabbix pick the default interface of the matching type",
		Default:     "0",
	},
}

// Prototype schema
var itemPrototypeSchema = map[string]*schema.Schema{
	"ruleid": &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Description:  "ID of the discovery rule this prototype belongs to. Changing it replaces the prototype",
	},
}

// Schema for preprocessor blocks
var itemPreprocessorSchema = &schema.Schema{
	Type:        schema.TypeList,
	Optional:    true,
	Description: preprocessorDescription,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Step ID, assigned by Zabbix",
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  preprocessorTypeDescription,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9]+$"), "must be numeric"),
			},
			"params": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotWhiteSpace,
				},
				Optional:    true,
				Description: preprocessorParamsDescription,
			},
			"error_handler": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: preprocessorErrorHandlerDescription,
			},
			"error_handler_params": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: preprocessorErrorHandlerParamsDescription,
			},
		},
	},
}

// Preprocessing step descriptions, shared verbatim between the item and LLD
// preprocessor schemas. The two are separate code paths (common_item.go and
// common_lld.go each build their own) but the same Zabbix object, so a reader
// arriving at either page should be told the same thing.
const (
	preprocessorDescription = "Preprocessing steps, applied in the order written. This is a " +
		"list rather than a set precisely because that order is semantic: Zabbix feeds each " +
		"step the output of the previous one."

	preprocessorTypeDescription = "Preprocessing step type, as Zabbix's numeric code — e.g. " +
		"`5` regular expression, `11` XML XPath, `12` JSONPath, `20` discard unchanged with " +
		"heartbeat. See the Zabbix API documentation for `preprocessing.type`; the full list " +
		"grows with every release, so it is not enumerated or validated here."

	preprocessorParamsDescription = "Parameters for the step, one element per line Zabbix " +
		"expects. Which parameters apply, and how many, depends entirely on `type`."

	preprocessorErrorHandlerDescription = "What to do when this step fails, as Zabbix's " +
		"numeric code: `0` discard the value and report the error (the default), `1` discard " +
		"the value silently, `2` set the value in `error_handler_params`, `3` report the " +
		"error text in `error_handler_params`."

	preprocessorErrorHandlerParamsDescription = "Value or error text used by `error_handler` " +
		"codes `2` and `3`. Ignored otherwise."
)

// Function signature for context manipulation
type ItemHandler func(*schema.ResourceData, interface{}, *zabbix.Item)

// return a terraform CreateFunc
func itemGetCreateWrapper(c ItemHandler, r ItemHandler) schema.CreateFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		return resourceItemCreate(d, m, c, r, false)
	}
}
func protoItemGetCreateWrapper(c ItemHandler, r ItemHandler) schema.CreateFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		return resourceItemCreate(d, m, c, r, true)
	}
}

// return a terraform UpdateFunc
func itemGetUpdateWrapper(c ItemHandler, r ItemHandler) schema.UpdateFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		return resourceItemUpdate(d, m, c, r, false)
	}
}
func protoItemGetUpdateWrapper(c ItemHandler, r ItemHandler) schema.UpdateFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		return resourceItemUpdate(d, m, c, r, true)
	}
}

// return a terraform ReadFunc
func itemGetReadWrapper(r ItemHandler) schema.ReadFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		return resourceItemRead(d, m, r, false)
	}
}
func protoItemGetReadWrapper(r ItemHandler) schema.ReadFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		return resourceItemRead(d, m, r, true)
	}
}

// Create Item Resource Handler
func resourceItemCreate(d *schema.ResourceData, m interface{}, c ItemHandler, r ItemHandler, prototype bool) error {
	api := m.(*zabbix.API)

	item := buildItemObject(d, api, prototype)

	// run custom function
	c(d, m, item)

	log.Trace("preparing item object for create/update: %#v", item)

	items := []zabbix.Item{*item}

	var err error

	if prototype {
		err = api.ProtoItemsCreate(items)
	} else {
		err = api.ItemsCreate(items)
	}

	if err != nil {
		return err
	}

	log.Trace("created item: %+v", items[0])

	d.SetId(items[0].ItemID)

	return resourceItemRead(d, m, r, prototype)
}

// Update Item Resource Handler
func resourceItemUpdate(d *schema.ResourceData, m interface{}, c ItemHandler, r ItemHandler, prototype bool) error {
	api := m.(*zabbix.API)

	item := buildItemObject(d, api, prototype)
	item.ItemID = d.Id()

	// run custom function
	c(d, m, item)

	log.Trace("preparing item object for create/update: %#v", item)

	items := []zabbix.Item{*item}

	var err error

	if prototype {
		err = api.ProtoItemsUpdate(items)
	} else {
		err = api.ItemsUpdate(items)
	}

	if err != nil {
		return err
	}

	return resourceItemRead(d, m, r, prototype)
}

// Read Item Resource Handler
func resourceItemRead(d *schema.ResourceData, m interface{}, r ItemHandler, prototype bool) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of item with id %s", d.Id())

	var items zabbix.Items
	var err error

	params := zabbix.Params{
		"itemids":             []string{d.Id()},
		"selectPreprocessing": "extend",
		"selectTags":          "extend",
	}

	if prototype {
		params["selectDiscoveryRule"] = "extend"
		items, err = api.ProtoItemsGet(params)
	} else {
		items, err = api.ItemsGet(params)
	}

	if err != nil {
		return err
	}

	if len(items) < 1 {
		d.SetId("")
		return nil
	}
	if len(items) > 1 {
		return errors.New("multiple items found")
	}
	item := items[0]

	log.Debug("Got item: %+v", item)

	d.SetId(item.ItemID)
	d.Set("hostid", item.HostID)
	d.Set("key", item.Key)
	d.Set("name", item.Name)
	d.Set("history", item.History)
	d.Set("trends", item.Trends)
	d.Set("valuetype", ITEM_VALUE_TYPES_REV[item.ValueType])
	d.Set("preprocessor", flattenItemPreprocessors(item))
	if prototype && item.DiscoveryRule != nil {
		d.Set("ruleid", item.DiscoveryRule.ItemID)
	}

	d.Set("tag", flattenTags(item.Tags))

	// run custom
	r(d, m, &item)

	return nil
}

// Build the base Item Object
func buildItemObject(d *schema.ResourceData, api *zabbix.API, prototype bool) *zabbix.Item {
	item := zabbix.Item{
		Key:       d.Get("key").(string),
		HostID:    d.Get("hostid").(string),
		Name:      d.Get("name").(string),
		History:   d.Get("history").(string),
		Trends:    d.Get("trends").(string),
		ValueType: ITEM_VALUE_TYPES[d.Get("valuetype").(string)],
	}
	item.Preprocessors = itemGeneratePreprocessors(d)
	item.Tags = tagGenerate(d)

	if v, ok := d.GetOk("trends"); ok {
		item.Trends = v.(string)
	} else {
		if item.ValueType == zabbix.Text || item.ValueType == zabbix.Log {
			item.Trends = "0"
		} else {
			item.Trends = "365d"
		}
		d.Set("trends", item.Trends)
	}

	if prototype {
		item.RuleID = d.Get("ruleid").(string)
	}

	return &item
}

// Generate preprocessor objects
func itemGeneratePreprocessors(d *schema.ResourceData) (preprocessors zabbix.Preprocessors) {
	preprocessorCount := d.Get("preprocessor.#").(int)
	preprocessors = make(zabbix.Preprocessors, preprocessorCount)

	for i := 0; i < preprocessorCount; i++ {
		prefix := fmt.Sprintf("preprocessor.%d.", i)
		params := d.Get(prefix + "params").([]interface{})
		pstrarr := make([]string, len(params))
		for i := 0; i < len(params); i++ {
			pstrarr[i] = params[i].(string)
		}

		preprocessors[i] = zabbix.Preprocessor{
			Type:               d.Get(prefix + "type").(string),
			Params:             strings.Join(pstrarr, "\n"),
			ErrorHandler:       d.Get(prefix + "error_handler").(string),
			ErrorHandlerParams: d.Get(prefix + "error_handler_params").(string),
		}
	}

	return
}

// Generate terraform flattened form of item preprocessors
func flattenItemPreprocessors(item zabbix.Item) []interface{} {
	val := make([]interface{}, len(item.Preprocessors))
	for i := 0; i < len(item.Preprocessors); i++ {
		val[i] = map[string]interface{}{
			//"id": host.Interfaces[i].InterfaceID,
			"type":                 item.Preprocessors[i].Type,
			"error_handler":        item.Preprocessors[i].ErrorHandler,
			"error_handler_params": item.Preprocessors[i].ErrorHandlerParams,
		}
		if item.Preprocessors[i].Params != "" {
			val[i].(map[string]interface{})["params"] = strings.Split(item.Preprocessors[i].Params, "\n")
		}
	}
	return val
}

// Delete Item Resource Handler
func resourceItemDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.ItemsDeleteByIds([]string{d.Id()})
}
func resourceProtoItemDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.ProtoItemsDeleteByIds([]string{d.Id()})
}
