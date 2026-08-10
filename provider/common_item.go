package provider

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
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

// Preprocessing step types.
//
// Zabbix numbers these internally and the numbers are what the API wants, but
// a number is not an interface: `type = "12"` says nothing, `type =
// "jsonpath"` says what the step does. Every other enum in the provider maps a
// name to Zabbix's code through the _LOOKUP/_REV/_ARR idiom (CLAUDE.md
// § "Shared schema helpers & the lookup-table idiom") and this is now no
// exception.
//
// The list was taken from the item object documentation for each supported
// version and then confirmed against live 6.0.48, 7.0.29, 7.4.13 and 8.0
// servers by attempting an item.create with every code from 1 to 35 and
// reading which ones came back rejected on the `type` property specifically:
//
//	6.0        1-27
//	7.0/7.4/8.0  1-30
//
// 28 and 29 arrived in 6.4 and 30 in 7.0, so the three of them are gated --
// see PREPROC_MIN_VERSION. The discovery-rule list is a *different, smaller*
// list and lives in common_lld.go; do not assume the two track each other.
var PREPROC_LOOKUP = map[string]string{
	"multiplier":                  "1",
	"rtrim":                       "2",
	"ltrim":                       "3",
	"trim":                        "4",
	"regex":                       "5",
	"bool_to_decimal":             "6",
	"octal_to_decimal":            "7",
	"hex_to_decimal":              "8",
	"simple_change":               "9",
	"change_per_second":           "10",
	"xml_xpath":                   "11",
	"jsonpath":                    "12",
	"in_range":                    "13",
	"matches_regex":               "14",
	"not_matches_regex":           "15",
	"check_json_error":            "16",
	"check_xml_error":             "17",
	"check_regex_error":           "18",
	"discard_unchanged":           "19",
	"discard_unchanged_heartbeat": "20",
	"javascript":                  "21",
	"prometheus_pattern":          "22",
	"prometheus_to_json":          "23",
	"csv_to_json":                 "24",
	"replace":                     "25",
	"check_unsupported":           "26",
	"xml_to_json":                 "27",
	"snmp_walk_value":             "28",
	"snmp_walk_to_json":           "29",
	"snmp_get_value":              "30",
}
var PREPROC_LOOKUP_REV = map[string]string{}
var PREPROC_LOOKUP_ARR = []string{}

// PREPROC_MIN_VERSION is the first Zabbix version that has each step type, for
// the types that are not in 6.0. A type missing from this map exists on every
// server the provider supports.
//
// This cannot be enforced in the schema -- a ValidateFunc runs before the
// provider has spoken to anything and has no idea what it is talking to -- so
// the check lives in the create/update path, where the version is known.
// Leaving it to the server is not good enough either: below 7.2 an unknown
// preprocessing type is reported as `unexpected value "30"`, a number the user
// never wrote.
var PREPROC_MIN_VERSION = map[string]preprocGate{
	"snmp_walk_value":   {zabbix.V64, "6.4"},
	"snmp_walk_to_json": {zabbix.V64, "6.4"},
	"snmp_get_value":    {zabbix.V70, "7.0"},
}

// preprocGate is a minimum version with the human form of it to put in the
// error message; Config.Version is an encoded integer and "60400" is not a
// version number anybody recognises.
type preprocGate struct {
	version int
	name    string
}

// generate the above structures
var _ = func() bool {
	for k, v := range PREPROC_LOOKUP {
		PREPROC_LOOKUP_REV[v] = k
		PREPROC_LOOKUP_ARR = append(PREPROC_LOOKUP_ARR, k)
	}
	// map iteration order is random; sort so that the generated documentation
	// and validation messages are stable between builds
	sort.Strings(PREPROC_LOOKUP_ARR)
	return false
}()

// preprocTypeList renders a preprocessing lookup for a schema Description,
// ordered by Zabbix's code rather than by name because that is the order the
// upstream documentation uses and the order a reader comparing the two will
// expect. Gated types carry the version they arrived in.
func preprocTypeList(lookup map[string]string, gates map[string]preprocGate) string {
	names := make([]string, 0, len(lookup))
	for name := range lookup {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		a, _ := strconv.Atoi(lookup[names[i]])
		b, _ := strconv.Atoi(lookup[names[j]])
		return a < b
	})

	parts := make([]string, len(names))
	for i, name := range names {
		if g, ok := gates[name]; ok {
			parts[i] = fmt.Sprintf("`%s` (%s, Zabbix %s and later)", name, lookup[name], g.name)
			continue
		}
		parts[i] = fmt.Sprintf("`%s` (%s)", name, lookup[name])
	}
	return strings.Join(parts, ", ")
}

