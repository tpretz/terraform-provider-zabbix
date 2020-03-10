package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

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
	},
}

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

func resourceItemDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.ItemsDeleteByIds([]string{d.Id()})
}
