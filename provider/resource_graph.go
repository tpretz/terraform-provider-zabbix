package provider

import (
	"errors"
	"fmt"
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
	"calculated": zabbix.GraphAxisCalculated,
	"fixed":      zabbix.GraphAxisFixed,
	"item":       zabbix.GraphAxisItem,
}
var GRAPH_AXIS_LOOKUP_REV = map[zabbix.GraphAxis]string{}
var GRAPH_AXIS_LOOKUP_ARR = []string{}

var GRAPH_FUNC_LOOKUP = map[string]zabbix.GraphItemFunc{
	"min":     zabbix.GraphItemMin,
	"average": zabbix.GraphItemAvg,
	"max":     zabbix.GraphItemMax,
	"all":     zabbix.GraphItemAll,
	"last":    zabbix.GraphItemLast,
}
var GRAPH_FUNC_LOOKUP_REV = map[zabbix.GraphItemFunc]string{}
var GRAPH_FUNC_LOOKUP_ARR = []string{}

var GRAPH_DRAW_LOOKUP = map[string]zabbix.GraphItemDraw{
	"line":     zabbix.GraphItemLine,
	"filled":   zabbix.GraphItemFilled,
	"bold":     zabbix.GraphItemBold,
	"dot":      zabbix.GraphItemDot,
	"dashed":   zabbix.GraphItemDashed,
	"gradient": zabbix.GraphItemGradient,
}
var GRAPH_DRAW_LOOKUP_REV = map[zabbix.GraphItemDraw]string{}
var GRAPH_DRAW_LOOKUP_ARR = []string{}

var GRAPH_ITYPE_LOOKUP = map[string]zabbix.GraphItemType{
	"simple": zabbix.GraphItemSimple,
	"sum":    zabbix.GraphItemSum,
}
var GRAPH_ITYPE_LOOKUP_REV = map[zabbix.GraphItemType]string{}
var GRAPH_ITYPE_LOOKUP_ARR = []string{}

var GRAPH_SIDE_LOOKUP = map[string]zabbix.GraphItemSide{
	"left":  zabbix.GraphItemLeft,
	"right": zabbix.GraphItemRight,
}
var GRAPH_SIDE_LOOKUP_REV = map[zabbix.GraphItemSide]string{}
var GRAPH_SIDE_LOOKUP_ARR = []string{}

var _ = func() bool {
	for k, v := range GRAPH_TYPE_LOOKUP {
		GRAPH_TYPE_LOOKUP_REV[v] = k
		GRAPH_TYPE_LOOKUP_ARR = append(GRAPH_TYPE_LOOKUP_ARR, k)
	}
	for k, v := range GRAPH_AXIS_LOOKUP {
		GRAPH_AXIS_LOOKUP_REV[v] = k
		GRAPH_AXIS_LOOKUP_ARR = append(GRAPH_AXIS_LOOKUP_ARR, k)
	}
	for k, v := range GRAPH_FUNC_LOOKUP {
		GRAPH_FUNC_LOOKUP_REV[v] = k
		GRAPH_FUNC_LOOKUP_ARR = append(GRAPH_FUNC_LOOKUP_ARR, k)
	}
	for k, v := range GRAPH_DRAW_LOOKUP {
		GRAPH_DRAW_LOOKUP_REV[v] = k
		GRAPH_DRAW_LOOKUP_ARR = append(GRAPH_DRAW_LOOKUP_ARR, k)
	}
	for k, v := range GRAPH_ITYPE_LOOKUP {
		GRAPH_ITYPE_LOOKUP_REV[v] = k
		GRAPH_ITYPE_LOOKUP_ARR = append(GRAPH_ITYPE_LOOKUP_ARR, k)
	}
	for k, v := range GRAPH_SIDE_LOOKUP {
		GRAPH_SIDE_LOOKUP_REV[v] = k
		GRAPH_SIDE_LOOKUP_ARR = append(GRAPH_SIDE_LOOKUP_ARR, k)
	}
	return false
}()

