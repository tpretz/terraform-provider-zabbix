package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
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
		Description:  "Host ID",
		ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9]+$"), "must be numeric"),
	},
	"key": &schema.Schema{
		Type:         schema.TypeString,
		Description:  "Item KEY",
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Required:     true,
	},
	"name": &schema.Schema{
		Type:         schema.TypeString,
		Description:  "Item Name",
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Required:     true,
	},
	"valuetype": &schema.Schema{
		Type:         schema.TypeString,
		ValidateFunc: validation.StringInSlice(ITEM_VALUE_TYPES_ARR, false),
		Description:  "Item Value Type, one of: " + strings.Join(ITEM_VALUE_TYPES_ARR, ", "),
		Required:     true,
	},
	"preprocessor": itemPreprocessorSchema,
	"applications": &schema.Schema{
		Type:        schema.TypeSet,
		Description: "Application IDs to associate this item with",
		Elem: &schema.Schema{
			Type:         schema.TypeString,
			ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9]+$"), "must be a numeric string"),
		},
		Optional: true,
	},
}

// Delay schema
var itemDelaySchema = map[string]*schema.Schema{
	"delay": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Default:      "1m",
		Description:  "Item Delay period",
	},
}

// Interface schema
var itemInterfaceSchema = map[string]*schema.Schema{
	"interfaceid": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Host Interface ID",
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
		Description:  "LLD Rule ID",
	},
}

// Schema for preprocessor blocks
var itemPreprocessorSchema = &schema.Schema{
	Type:     schema.TypeList,
	Optional: true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Preprocessor type, zabbix identifier number",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9]+$"), "must be numeric"),
			},
			"params": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotWhiteSpace,
				},
				Optional:    true,
				Description: "Preprocessor parameters",
			},
			"error_handler": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"error_handler_params": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
		},
	},
}

// Function signature for context manipulation
type ItemHandler func(*schema.ResourceData, *zabbix.Item)

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

	item := buildItemObject(d, prototype)

	// run custom function
	c(d, item)

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

	item := buildItemObject(d, prototype)
	item.ItemID = d.Id()

	// run custom function
	c(d, item)

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
		"selectApplications":  "extend",
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
	d.Set("valuetype", ITEM_VALUE_TYPES_REV[item.ValueType])
	d.Set("preprocessor", flattenItemPreprocessors(item))
	if prototype && item.DiscoveryRule != nil {
		d.Set("ruleid", item.DiscoveryRule.ItemID)
	}

	var applications zabbix.Applications
	err = json.Unmarshal(item.Applications, &applications)
	if err != nil {
		return err
	}

	applicationSet := schema.NewSet(schema.HashString, []interface{}{})
	for _, v := range applications {
		applicationSet.Add(v.ApplicationID)
	}
	d.Set("applications", applicationSet)

	// run custom
	r(d, &item)

	return nil
}

// Build the base Item Object
func buildItemObject(d *schema.ResourceData, prototype bool) *zabbix.Item {
	item := zabbix.Item{
		Key:       d.Get("key").(string),
		HostID:    d.Get("hostid").(string),
		Name:      d.Get("name").(string),
		ValueType: ITEM_VALUE_TYPES[d.Get("valuetype").(string)],
	}
	item.Preprocessors = itemGeneratePreprocessors(d)

	text, _ := json.Marshal(d.Get("applications").(*schema.Set).List())
	raw := json.RawMessage(text)
	item.Applications = raw

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
		parr := strings.Split(item.Preprocessors[i].Params, "\n")
		val[i] = map[string]interface{}{
			//"id": host.Interfaces[i].InterfaceID,
			"type":                 item.Preprocessors[i].Type,
			"params":               parr,
			"error_handler":        item.Preprocessors[i].ErrorHandler,
			"error_handler_params": item.Preprocessors[i].ErrorHandlerParams,
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