// preprocessorTypeValidator accepts a step-type name, or -- deprecated -- the
// numeric code it stands for.
//
// The numeric form is here for one release only. Every v0.x configuration in
// existence writes `type = "12"`, and v2.0.0 already asks enough of them; the
// name is canonical and the read path always writes it back, so a numeric
// configuration is rewritten to the name on the first apply and the user is
// warned once per step in the meantime.
//
// The rejection message is built by hand rather than by calling
// validation.StringInSlice, and its wording is deliberately identical:
// schema_enum_test.go discovers the provider's enums by feeding every
// validator a value nothing accepts and parsing the permitted set back out of
// "to be one of [...]", and acc_negative_test.go matches the same phrase. An
// enum that words its rejection differently drops silently out of both.
func preprocessorTypeValidator(lookup, rev map[string]string, arr []string) schema.SchemaValidateFunc {
	return func(i interface{}, k string) ([]string, []error) {
		v, ok := i.(string)
		if !ok {
			return nil, []error{fmt.Errorf("expected type of %s to be string", k)}
		}
		if _, ok := lookup[v]; ok {
			return nil, nil
		}
		if name, ok := rev[v]; ok {
			return []string{fmt.Sprintf(
				"%s: %q is Zabbix's internal code for the %q preprocessing step. The "+
					"numeric form is deprecated and will be removed in the next major "+
					"release of this provider; write %q instead. It will be rewritten to "+
					"%q in state on the next apply.", k, v, name, name, name)}, nil
		}
		return nil, []error{fmt.Errorf("expected %s to be one of %q, got %s", k, arr, v)}
	}
}

// preprocessorTypeStateFunc normalises the deprecated numeric form to the name
// before it reaches state, which is what makes a v0.x configuration converge:
// the diff is computed against the normalised value, so `type = "12"` against
// a state holding "jsonpath" plans empty rather than for ever.
func preprocessorTypeStateFunc(rev map[string]string) schema.SchemaStateFunc {
	return func(i interface{}) string {
		v, ok := i.(string)
		if !ok {
			return ""
		}
		if name, ok := rev[v]; ok {
			return name
		}
		return v
	}
}

// resolvePreprocessorType turns a configured step type into the numeric code
// Zabbix wants, refusing the types this server does not have.
func resolvePreprocessorType(v string, lookup, rev map[string]string, gates map[string]preprocGate, version int) (string, error) {
	name := v
	if n, ok := rev[v]; ok {
		name = n // deprecated numeric form
	}

	code, ok := lookup[name]
	if !ok {
		return "", fmt.Errorf("unknown preprocessing step type %q", v)
	}
	if g, ok := gates[name]; ok && version < g.version {
		return "", fmt.Errorf(
			"preprocessing step type %q (Zabbix code %s) requires Zabbix %s or later; this server is %s",
			name, code, g.name, zabbixVersionString(version))
	}
	return code, nil
}

// flattenPreprocessorType is the read half: Zabbix's code back to the name.
// An unrecognised code is passed through as-is rather than read back as the
// empty string, so that a server newer than this provider produces a diff a
// human can understand instead of an attribute that has silently vanished.
func flattenPreprocessorType(code string, rev map[string]string) string {
	if name, ok := rev[code]; ok {
		return name
	}
	return code
}

// zabbixVersionString undoes the encoding parseVersionString applies, for
// error messages. 60048 is not something to show a user.
func zabbixVersionString(v int) string {
	if v == 0 {
		return "of an unknown version"
	}
	return fmt.Sprintf("%d.%d.%d", v/10000, (v/100)%100, v%100)
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
				ValidateFunc: preprocessorTypeValidator(PREPROC_LOOKUP, PREPROC_LOOKUP_REV, PREPROC_LOOKUP_ARR),
				StateFunc:    preprocessorTypeStateFunc(PREPROC_LOOKUP_REV),
			},
			"params": &schema.Schema{
				Type: schema.TypeList,
				// deliberately unvalidated. A preprocessing step's parameters
				// are positional slots in one newline-separated string, and an
				// empty slot is a real value: `prometheus_pattern` with output
				// "value" is stored by every supported version as
				// "<pattern>\nvalue\n" and read back as three parameters, the
				// third empty. StringIsNotWhiteSpace here refused to let that
				// be written, so the attribute could not be made to match what
				// the server was always going to return and the resource sat in
				// a diff no apply could clear. Zabbix validates these itself,
				// per type, and says something far more useful than "must not
				// be empty" when they are wrong.
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: preprocessorParamsDescription,
			},
			"error_handler": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				// "0" (report the error), not "": Zabbix requires this property on
				// every preprocessing step and rejects an empty string outright --
				// 6.0 with "missing parameters: error_handler_params", 7.4 with
				// "an integer is expected". Defaulting to "" made a preprocessor
				// block that omitted error_handler fail on create, on every version.
				Default:     "0",
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

	preprocessorParamsDescription = "Parameters for the step, one element per line Zabbix " +
		"expects. Which parameters apply, and how many, depends entirely on `type`. They are " +
		"positional, so an empty string is a meaningful value: `prometheus_pattern` with " +
		"output `value` is stored and returned by Zabbix as three parameters, the third empty."

	preprocessorErrorHandlerDescription = "What to do when this step fails, as Zabbix's " +
		"numeric code: `0` discard the value and report the error (the default), `1` discard " +
		"the value silently, `2` set the value in `error_handler_params`, `3` report the " +
		"error text in `error_handler_params`."

	preprocessorErrorHandlerParamsDescription = "Value or error text used by `error_handler` " +
		"codes `2` and `3`. Ignored otherwise."

	// the numeric-compatibility paragraph, worded once and appended to both
	// the item and the discovery-rule description
	preprocessorTypeCompatNote = " Zabbix's numeric code is accepted too, for compatibility " +
		"with provider v0.x configurations — `\"12\"` means `jsonpath` — but it is " +
		"deprecated, warns on every plan, and will be removed in the next major release. " +
		"The name is canonical: a numeric configuration is rewritten to the name in state " +
		"on the first apply, so the plan after that is empty."
)

