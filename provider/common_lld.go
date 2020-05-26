package provider

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/tpretz/go-zabbix-api"
)

// eval type
var LLD_EVALTYPE = map[string]zabbix.LLDEvalType{
	"andor":  zabbix.LLDAndOr,
	"and":    zabbix.LLDAnd,
	"or":     zabbix.LLDOr,
	"custom": zabbix.LLDCustom,
}
var LLD_EVALTYPE_REV = map[zabbix.LLDEvalType]string{}
var LLD_EVALTYPE_ARR = []string{}

// generate the above structures
var _ = func() bool {
	for k, v := range LLD_EVALTYPE {
		LLD_EVALTYPE_REV[v] = k
		LLD_EVALTYPE_ARR = append(LLD_EVALTYPE_ARR, k)
	}
	return false
}()

// operator
var LLD_OPERATOR = map[string]zabbix.LLDOperatorType{
	"match":    zabbix.LLDMatch,
	"notmatch": zabbix.LLDNotMatch,
}

var LLD_OPERATOR_REV = map[zabbix.LLDOperatorType]string{}
var LLD_OPERATOR_ARR = []string{}

// generate the above structures
var _ = func() bool {
	for k, v := range LLD_OPERATOR {
		LLD_OPERATOR_REV[v] = k
		LLD_OPERATOR_ARR = append(LLD_OPERATOR_ARR, k)
	}
	return false
}()

// common schema elements for all lld types
var lldCommonSchema = map[string]*schema.Schema{
	"hostid": &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		Description:  "Host ID",
		ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9]+$"), "must be numeric"),
	},
	"delay": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Default:      "3600",
		Description:  "LLD Delay period",
	},
	"lifetime": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Default:      "30d",
		Description:  "LLD Stale Item Lifetime",
	},
	"key": &schema.Schema{
		Type:         schema.TypeString,
		Description:  "LLD KEY",
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Required:     true,
	},
	"name": &schema.Schema{
		Type:         schema.TypeString,
		Description:  "LLD Name",
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Required:     true,
	},
	"preprocessor": lldPreprocessorSchema,
	"condition":    lldFilterConditionSchema,
	"macropath":    lldMacroPathSchema,
	"evaltype": &schema.Schema{
		Type:         schema.TypeString,
		Description:  "EvalType, one of: " + strings.Join(LLD_EVALTYPE_ARR, ", "),
		ValidateFunc: validation.StringInSlice(LLD_EVALTYPE_ARR, false),
		Default:      "andor",
		Optional:     true,
	},
	"formula": &schema.Schema{
		Type:        schema.TypeString,
		Description: "Formula",
		Default:     "",
		Optional:    true,
	},
}

// Interface schema
var lldInterfaceSchema = map[string]*schema.Schema{
	"interfaceid": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Host Interface ID",
		Default:     "0",
	},
}

// Schema for preprocessor blocks
var lldPreprocessorSchema = &schema.Schema{
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

var lldMacroPathSchema = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"macro": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Macro",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"path": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Macro Path",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	},
}

// Schema for filter block
var lldFilterConditionSchema = &schema.Schema{
	Type:     schema.TypeList,
	Optional: true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"macro": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Filter Macro",
			},
			"value": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Filter Valu",
			},
			"operator": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "match",
				Description:  "Operator, one of: " + strings.Join(LLD_OPERATOR_ARR, ", "),
				ValidateFunc: validation.StringInSlice(LLD_OPERATOR_ARR, false),
			},
		},
	},
}

// Function signature for context manipulation
type LLDHandler func(*schema.ResourceData, *zabbix.LLDRule)

// return a terraform CreateFunc
func lldGetCreateWrapper(c LLDHandler, r LLDHandler) schema.CreateFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		return resourceLLDCreate(d, m, c, r)
	}
}

// return a terraform UpdateFunc
func lldGetUpdateWrapper(c LLDHandler, r LLDHandler) schema.UpdateFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		return resourceLLDUpdate(d, m, c, r)
	}
}

// return a terraform ReadFunc
func lldGetReadWrapper(r LLDHandler) schema.ReadFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		return resourceLLDRead(d, m, r)
	}
}

// Create lld Resource Handler
func resourceLLDCreate(d *schema.ResourceData, m interface{}, c LLDHandler, r LLDHandler) error {
	api := m.(*zabbix.API)

	lld := buildLLDObject(d)

	// run custom function
	c(d, lld)

	log.Trace("preparing lld object for create/update: %#v", lld)

	llds := []zabbix.LLDRule{*lld}

	err := api.LLDsCreate(llds)

	if err != nil {
		return err
	}

	log.Trace("created lld: %+v", llds[0])

	d.SetId(llds[0].ItemID)

	return resourceLLDRead(d, m, r)
}

// Update lld Resource Handler
func resourceLLDUpdate(d *schema.ResourceData, m interface{}, c LLDHandler, r LLDHandler) error {
	api := m.(*zabbix.API)

	lld := buildLLDObject(d)
	lld.ItemID = d.Id()

	// run custom function
	c(d, lld)

	log.Trace("preparing lld object for create/update: %#v", lld)

	llds := []zabbix.LLDRule{*lld}

	err := api.LLDsUpdate(llds)

	if err != nil {
		return err
	}

	return resourceLLDRead(d, m, r)
}