var schemaGraphItem = &schema.Schema{
	Type:     schema.TypeList,
	Required: true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"color": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "color",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"itemid": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "itemid",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"function": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "min",
				Description:  "Function, one of: " + strings.Join(GRAPH_FUNC_LOOKUP_ARR, ", "),
				ValidateFunc: validation.StringInSlice(GRAPH_FUNC_LOOKUP_ARR, false),
			},
			"drawtype": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "line",
				Description:  "Draw Type, one of: " + strings.Join(GRAPH_DRAW_LOOKUP_ARR, ", "),
				ValidateFunc: validation.StringInSlice(GRAPH_DRAW_LOOKUP_ARR, false),
			},
			"sortorder": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "sort order",
				Default:      "0",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "simple",
				Description:  "Type, one of: " + strings.Join(GRAPH_ITYPE_LOOKUP_ARR, ", "),
				ValidateFunc: validation.StringInSlice(GRAPH_ITYPE_LOOKUP_ARR, false),
			},
			"yaxis_side": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "left",
				Description:  "Y Axis Side, one of: " + strings.Join(GRAPH_SIDE_LOOKUP_ARR, ", "),
				ValidateFunc: validation.StringInSlice(GRAPH_SIDE_LOOKUP_ARR, false),
			},
		},
	},
}

var schemaGraph = map[string]*schema.Schema{
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
	"do3d": &schema.Schema{
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
		Default:      "calculated",
		Optional:     true,
		Description:  "Y Axis Max Type, one of: " + strings.Join(GRAPH_AXIS_LOOKUP_ARR, ", "),
		ValidateFunc: validation.StringInSlice(GRAPH_AXIS_LOOKUP_ARR, false),
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
		Default:      "calculated",
		Optional:     true,
		Description:  "Y Axis Min Type, one of: " + strings.Join(GRAPH_AXIS_LOOKUP_ARR, ", "),
		ValidateFunc: validation.StringInSlice(GRAPH_AXIS_LOOKUP_ARR, false),
	},
	"item": schemaGraphItem,
}

// resourceGraph terraform resource handler
func resourceGraph() *schema.Resource {
	return &schema.Resource{
		Create: resourceGraphCreate(false),
		Read:   resourceGraphRead(false),
		Update: resourceGraphUpdate(false),
		Delete: resourceGraphDelete(false),
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: schemaGraph,
	}
}
func resourceProtoGraph() *schema.Resource {
	return &schema.Resource{
		Create: resourceGraphCreate(true),
		Read:   resourceGraphRead(true),
		Update: resourceGraphUpdate(true),
		Delete: resourceGraphDelete(true),
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: schemaGraph,
	}
}

// terraform Graph create function
func resourceGraphCreate(prototype bool) schema.CreateFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		api := m.(*zabbix.API)

		item := buildGraphObject(d)

		items := []zabbix.Graph{item}

		var err error
		if prototype {
			err = api.GraphProtosCreate(items)
		} else {
			err = api.GraphsCreate(items)
		}

		if err != nil {
			return err
		}

		log.Trace("created Graph: %+v", items[0])

		d.SetId(items[0].GraphID)

		return resourceGraphRead(prototype)(d, m)
	}
}

func buildGraphObject(d *schema.ResourceData) zabbix.Graph {
	item := zabbix.Graph{
		Name:           d.Get("name").(string),
		Height:         d.Get("height").(string),
		Width:          d.Get("width").(string),
		Type:           GRAPH_TYPE_LOOKUP[d.Get("type").(string)],
		PercentLeft:    d.Get("percent_left").(string),
		PercentRight:   d.Get("percent_right").(string),
		Show3d:         "0",
		ShowLegend:     "0",
		ShowWorkPeriod: "0",
		YMax:           d.Get("ymax").(string),
		YMaxItemId:     d.Get("ymax_itemid").(string),
		YMaxType:       GRAPH_AXIS_LOOKUP[d.Get("ymax_type").(string)],
		YMin:           d.Get("ymin").(string),
		YMinItemId:     d.Get("ymin_itemid").(string),
		YMinType:       GRAPH_AXIS_LOOKUP[d.Get("ymin_type").(string)],
	}
	//item.GItems = []
	if d.Get("do3d").(bool) {
		item.Show3d = "1"
	}
	if d.Get("legend").(bool) {
		item.ShowLegend = "1"
	}
	if d.Get("work_period").(bool) {
		item.ShowWorkPeriod = "1"
	}

	item.GraphItems = buildGraphItems(d)

	return item
}

