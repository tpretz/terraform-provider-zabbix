package provider

import (
	"errors"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

var TRIGGER_PRIORITY = map[string]zabbix.SeverityType{
	"not_classified": zabbix.NotClassified,
	"info":           zabbix.Information,
	"warn":           zabbix.Warning,
	"average":        zabbix.Average,
	"high":           zabbix.High,
	"disaster":       zabbix.Critical,
}
var TRIGGER_PRIORITY_REV = map[zabbix.SeverityType]string{}
var TRIGGER_PRIORITY_ARR = []string{}

// TRIGGER_CORRELATION maps the Terraform "correlation_mode" values onto the
// API's numeric event correlation codes.
var TRIGGER_CORRELATION = map[string]int{
	"all": 0,
	"tag": 1,
}
var TRIGGER_CORRELATION_REV = map[int]string{}
var TRIGGER_CORRELATION_ARR = []string{}

// generate the above structures
var _ = func() bool {
	for k, v := range TRIGGER_PRIORITY {
		TRIGGER_PRIORITY_REV[v] = k
		TRIGGER_PRIORITY_ARR = append(TRIGGER_PRIORITY_ARR, k)
	}
	for k, v := range TRIGGER_CORRELATION {
		TRIGGER_CORRELATION_REV[v] = k
		TRIGGER_CORRELATION_ARR = append(TRIGGER_CORRELATION_ARR, k)
	}
	sort.Strings(TRIGGER_CORRELATION_ARR)
	return false
}()

var schemaTrigger = map[string]*schema.Schema{
	// api "description", gui rewrites to name, so shall we
	"name": &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Description:  "Trigger name",
	},
	"expression": &schema.Schema{
		Type:         schema.TypeString,
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Description:  "Trigger Expression",
		Required:     true,
	},
	"comments": &schema.Schema{
		Type:        schema.TypeString,
		Description: "Trigger comments",
		Optional:    true,
	},
	"priority": &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "Trigger Priority level, one of: " + strings.Join(TRIGGER_PRIORITY_ARR, ", "),
		ValidateFunc: validation.StringInSlice(TRIGGER_PRIORITY_ARR, false),
		Default:      "not_classified",
	},
	"enabled": &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     true,
		Description: "Enable this trigger",
	},
	"multiple": &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		Description: "generate multiple events",
	},
	"url": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "link to url relevent to trigger",
		// the empty string has to be allowed through, or the attribute is
		// write-once: IsURLWithHTTPorHTTPS on its own rejects "", so a
		// trigger that had been given a link could never have it taken away
		ValidateFunc: validation.Any(
			validation.StringIsEmpty,
			validation.IsURLWithHTTPorHTTPS,
		),
	},
	"recovery_none": &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		Description: "set recovery mode to none",
	},
	"recovery_expression": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "use recovery expression (recovery_none must not be true)",
	},
	"event_name": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Problem event name, may contain macros from the trigger expression. Defaults to the trigger name when empty",
	},
	"opdata": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Operational data shown alongside the problem, may contain item value macros",
	},
	"correlation_mode": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		// Optional+Computed rather than defaulted: before this attribute
		// existed the mode was inferred from correlation_tag, and a default
		// would put every such configuration into permanent drift.
		Computed: true,
		Description: "Event correlation mode, one of: " + strings.Join(TRIGGER_CORRELATION_ARR, ", ") +
			". \"tag\" closes a problem only when the matching event carries the same correlation_tag value, and requires correlation_tag to be set. Inferred from correlation_tag when omitted",
		ValidateFunc: validation.StringInSlice(TRIGGER_CORRELATION_ARR, false),
	},
	"correlation_tag": &schema.Schema{
		Type:        schema.TypeString,
		Description: "Tag name used to match problem and recovery events when correlation_mode is \"tag\"",
		Optional:    true,
	},
	"manual_close": &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		Description: "Manual resolution",
	},
	"dependencies": &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Schema{
			Type:         schema.TypeString,
			ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9]+$"), "must be a numeric string"),
		},
		Description: "Trigger Dependencies",
	},
	"tag": tagSetSchema,
}

// terraform resource handler for triggers
func resourceTrigger() *schema.Resource {
	return &schema.Resource{
		Create: resourceTriggerCreate(false),
		Read:   resourceTriggerRead(false),
		Update: resourceTriggerUpdate(false),
		Delete: resourceTriggerDelete(false),
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: schemaTrigger,
	}
}
func resourceProtoTrigger() *schema.Resource {
	return &schema.Resource{
		Create: resourceTriggerCreate(true),
		Read:   resourceTriggerRead(true),
		Update: resourceTriggerUpdate(true),
		Delete: resourceTriggerDelete(true),
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: schemaTrigger,
	}
}

