package provider

import (
	"context"
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
		Description:  "How long Zabbix keeps hourly trends for this item, as a time suffix string, e.g. \"365d\". \"0\" disables trends. Derived from `valuetype` when it is not given: \"0\" for the non-numeric types (character, log, text), which is the only value Zabbix accepts for those, and \"365d\" for float and unsigned. The derived value is shown in the plan rather than only after apply, and changing `valuetype` across that boundary re-derives it; changing it within a class (unsigned to float, text to log) leaves the stored value alone. Deleting the line otherwise changes nothing: Terraform keeps the last value it read, and the way back to the derived default is to write it out",
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
	"units": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		// deliberately unvalidated. Zabbix accepts any string here -- it is a
		// label, not an enum -- and StringIsNotWhiteSpace would forbid the
		// empty string, which is both the default and the only way back.
		Description: itemUnitsDescription,
	},
	"description": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Free-text description of the item, shown in the frontend. Has no effect on collection",
	},
	"preprocessor": itemPreprocessorSchema,
	"tag":          tagSetSchema,
}

// itemUnitsDescription is worded once rather than inline because it is the
// densest attribute description in the item schema and every clause in it was
// measured. See the note on zabbix.Item.Units for the probe results.
const itemUnitsDescription = "Unit symbol shown after the value in the frontend, e.g. `B`, `Bps`, " +
	"`%`, `s`. Zabbix scales the number to the unit automatically -- 1048576 with `B` is displayed " +
	"as \"1 MB\" -- and two units are special-cased rather than scaled: `unixtime` renders the value " +
	"as a date and time, and `uptime` as a duration. A leading `!` suppresses the scaling and shows " +
	"the raw number with the unit after it, so `!B` displays 1048576 as \"1048576 B\". " +
	"Only numeric items may carry a unit: from Zabbix 7.0 the server rejects a non-empty `units` on " +
	"a character, log or text item with `value must be empty`, while 6.0 accepts and stores one on " +
	"any value type. Leave it empty for no unit"

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
		Description: "ID of the host interface to poll through. \"0\", the default, means no interface: it is the only value a template accepts, and it is what item types Zabbix does not poll through an interface use. An item on a host that has interfaces must name one of them -- Zabbix rejects \"0\" there, and rejects the property being omitted as well",
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
	d.Set("units", item.Units)
	d.Set("description", item.Description)
	d.Set("preprocessor", flattenItemPreprocessors(item))
	if prototype && item.DiscoveryRule != nil {
		d.Set("ruleid", item.DiscoveryRule.ItemID)
	}

	d.Set("tag", flattenTags(item.Tags))

	// run custom
	r(d, m, &item)

	return nil
}

// itemTrendless reports whether Zabbix keeps no trends at all for a value
// type. Trends are hourly minimum/average/maximum, so only the two numeric
// types have anything to put in them: character, log and text are all
// trendless, and **character is the one this codebase had wrong**.
//
// Probed with item.create on all four servers, trends omitted and then "365d":
//
//	value type   omitted   "365d"
//	float        365d      365d
//	unsigned     365d      365d
//	character    0         6.0 stores 0; 7.0/7.4/8.0 reject it
//	log          0         same
//	text         0         same
//
// Only text and log were treated as trendless before, so a character item
// with no `trends` in the configuration had "365d" derived for it and
// **could not be created at all from 7.0** -- Zabbix answered `Invalid
// parameter "/1/trends": value must be 0` for an attribute the user had never
// written. On 6.0 it was accepted and silently stored as 0. The suite missed
// it because its one character-valued fixture inherits trends "0" from an
// earlier step.
func itemTrendless(vt zabbix.ValueType) bool {
	return vt == zabbix.Character || vt == zabbix.Log || vt == zabbix.Text
}

// itemDerivedTrends is the `trends` a configuration that does not mention it
// gets: Zabbix's own default of "365d", or "0" where the value type keeps no
// trends. It is the provider's derivation rather than the server's, because
// the server only applies it on create -- see itemTrendsCustomizeDiff.
func itemDerivedTrends(vt zabbix.ValueType) string {
	if itemTrendless(vt) {
		return "0"
	}
	return "365d"
}

