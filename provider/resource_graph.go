package provider

import (
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

var GRAPH_TYPE_LOOKUP = map[string]zabbix.GraphType{
	"normal":   zabbix.GraphNormal,
	"stacked":  zabbix.GraphStacked,
	"pie":      zabbix.GraphPie,
	"exploded": zabbix.GraphExploded,
}
var GRAPH_TYPE_LOOKUP_REV = map[zabbix.GraphType]string{}
var GRAPH_TYPE_LOOKUP_ARR = []string{}

var GRAPH_AXIS_LOOKUP = map[string]zabbix.GraphAxis{
	"calculated": zabbix.GraphCalculated,
	"fixed":      zabbix.GraphFixed,
	"item":       zabbix.GraphItem,
}
var GRAPH_AXIS_LOOKUP_REV = map[zabbix.GraphAxis]string{}
var GRAPH_AXIS_LOOKUP_ARR = []string{}

var _ = func() bool {
	for k, v := range GRAPH_TYPE_LOOKUP {
		GRAPH_TYPE_LOOKUP_REV[v] = k
		GRAPH_TYPE_LOOKUP_ARR = append(GRAPH_TYPE_LOOKUP_ARR, k)
	}
	for k, v := range GRAPH_AXIS_LOOKUP {
		GRAPH_AXIS_LOOKUP_REV[v] = k
		GRAPH_AXIS_LOOKUP_ARR = append(GRAPH_AXIS_LOOKUP_ARR, k)
	}
	return false
}()

// resourceGraph terraform resource handler
func resourceGraph() *schema.Resource {
	return &schema.Resource{
		Create: resourceGraphCreate,
		Read:   resourceGraphRead,
		Update: resourceGraphUpdate,
		Delete: resourceGraphDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Graph Name",
				Required:     true,
			},
			"height": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Height",
				Required:     true,
			},
			"width": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Width",
				Required:     true,
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Default:      "normal",
				Description:  "Type, one of: " + strings.Join(GRAPH_TYPE_LOOKUP_ARR, ", "),
				ValidateFunc: validation.StringInSlice(GRAPH_TYPE_LOOKUP_ARR, false),
				Optional:     true,
			},
			"percent_left": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Left percentile",
				Default:      "0",
				Optional:     true,
			},
			"percent_right": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Right percentile",
				Default:      "0",
				Optional:     true,
			},
			"3d": &schema.Schema{
				Type:        schema.TypeBool,
				Description: "Show 3d graph",
				Default:     false,
				Optional:    true,
			},
			"legend": &schema.Schema{
				Type:        schema.TypeBool,
				Description: "Show legend",
				Default:     true,
				Optional:    true,
			},
			"work_period": &schema.Schema{
				Type:        schema.TypeBool,
				Description: "Show work period",
				Default:     true,
				Optional:    true,
			},
			"ymax": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Y Axis Max",
				Default:      "100",
				Optional:     true,
			},
			"ymax_itemid": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Y Axis Max ItemId",
				Optional:     true,
			},
			"ymax_type": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Y Axis Max Type",
				Default:      "calculated",
				Optional:     true,
			},
			"ymin": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Y Axis Min",
				Default:      "0",
				Optional:     true,
			},
			"ymin_itemid": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Y Axis Min ItemId",
				Optional:     true,
			},
			"ymin_type": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Y Axis Min Type",
				Default:      "calculated",
				Optional:     true,
			},
		},
	}
}

// terraform Graph create function
func resourceGraphCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := zabbix.Graph{
		Name: d.Get("name").(string),
	}

	items := []zabbix.Graph{item}

	err := api.GraphsCreate(items)

	if err != nil {
		return err
	}

	log.Trace("created Graph: %+v", items[0])

	d.SetId(items[0].GroupID)

	return resourceGraphRead(d, m)
}

// GraphRead terraform Graph read function
func GraphRead(d *schema.ResourceData, m interface{}, params zabbix.Params) error {
	api := m.(*zabbix.API)

	Graphs, err := api.GraphsGet(params)

	if err != nil {
		return err
	}

	if len(Graphs) < 1 {
		d.SetId("")
		return nil
	}
	if len(Graphs) > 1 {
		return errors.New("multiple Graphs found")
	}
	t := Graphs[0]

	log.Debug("Got Graph: %+v", t)

	d.SetId(t.GroupID)
	d.Set("name", t.Name)

	return nil
}

// dataGraphRead terraform data resource read handler
func dataGraphRead(d *schema.ResourceData, m interface{}) error {
	return GraphRead(d, m, zabbix.Params{
		"filter": map[string]interface{}{
			"name": d.Get("name"),
		},
	})
}

// resourceGraphRead terraform resource read handler
func resourceGraphRead(d *schema.ResourceData, m interface{}) error {
	log.Debug("Lookup of Graph with id %s", d.Id())

	return GraphRead(d, m, zabbix.Params{
		"groupids": d.Id(),
	})
}

// resourceGraphUpdate terraform resource update handler
func resourceGraphUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := zabbix.Graph{
		GroupID: d.Id(),
		Name:    d.Get("name").(string),
	}

	items := []zabbix.Graph{item}

	err := api.GraphsUpdate(items)

	if err != nil {
		return err
	}

	return resourceGraphRead(d, m)
}

// resourceGraphDelete terraform resource delete handler
func resourceGraphDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.GraphsDeleteByIds([]string{d.Id()})
}
