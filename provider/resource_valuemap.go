package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

var VALUEMAP_MAPPING_TYPE_LOOKUP = map[string]string{
	"equal":            "0",
	"greater_or_equal": "1",
	"less_or_equal":    "2",
	"in_range":         "3",
	"regexp":           "4",
	"default":          "5",
}

var VALUEMAP_MAPPING_TYPE_LOOKUP_REV = map[string]string{
	"0": "equal",
	"1": "greater_or_equal",
	"2": "less_or_equal",
	"3": "in_range",
	"4": "regexp",
	"5": "default",
}

var VALUEMAP_MAPPING_TYPE_ARR = []string{
	"equal",
	"greater_or_equal",
	"less_or_equal",
	"in_range",
	"regexp",
	"default",
}

func resourceValueMap() *schema.Resource {
	return &schema.Resource{
		Create: resourceValueMapCreate,
		Read:   resourceValueMapRead,
		Update: resourceValueMapUpdate,
		Delete: resourceValueMapDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"hostid": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the host or template the value map belongs to",
			},
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Name of the value map",
			},
			"mappings": &schema.Schema{
				Type:        schema.TypeList,
				Required:    true,
				Description: "Value map mappings",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice(VALUEMAP_MAPPING_TYPE_ARR, false),
							Description:  "Mapping match type",
						},
						"value": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "Original value. Must be empty for type default.",
						},
						"newvalue": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "Value to which the original value is mapped to",
						},
					},
				},
			},
		},
	}
}

func dataValueMap() *schema.Resource {
	return &schema.Resource{
		Read: dataValueMapRead,

		Schema: map[string]*schema.Schema{
			"hostid": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the host or template the value map belongs to",
			},
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Name of the value map",
			},
			"mappings": &schema.Schema{
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Value map mappings",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"newvalue": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func valuemapBuildMappings(d *schema.ResourceData) zabbix.ValueMapMappings {
	raw := d.Get("mappings").([]interface{})
	mappings := make(zabbix.ValueMapMappings, len(raw))
	for i, v := range raw {
		m := v.(map[string]interface{})
		mappings[i] = zabbix.ValueMapMapping{
			Type:     zabbix.ValueMapMappingType(VALUEMAP_MAPPING_TYPE_LOOKUP[m["type"].(string)]),
			Value:    m["value"].(string),
			Newvalue: m["newvalue"].(string),
		}
	}
	return mappings
}

func valuemapFlattenMappings(mappings zabbix.ValueMapMappings) []map[string]interface{} {
	result := make([]map[string]interface{}, len(mappings))
	for i, m := range mappings {
		result[i] = map[string]interface{}{
			"type":     VALUEMAP_MAPPING_TYPE_LOOKUP_REV[string(m.Type)],
			"value":    m.Value,
			"newvalue": m.Newvalue,
		}
	}
	return result
}

func resourceValueMapCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	vm := zabbix.ValueMap{
		HostID:   d.Get("hostid").(string),
		Name:     d.Get("name").(string),
		Mappings: valuemapBuildMappings(d),
	}

	vms := zabbix.ValueMaps{vm}
	err := api.ValueMapsCreate(vms)
	if err != nil {
		return err
	}

	log.Trace("created valuemap: %+v", vms[0])
	d.SetId(vms[0].ValueMapID)

	return resourceValueMapRead(d, m)
}

func resourceValueMapRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of valuemap with id %s", d.Id())

	vm, err := api.ValueMapGetByID(d.Id())
	if err != nil {
		return err
	}
	if vm == nil {
		d.SetId("")
		return nil
	}

	d.SetId(vm.ValueMapID)
	d.Set("hostid", vm.HostID)
	d.Set("name", vm.Name)
	d.Set("mappings", valuemapFlattenMappings(vm.Mappings))

	return nil
}

func resourceValueMapUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	vm := zabbix.ValueMap{
		ValueMapID: d.Id(),
		Name:       d.Get("name").(string),
		Mappings:   valuemapBuildMappings(d),
		// hostid is NOT updatable - rejected by the API
	}

	err := api.ValueMapsUpdate(zabbix.ValueMaps{vm})
	if err != nil {
		return err
	}

	return resourceValueMapRead(d, m)
}

func resourceValueMapDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.ValueMapsDeleteByIds([]string{d.Id()})
}

func dataValueMapRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	params := zabbix.Params{
		"selectMappings": "extend",
		"filter": map[string]interface{}{
			"name": d.Get("name").(string),
		},
		"hostids": d.Get("hostid").(string),
	}

	vms, err := api.ValueMapsGet(params)
	if err != nil {
		return err
	}

	if len(vms) < 1 {
		return errors.New("no valuemap found")
	}
	if len(vms) > 1 {
		return errors.New("multiple valuemaps found")
	}

	vm := vms[0]
	d.SetId(vm.ValueMapID)
	d.Set("hostid", vm.HostID)
	d.Set("name", vm.Name)
	d.Set("mappings", valuemapFlattenMappings(vm.Mappings))

	return nil
}