// Build Trigger struct for create/modify
func buildTriggerObject(d *schema.ResourceData) (zabbix.Trigger, error) {
	item := zabbix.Trigger{
		Description:        d.Get("name").(string),
		EventName:          d.Get("event_name").(string),
		Opdata:             d.Get("opdata").(string),
		Expression:         d.Get("expression").(string),
		Comments:           d.Get("comments").(string),
		Priority:           TRIGGER_PRIORITY[d.Get("priority").(string)],
		Status:             0,
		Type:               0,
		Url:                d.Get("url").(string),
		RecoveryMode:       0,
		RecoveryExpression: "",
		CorrelationMode:    0,
		CorrelationTag:     "",
		ManualClose:        0,
	}

	if !d.Get("enabled").(bool) {
		item.Status = 1
	}
	if d.Get("multiple").(bool) {
		item.Type = 1
	}

	if d.Get("recovery_none").(bool) {
		item.RecoveryMode = 2
	} else if v := d.Get("recovery_expression").(string); v != "" {
		item.RecoveryMode = 1
		item.RecoveryExpression = v
	}

	// correlation_mode is Optional+Computed, so an empty value means "not in
	// the configuration". Historically the only way to ask for tag correlation
	// was to set correlation_tag, and that keeps working.
	tag := d.Get("correlation_tag").(string)
	switch mode := d.Get("correlation_mode").(string); {
	case mode != "":
		item.CorrelationMode = TRIGGER_CORRELATION[mode]
	case tag != "":
		item.CorrelationMode = 1
	}
	if item.CorrelationMode == 1 {
		if tag == "" {
			return item, errors.New("correlation_mode \"tag\" requires correlation_tag to be set")
		}
		item.CorrelationTag = tag
	} else if tag != "" {
		return item, errors.New("correlation_tag requires correlation_mode \"tag\"")
	}

	if d.Get("manual_close").(bool) {
		item.ManualClose = 1
	}

	item.Dependencies = buildTriggerIds(d.Get("dependencies").(*schema.Set))
	item.Tags = tagGenerate(d)

	return item, nil
}

// create trigger terraform handler
func resourceTriggerCreate(prototype bool) schema.CreateFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		api := m.(*zabbix.API)

		item, err := buildTriggerObject(d)
		if err != nil {
			return err
		}

		items := []zabbix.Trigger{item}

		if prototype {
			err = api.ProtoTriggersCreate(items)
		} else {
			err = api.TriggersCreate(items)
		}

		if err != nil {
			return err
		}

		log.Trace("crated trigger: %+v", items[0])

		d.SetId(items[0].TriggerID)

		return resourceTriggerRead(prototype)(d, m)
	}
}

// read tirgger terraform handler
func resourceTriggerRead(prototype bool) schema.ReadFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		api := m.(*zabbix.API)

		log.Debug("Lookup of trigger with id %s", d.Id())

		params := zabbix.Params{
			"triggerids":         d.Id(),
			"expandExpression":   "extend",
			"selectDependencies": "extend",
			"selectTags":         "extend",
		}

		var triggers zabbix.Triggers
		var err error

		if prototype {
			triggers, err = api.ProtoTriggersGet(params)
		} else {
			triggers, err = api.TriggersGet(params)
		}

		if err != nil {
			return err
		}

		if len(triggers) < 1 {
			d.SetId("")
			return nil
		}
		if len(triggers) > 1 {
			return errors.New("multiple triggers found")
		}
		t := triggers[0]

		log.Debug("Got trigger: %+v", t)

		d.Set("name", t.Description)
		d.Set("event_name", t.EventName)
		d.Set("opdata", t.Opdata)
		d.Set("expression", t.Expression)
		d.Set("comments", t.Comments)
		d.Set("priority", TRIGGER_PRIORITY_REV[t.Priority])
		d.Set("enabled", t.Status == 0)
		d.Set("multiple", t.Type == 1)
		d.Set("url", t.Url)
		d.Set("recovery_expression", t.RecoveryExpression)
		d.Set("correlation_mode", TRIGGER_CORRELATION_REV[t.CorrelationMode])
		d.Set("correlation_tag", t.CorrelationTag)
		d.Set("manual_close", t.ManualClose == 1)
		d.Set("tag", flattenTags(t.Tags))

		if t.RecoveryMode == 2 {
			d.Set("recovery_none", true)
		} else {
			d.Set("recovery_none", false)
		}

		// should not occur, but need to express somehow, in a way that allows cleanup
		if t.RecoveryMode == 1 && t.RecoveryExpression == "" {
			// this should trigger a mismatch, and by setting to 0 len str it should flip recovery mode
			d.Set("recovery_expression", "<recovery_mode_enabled_no_expression>")
		}
		// correlation_mode is now round-tripped in its own right, so the
		// mode-1-with-no-tag case needs no sentinel value to force a diff:
		// buildTriggerObject rejects that combination outright.

		dependenciesSet := schema.NewSet(schema.HashString, []interface{}{})
		for _, v := range t.Dependencies {
			dependenciesSet.Add(v.TriggerID)
		}
		d.Set("dependencies", dependenciesSet)

		return nil
	}
}

// update trigger terraform handler
func resourceTriggerUpdate(prototype bool) schema.UpdateFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		api := m.(*zabbix.API)

		item, err := buildTriggerObject(d)
		if err != nil {
			return err
		}

		item.TriggerID = d.Id()

		items := []zabbix.Trigger{item}

		if prototype {
			err = api.ProtoTriggersUpdate(items)
		} else {
			err = api.TriggersUpdate(items)
		}

		if err != nil {
			return err
		}

		return resourceTriggerRead(prototype)(d, m)
	}
}

// delete trigger terraform handler
func resourceTriggerDelete(prototype bool) schema.DeleteFunc {
	return func(d *schema.ResourceData, m interface{}) error {
		api := m.(*zabbix.API)
		if prototype {
			return api.ProtoTriggersDeleteByIds([]string{d.Id()})
		}
		return api.TriggersDeleteByIds([]string{d.Id()})
	}
}