// preprocessorTypeDescription is built from PREPROC_LOOKUP rather than written
// out, so that adding a type cannot leave the published documentation behind.
// TestEnumDescriptionsListValues enforces the same thing from the other side.
var preprocessorTypeDescription = "Preprocessing step type, one of: " +
	preprocTypeList(PREPROC_LOOKUP, PREPROC_MIN_VERSION) + "." + preprocessorTypeCompatNote

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

	item, err := buildItemObject(d, api, prototype)
	if err != nil {
		return err
	}

	// run custom function
	c(d, m, item)

	log.Trace("preparing item object for create/update: %#v", item)

	items := []zabbix.Item{*item}

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

	item, err := buildItemObject(d, api, prototype)
	if err != nil {
		return err
	}
	item.ItemID = d.Id()

	// run custom function
	c(d, m, item)

	log.Trace("preparing item object for create/update: %#v", item)

	items := []zabbix.Item{*item}

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
func buildItemObject(d *schema.ResourceData, api *zabbix.API, prototype bool) (*zabbix.Item, error) {
	item := zabbix.Item{
		Key:       d.Get("key").(string),
		HostID:    d.Get("hostid").(string),
		Name:      d.Get("name").(string),
		History:   d.Get("history").(string),
		Trends:    d.Get("trends").(string),
		ValueType: ITEM_VALUE_TYPES[d.Get("valuetype").(string)],
	}
	preprocessors, err := itemGeneratePreprocessors(d, api)
	if err != nil {
		return nil, err
	}
	item.Preprocessors = preprocessors
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

	return &item, nil
}

// Generate preprocessor objects
func itemGeneratePreprocessors(d *schema.ResourceData, api *zabbix.API) (zabbix.Preprocessors, error) {
	preprocessorCount := d.Get("preprocessor.#").(int)
	preprocessors := make(zabbix.Preprocessors, preprocessorCount)

	for i := 0; i < preprocessorCount; i++ {
		prefix := fmt.Sprintf("preprocessor.%d.", i)
		params := d.Get(prefix + "params").([]interface{})
		pstrarr := make([]string, len(params))
		for i := 0; i < len(params); i++ {
			// not params[i].(string): an empty parameter -- a real, meaningful
			// positional slot, see the schema comment -- comes back from the
			// field reader as a nil interface rather than as "", and the
			// unchecked assertion panicked the plugin outright. It was
			// unreachable only for as long as the params validator refused to
			// let an empty one be written.
			s, _ := params[i].(string)
			pstrarr[i] = s
		}

		code, err := resolvePreprocessorType(
			d.Get(prefix+"type").(string),
			PREPROC_LOOKUP, PREPROC_LOOKUP_REV, PREPROC_MIN_VERSION, api.Config.Version)
		if err != nil {
			return nil, fmt.Errorf("preprocessor %d: %w", i, err)
		}

		preprocessors[i] = zabbix.Preprocessor{
			Type:               code,
			Params:             strings.Join(pstrarr, "\n"),
			ErrorHandler:       d.Get(prefix + "error_handler").(string),
			ErrorHandlerParams: d.Get(prefix + "error_handler_params").(string),
		}
	}

	return preprocessors, nil
}

// Generate terraform flattened form of item preprocessors
func flattenItemPreprocessors(item zabbix.Item) []interface{} {
	val := make([]interface{}, len(item.Preprocessors))
	for i := 0; i < len(item.Preprocessors); i++ {
		val[i] = map[string]interface{}{
			//"id": host.Interfaces[i].InterfaceID,
			"type":                 flattenPreprocessorType(item.Preprocessors[i].Type, PREPROC_LOOKUP_REV),
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