// Read lld Resource Handler
func resourceLLDRead(d *schema.ResourceData, m interface{}, r LLDHandler) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of lld with id %s", d.Id())

	llds, err := api.LLDsGet(zabbix.Params{
		"itemids":             []string{d.Id()},
		"selectPreprocessing": "extend",
		"selectLLDMacroPaths": "extend",
		"selectFilter":        "extend",
	})

	if err != nil {
		return err
	}

	if len(llds) < 1 {
		d.SetId("")
		return nil
	}
	if len(llds) > 1 {
		return errors.New("multiple llds found")
	}
	lld := llds[0]

	log.Debug("Got lld: %+v", lld)

	d.SetId(lld.ItemID)
	d.Set("hostid", lld.HostID)
	d.Set("key", lld.Key)
	d.Set("name", lld.Name)
	d.Set("delay", lld.Delay)
	d.Set("lifetime", lld.LifeTime)
	d.Set("evaltype", LLD_EVALTYPE_REV[lld.Filter.EvalType])
	d.Set("formula", lld.Filter.Formula)
	d.Set("condition", flattenlldConditions(lld))
	d.Set("preprocessor", flattenlldPreprocessors(lld))
	d.Set("macropath", flattenlldMacroPaths(lld))

	// run custom
	r(d, &lld)

	return nil
}

// Build the base lld Object
func buildLLDObject(d *schema.ResourceData) *zabbix.LLDRule {
	lld := zabbix.LLDRule{
		Key:      d.Get("key").(string),
		HostID:   d.Get("hostid").(string),
		Name:     d.Get("name").(string),
		Delay:    d.Get("delay").(string),
		LifeTime: d.Get("lifetime").(string),
	}

	lld.Preprocessors = lldGeneratePreprocessors(d)
	lld.MacroPaths = lldGenerateMacroPaths(d)

	lld.Filter.EvalType = LLD_EVALTYPE[d.Get("evaltype").(string)]
	lld.Filter.Formula = d.Get("formula").(string)
	lld.Filter.Conditions = lldGenerateConditions(d)

	return &lld
}

// Generate preprocessor objects
func lldGeneratePreprocessors(d *schema.ResourceData) (preprocessors zabbix.Preprocessors) {
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

// Generate macro path objects
func lldGenerateMacroPaths(d *schema.ResourceData) (paths zabbix.LLDMacroPaths) {
	set := d.Get("macropath").(*schema.Set).List()
	paths = make(zabbix.LLDMacroPaths, len(set))

	for i := 0; i < len(paths); i++ {
		current := set[i].(map[string]interface{})
		paths[i] = zabbix.LLDMacroPath{
			Macro: current["macro"].(string),
			Path:  current["path"].(string),
		}
	}

	return
}

// Generate LLD Filter Conditions
func lldGenerateConditions(d *schema.ResourceData) (conditions zabbix.LLDRuleFilterConditions) {
	conditionsCount := d.Get("condition.#").(int)
	conditions = make(zabbix.LLDRuleFilterConditions, conditionsCount)

	for i := 0; i < conditionsCount; i++ {
		prefix := fmt.Sprintf("condition.%d.", i)

		conditions[i] = zabbix.LLDRuleFilterCondition{
			Macro:    d.Get(prefix + "macro").(string),
			Value:    d.Get(prefix + "value").(string),
			Operator: LLD_OPERATOR[d.Get(prefix+"operator").(string)],
		}
		id := d.Get(prefix + "id").(string)
		if id != "" {
			conditions[i].FormulaID = id
		}
	}

	return
}

// Generate terraform flattened form of lld preprocessors
func flattenlldPreprocessors(lld zabbix.LLDRule) []interface{} {
	val := make([]interface{}, len(lld.Preprocessors))
	for i := 0; i < len(lld.Preprocessors); i++ {
		parr := strings.Split(lld.Preprocessors[i].Params, "\n")
		val[i] = map[string]interface{}{
			//"id": host.Interfaces[i].InterfaceID,
			"type":                 lld.Preprocessors[i].Type,
			"params":               parr,
			"error_handler":        lld.Preprocessors[i].ErrorHandler,
			"error_handler_params": lld.Preprocessors[i].ErrorHandlerParams,
		}
	}
	return val
}

func flattenlldMacroPaths(lld zabbix.LLDRule) *schema.Set {
	set := schema.NewSet(func(i interface{}) int {
		m := i.(map[string]interface{})
		return hashcode.String(m["macro"].(string) + "P" + m["path"].(string))
	}, []interface{}{})
	for i := 0; i < len(lld.MacroPaths); i++ {
		set.Add(map[string]interface{}{
			"macro": lld.MacroPaths[i].Macro,
			"path":  lld.MacroPaths[i].Path,
		})
	}
	return set
}

// Generate terraform flattened form of lld filter conditions
func flattenlldConditions(lld zabbix.LLDRule) []interface{} {
	val := make([]interface{}, len(lld.Filter.Conditions))
	for i := 0; i < len(lld.Filter.Conditions); i++ {
		val[i] = map[string]interface{}{
			"id":       lld.Filter.Conditions[i].FormulaID,
			"macro":    lld.Filter.Conditions[i].Macro,
			"value":    lld.Filter.Conditions[i].Value,
			"operator": LLD_OPERATOR_REV[lld.Filter.Conditions[i].Operator],
		}
	}
	return val
}

// Delete lld Resource Handler
func resourceLLDDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.LLDDeleteByIds([]string{d.Id()})
}