func buildGraphItems(d *schema.ResourceData) (els zabbix.GraphItems) {
	count := d.Get("item.#").(int)
	els = make(zabbix.GraphItems, count)

	for i := 0; i < count; i++ {
		prefix := fmt.Sprintf("item.%d.", i)

		els[i] = zabbix.GraphItem{
			Color:     d.Get(prefix + "color").(string),
			ItemID:    d.Get(prefix + "itemid").(string),
			CalcFunc:  GRAPH_FUNC_LOOKUP[d.Get(prefix+"function").(string)],
			DrawType:  GRAPH_DRAW_LOOKUP[d.Get(prefix+"drawtype").(string)],
			SortOrder: d.Get(prefix + "sortorder").(string),
			Type:      GRAPH_ITYPE_LOOKUP[d.Get(prefix+"type").(string)],
			YAxisSide: GRAPH_SIDE_LOOKUP[d.Get(prefix+"yaxis_side").(string)],
		}

		// if we have an id (i.e an update)
		if str := d.Get(prefix + "id").(string); str != "" {
			els[i].GItemID = str
		}
	}

	return
}

func flattenGraphItems(items zabbix.GraphItems) []interface{} {
	val := make([]interface{}, len(items))
	for i := 0; i < len(items); i++ {
		val[i] = map[string]interface{}{
			"id":         items[i].GItemID,
			"color":      items[i].Color,
			"itemid":     items[i].ItemID,
			"function":   GRAPH_FUNC_LOOKUP_REV[items[i].CalcFunc],
			"drawtype":   GRAPH_DRAW_LOOKUP_REV[items[i].DrawType],
			"sortorder":  items[i].SortOrder,
			"type":       GRAPH_ITYPE_LOOKUP_REV[items[i].Type],
			"yaxis_side": GRAPH_SIDE_LOOKUP_REV[items[i].YAxisSide],
		}
	}
	return val
}

// resourceGraphRead terraform resource read handler
func resourceGraphRead(prototype bool) schema.ReadFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		api := m.(*zabbix.API)

		log.Debug("Lookup of Graph with id %s", d.Id())
		params := zabbix.Params{
			"graphids":         d.Id(),
			"selectGraphItems": "extend",
		}

		var graphs zabbix.Graphs
		var err error

		if prototype {
			graphs, err = api.GraphProtosGet(params)
		} else {
			graphs, err = api.GraphsGet(params)
		}

		if err != nil {
			return err
		}

		if len(graphs) < 1 {
			d.SetId("")
			return nil
		}
		if len(graphs) > 1 {
			return errors.New("multiple Graphs found")
		}
		t := graphs[0]

		log.Debug("Got Graph: %+v", t)

		d.SetId(t.GraphID)
		d.Set("name", t.Name)
		d.Set("height", t.Height)
		d.Set("width", t.Width)
		d.Set("type", GRAPH_TYPE_LOOKUP_REV[t.Type])
		d.Set("percent_left", t.PercentLeft)
		d.Set("percent_right", t.PercentRight)
		d.Set("do3d", t.Show3d == "1")
		d.Set("legend", t.ShowLegend == "1")
		d.Set("work_period", t.ShowWorkPeriod == "1")
		d.Set("ymax", t.YMax)
		d.Set("ymax_itemid", t.YMaxItemId)
		d.Set("ymax_type", GRAPH_AXIS_LOOKUP_REV[t.YMaxType])
		d.Set("ymin", t.YMin)
		d.Set("ymin_itemid", t.YMinItemId)
		d.Set("ymin_type", GRAPH_AXIS_LOOKUP_REV[t.YMinType])

		d.Set("item", flattenGraphItems(t.GraphItems))

		return nil
	}
}

// resourceGraphUpdate terraform resource update handler
func resourceGraphUpdate(prototype bool) schema.UpdateFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		api := m.(*zabbix.API)

		item := buildGraphObject(d)

		items := []zabbix.Graph{item}

		var err error

		if prototype {
			err = api.GraphProtosUpdate(items)
		} else {
			err = api.GraphsUpdate(items)
		}

		if err != nil {
			return err
		}

		return resourceGraphRead(prototype)(d, m)
	}
}

// resourceGraphDelete terraform resource delete handler
func resourceGraphDelete(prototype bool) schema.DeleteFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		api := m.(*zabbix.API)

		if prototype {
			return api.GraphProtosDeleteByIds([]string{d.Id()})
		}

		return api.GraphsDeleteByIds([]string{d.Id()})
	}
}