// itemTrendsCustomizeDiff puts the derived `trends` into the *plan*, so that a
// configuration which does not mention the attribute shows what it is going to
// get instead of "(known after apply)", and so that a value type change shows
// the trends change it drags along with it.
//
// `trends` is Optional+Computed because its default follows `valuetype` and no
// single Default: can express that (R2, acc_removal_test.go). The flag's
// contract is "if the configuration is silent, keep what the provider last
// returned", and for an attribute derived from *server* state that is exactly
// right -- re-deriving would clobber a value the user owns. This one is
// different: it is derived from another attribute of the same resource, one
// the configuration itself supplies, so the derivation can be re-run without
// consulting anything the user might have set behind Terraform's back.
//
// It fires in two cases only, and the narrowness is the point:
//
//   - on create, where there is no prior value to clobber;
//   - on a `valuetype` change that crosses the numeric/non-numeric boundary,
//     where the stored value was derived from a value type that is going away.
//
// Everything else is left alone. In particular a value type change *within* a
// class -- unsigned to float, text to character -- keeps whatever trends the
// item has, because a `trends` set in the Zabbix frontend and then imported into a
// configuration that does not manage it is the user's, not ours.
//
// The boundary crossing is not symmetric, and both halves were broken:
//
//   - into a non-numeric type, the stored "365d" is *rejected* by item.update from
//     7.0 (`Invalid parameter "/1/trends": value must be 0`) and silently
//     rewritten to "0" on 6.0. buildItemObject already forced "0" here, so the
//     apply worked -- but the plan had said nothing about trends changing,
//     which the SDK's legacy type system downgrades from an error to a log
//     line nobody reads. Now the plan says it.
//   - out of one, nothing forced anything: the item kept trends "0"
//     for ever, because the configuration was silent and Optional+Computed
//     means keep. An item switched from text to unsigned quietly collected no
//     trends at all. Verified against 6.0.48, 7.0.29, 7.4.13 and 8.0-trunk:
//     item.update with a new value_type and no trends leaves the stored "0"
//     exactly as it was, so the server will not do this for us.
func itemTrendsCustomizeDiff(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	// the configuration owns the value; buildItemObject validates it
	if _, given := configuredString(d, "trends"); given {
		return nil
	}
	// an unresolved reference: nothing to derive from yet, and the write path
	// derives again at apply
	if !d.NewValueKnown("valuetype") {
		return nil
	}
	newType, ok := ITEM_VALUE_TYPES[d.Get("valuetype").(string)]
	if !ok {
		return nil // rejected by the validator, with a better message than any we could add
	}

	if d.Id() == "" {
		return d.SetNew("trends", itemDerivedTrends(newType))
	}

	old, _ := d.GetChange("valuetype")
	oldType, ok := ITEM_VALUE_TYPES[old.(string)]
	if !ok || itemTrendless(oldType) == itemTrendless(newType) {
		return nil
	}
	return d.SetNew("trends", itemDerivedTrends(newType))
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
		// d.Get, not d.GetOk: "" is a value here rather than the absence of
		// one, and GetOk reports it as unset, which is how the host inventory
		// fields ended up unclearable. Neither field carries omitempty, so ""
		// reaches the server as "" and the clear lands.
		Units:       d.Get("units").(string),
		Description: d.Get("description").(string),
	}
	preprocessors, err := itemGeneratePreprocessors(d, api)
	if err != nil {
		return nil, err
	}
	item.Preprocessors = preprocessors
	item.Tags = tagGenerate(d)

	// itemTrendsCustomizeDiff has normally derived this already and it arrives
	// here in the plan; this is the fallback for the case it steps aside from,
	// a `valuetype` that was not yet known when the plan was made.
	if v, ok := d.GetOk("trends"); ok {
		item.Trends = v.(string)
	} else {
		item.Trends = itemDerivedTrends(item.ValueType)
		d.Set("trends", item.Trends)
	}

	// Text and log items keep no trends at all, and the value type is what
	// decides that. itemTrendsCustomizeDiff plans "0" whenever the value type
	// moves into that class with the configuration silent; this is the last
	// word on the same rule, and it is what catches the configuration that
	// *does* name a trends the value type cannot have.
	//
	// Probed by calling item.update with value_type 4 and trends "30d":
	//
	//	6.0.48       accepted, and trends silently stored as 0
	//	7.0.29       Invalid parameter "/1/trends": value must be 0.
	//	7.4.13       same
	//	8.0-trunk    same
	//
	// item.create behaves the same way, so a text item written with an
	// explicit non-zero trends was a hard failure from 7.0 and a diff that
	// never converged on 6.0 -- the server stored 0 and the read put 0 back.
	// That one is the user's own value, so it is an error rather than an
	// override: silently rewriting it would leave the same diff behind.
	if itemTrendless(item.ValueType) {
		if v, ok := configuredString(d, "trends"); ok && v != "0" {
			return nil, fmt.Errorf(
				"trends must be \"0\" for a %s item, not %q: Zabbix keeps trends only for numeric values",
				ITEM_VALUE_TYPES_REV[item.ValueType], v)
		}
		item.Trends = "0"
		d.Set("trends", "0")
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
