package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

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
