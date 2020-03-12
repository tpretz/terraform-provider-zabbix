package provider

import (
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
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

// generate the above structures
var _ = func() bool {
	for k, v := range TRIGGER_PRIORITY {
		TRIGGER_PRIORITY_REV[v] = k
		TRIGGER_PRIORITY_ARR = append(TRIGGER_PRIORITY_ARR, k)
	}
	return false
}()

// terraform resource handler for triggers
func resourceTrigger() *schema.Resource {
	return &schema.Resource{
		Create: resourceTriggerCreate,
		Read:   resourceTriggerRead,
		Update: resourceTriggerUpdate,
		Delete: resourceTriggerDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
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
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "link to url relevent to trigger",
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
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
			"correlation_mode": &schema.Schema{ // tie to tag
				Type:     schema.TypeString,
				Optional: true,
				Default:  "0",
			},
			"correlation_tag": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"manual_close": &schema.Schema{ // change to boolean
				Type:     schema.TypeString,
				Optional: true,
				Default:  "0",
			},
			"dependencies": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			// add tags
		},
	}
}

func buildTriggerObject(d *schema.ResourceData) zabbix.Trigger {
	item := zabbix.Trigger{
		Description:        d.Get("name").(string),
		Expression:         d.Get("expression").(string),
		Comments:           d.Get("comments").(string),
		Priority:           TRIGGER_PRIORITY[d.Get("priority").(string)],
		Status:             0,
		Type:               "0",
		Url:                d.Get("url").(string),
		RecoveryMode:       "0",
		RecoveryExpression: "",
		CorrelationMode:    d.Get("correlation_mode").(string),
		CorrelationTag:     d.Get("correlation_tag").(string),
		ManualClose:        d.Get("manual_close").(string),
	}

	if !d.Get("enabled").(bool) {
		item.Status = 1
	}
	if d.Get("multiple").(bool) {
		item.Type = "1"
	}

	if d.Get("recovery_none").(bool) {
		item.RecoveryMode = "2"
	} else if v := d.Get("recovery_expression").(string); v != "" {
		item.RecoveryMode = "1"
		item.RecoveryExpression = v
	}

	item.Dependencies = buildTriggerIds(d.Get("dependencies").(*schema.Set))

	return item
}

func resourceTriggerCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := buildTriggerObject(d)

	items := []zabbix.Trigger{item}

	err := api.TriggersCreate(items)

	if err != nil {
		return err
	}

	log.Trace("crated trigger: %+v", items[0])

	d.SetId(items[0].TriggerID)

	return resourceTriggerRead(d, m)
}

func resourceTriggerRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of trigger with id %s", d.Id())

	triggers, err := api.TriggersGet(zabbix.Params{
		"triggerids":         d.Id(),
		"expandExpression":   "extend",
		"selectDependencies": "extend",
	})

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
	d.Set("expression", t.Expression)
	d.Set("comments", t.Comments)
	d.Set("priority", TRIGGER_PRIORITY_REV[t.Priority])
	d.Set("enabled", t.Status == 0)
	d.Set("multiple", t.Type == "1")
	d.Set("url", t.Url)
	d.Set("recovery_mode", t.RecoveryMode)
	d.Set("recovery_expression", t.RecoveryExpression)
	d.Set("correlation_mode", t.CorrelationMode)
	d.Set("correlation_tag", t.CorrelationTag)
	d.Set("manual_close", t.ManualClose)

	if t.RecoveryMode == "2" {
		d.Set("recovery_none", true)
	} else {
		d.Set("recovery_none", false)
	}

	// should not occur, but need to express somehow, in a way that allows cleanup
	if t.RecoveryMode == "1" && t.RecoveryExpression == "" {
		// this should trigger a mismatch, and by setting to 0 len str it should flip recovery mode
		d.Set("recovery_expression", "<recovery_mode_enabled_no_expression>")
	}

	dependenciesSet := schema.NewSet(schema.HashString, []interface{}{})
	for _, v := range t.Dependencies {
		dependenciesSet.Add(v.TriggerID)
	}
	d.Set("dependencies", dependenciesSet)

	return nil
}

func resourceTriggerUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := buildTriggerObject(d)

	item.TriggerID = d.Id()

	items := []zabbix.Trigger{item}

	err := api.TriggersUpdate(items)

	if err != nil {
		return err
	}

	return resourceTriggerRead(d, m)
}

func resourceTriggerDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.TriggersDeleteByIds([]string{d.Id()})
}
