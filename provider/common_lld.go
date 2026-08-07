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
	"macro":        lldMacroPathSchema,
	"evaltype": &schema.Schema{
		Type:         schema.TypeString,
		Description:  "EvalType, one of: " + strings.Join(LLD_EVALTYPE_ARR, ", "),
		ValidateFunc: validation.StringInSlice(LLD_EVALTYPE_ARR, false),
		Default:      "andor",
		Optional:     true,
	},
	"formula": &schema.Schema{
		Type: schema.TypeString,
		Description: "Custom filter expression over the condition ids, e.g. \"A or (B and C)\". Only used when evaltype is " +
			"\"custom\". Zabbix renumbers the ids into the order they first appear in the formula, so write it in that " +
			"order or it will be read back rewritten.",
		Default:  "",
		Optional: true,
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

var lldValidationMacro = validation.StringMatch(regexp.MustCompile("^\\{#[A-Z][A-Z._]*\\}$"), "must be a LLD macro format")

var lldMacroPathSchema = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"macro": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Macro",
				ValidateFunc: lldValidationMacro,
			},
			"path": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Macro Path",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^\\$"), "must be a json path"),
			},
		},
	},
}

// lldConditionHash hashes a filter condition over everything the user writes:
// macro, value, operator, and - for evaltype "custom" only - the formula id.
//
// Filter conditions are unordered. Zabbix does not preserve the order they are
// submitted in, and does not even return them consistently across versions:
// 6.0 returns them in submission order while 7.2+ returns them sorted by
// formula id. For every evaltype but "custom" the server assigns the formula
// ids itself, and 7.2+ rejects the request outright if the client supplies any
// - so no ordering a caller could express survives the round trip, and this is
// a TypeSet.
//
// `id` can safely be part of the hash, unlike the computed ids on graph items
// and host interfaces, because flattenlldConditions only reports one under
// evaltype "custom" - the one case where it came from the config in the first
// place. Everything the user writes has to be in the hash or changes to it are
// silently dropped; see hashElementExcept.
func lldConditionHash(v interface{}) int {
	return hashElementExcept(v)
}

// Schema for filter block
var lldFilterConditionSchema = &schema.Schema{
	Type:        schema.TypeSet,
	Optional:    true,
	Set:         lldConditionHash,
	Description: "LLD filter conditions (unordered). With evaltype \"custom\", set `id` on each condition and reference those ids from `formula`.",
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Description: "Formula ID (A, B, C, ...). Required, and only meaningful, when evaltype is " +
					"\"custom\"; under every other evaltype Zabbix assigns the ids itself, 7.2+ rejects a " +
					"caller-supplied value, and this attribute stays empty.",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^[A-Z]+$"), "must be one or more uppercase letters, e.g. A"),
			},
			"macro": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Filter Macro",
				ValidateFunc: lldValidationMacro,
			},
			"value": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Filter Value",
				ValidateFunc: validation.StringIsNotWhiteSpace,
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
type LLDHandler func(*schema.ResourceData, interface{}, *zabbix.LLDRule)

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
	c(d, m, lld)

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
	c(d, m, lld)

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
	d.Set("macro", flattenlldMacroPaths(lld))

	// run custom
	r(d, m, &lld)

	return nil
}

// lldZeroDelaySchema replaces lldDelaySchema for discovery-rule types that are
// not polled: dependent rules are driven by their master item, and trapper rules
// receive pushed data. Zabbix requires delay == 0 for both and rejects anything
// else, so inheriting lldDelaySchema's 3600 default made those resources fail on
// create unless the user knew to write delay = "0" - which is not discoverable.
//
// delay stays in the schema rather than being omitted, because the shared LLD
// read path calls d.Set("delay", ...) unconditionally and helper/schema panics
// on a key the schema does not declare. Pinning the value is what makes that
// safe while still rejecting what Zabbix would reject.
var lldZeroDelaySchema = map[string]*schema.Schema{
	"delay": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Default:      "0",
		ValidateFunc: validation.StringInSlice([]string{"0"}, false),
		Description:  "LLD Delay period, must be 0 for this discovery rule type",
	},
}

// lldDelaySchema is separate from lldCommonSchema because dependent LLD rules
// are driven by their master item rather than polled: Zabbix requires delay == 0
// for them and rejects any other value. This mirrors how itemDelaySchema is kept
// out of resourceItemDependent.
var lldDelaySchema = map[string]*schema.Schema{
	"delay": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Default:      "3600",
		Description:  "LLD Delay period",
	},
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
	if paths := lldGenerateMacroPaths(d); len(paths) > 0 {
		lld.MacroPaths = &paths
	}

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
	set := d.Get("macro").(*schema.Set).List()
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
//
// formulaid is only ever sent for evaltype "custom", where the formula
// references conditions by it and Zabbix requires one per condition. Under any
// other evaltype Zabbix assigns the formula ids itself and 7.2+ fails the
// whole call with `value must be empty` if the client echoes back the ids it
// read a moment earlier - which is exactly what an update would otherwise do.
func lldGenerateConditions(d *schema.ResourceData) (conditions zabbix.LLDRuleFilterConditions) {
	set := d.Get("condition").(*schema.Set).List()
	conditions = make(zabbix.LLDRuleFilterConditions, len(set))
	custom := LLD_EVALTYPE[d.Get("evaltype").(string)] == zabbix.LLDCustom

	for i, raw := range set {
		m := raw.(map[string]interface{})

		conditions[i] = zabbix.LLDRuleFilterCondition{
			Macro:    m["macro"].(string),
			Value:    m["value"].(string),
			Operator: LLD_OPERATOR[m["operator"].(string)],
		}
		if id, _ := m["id"].(string); custom && id != "" {
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
		return schema.HashString(m["macro"].(string) + "P" + m["path"].(string))
	}, []interface{}{})
	if lld.MacroPaths == nil {
		return set
	}
	for _, p := range *lld.MacroPaths {
		set.Add(map[string]interface{}{
			"macro": p.Macro,
			"path":  p.Path,
		})
	}
	return set
}

// Generate terraform flattened form of lld filter conditions
//
// The formula id is only reported back under evaltype "custom". Anywhere else
// it is a value Zabbix invented for its own use, which the config cannot set
// and the provider must not send; reporting it would put a value in state that
// no configuration can ever match.
func flattenlldConditions(lld zabbix.LLDRule) []interface{} {
	custom := lld.Filter.EvalType == zabbix.LLDCustom

	val := make([]interface{}, len(lld.Filter.Conditions))
	for i := 0; i < len(lld.Filter.Conditions); i++ {
		formulaID := ""
		if custom {
			formulaID = lld.Filter.Conditions[i].FormulaID
		}
		val[i] = map[string]interface{}{
			"id":       formulaID,
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
