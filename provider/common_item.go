package provider

import (
	"errors"
	"fmt"

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
		Type:         schema.TypeString,
		ValidateFunc: validation.StringInSlice(ITEM_VALUE_TYPES_ARR, false),
		Required:     true,
	},
	"preprocessor": itemPreprocessorSchema,
}

// Delay schema
var itemDelaySchema = map[string]*schema.Schema{
	"delay": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		Default:  "1m",
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
		return resourceItemCreate(d, m, c, r)
	}
}

// return a terraform UpdateFunc
func itemGetUpdateWrapper(c ItemHandler, r ItemHandler) schema.UpdateFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		return resourceItemUpdate(d, m, c, r)
	}
}

// return a terraform ReadFunc
func itemGetReadWrapper(r ItemHandler) schema.ReadFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		return resourceItemRead(d, m, r)
	}
}

// Create Item Resource Handler
func resourceItemCreate(d *schema.ResourceData, m interface{}, c ItemHandler, r ItemHandler) error {
	api := m.(*zabbix.API)

	item := buildItemObject(d)

	// run custom function
	c(d, item)

	log.Trace("preparing item object for create/update: %#v", item)

	items := []zabbix.Item{*item}

	err := api.ItemsCreate(items)

	if err != nil {
		return err
	}

	log.Trace("created item: %+v", items[0])

	d.SetId(items[0].ItemID)

	return resourceItemRead(d, m, r)
}

// Update Item Resource Handler
func resourceItemUpdate(d *schema.ResourceData, m interface{}, c ItemHandler, r ItemHandler) error {
	api := m.(*zabbix.API)

	item := buildItemObject(d)
	item.ItemID = d.Id()

	// run custom function
	c(d, item)

	log.Trace("preparing item object for create/update: %#v", item)

	items := []zabbix.Item{*item}

	err := api.ItemsUpdate(items)

	if err != nil {
		return err
	}

	return resourceItemRead(d, m, r)
}

// Read Item Resource Handler
func resourceItemRead(d *schema.ResourceData, m interface{}, r ItemHandler) error {
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

	// run custom
	r(d, &item)

	return nil
}

// Build the base Item Object
func buildItemObject(d *schema.ResourceData) *zabbix.Item {
	item := zabbix.Item{
		Key:       d.Get("key").(string),
		HostID:    d.Get("hostid").(string),
		Name:      d.Get("name").(string),
		ValueType: ITEM_VALUE_TYPES[d.Get("valuetype").(string)],
	}
	item.Preprocessors = itemGeneratePreprocessors(d)

	return &item
}

// Generate preprocessor objects
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

// Generate terraform flattened form of item preprocessors
func flattenItemPreprocessors(item zabbix.Item) []interface{} {
	val := make([]interface{}, len(item.Preprocessors))
	for i := 0; i < len(item.Preprocessors); i++ {
		val[i] = map[string]interface{}{
			//"id": host.Interfaces[i].InterfaceID,
			"type":                 item.Preprocessors[i].Type,
			"params":               item.Preprocessors[i].Params,
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
