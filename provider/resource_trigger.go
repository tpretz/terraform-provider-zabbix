package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

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
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Host ID",
			},
			"expression": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"comments": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"opdata": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			// priority
			"status": &schema.Schema{ // change to "enabled"
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"type": &schema.Schema{ // change to "multiple"
				Type:     schema.TypeString,
				Optional: true,
				Default:  "0",
			},
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"recovery_mode": &schema.Schema{ // change to enum or tie to expression
				Type:     schema.TypeString,
				Optional: true,
				Default:  "0",
			},
			"recovery_expression": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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
		Description:        d.Get("description").(string),
		Expression:         d.Get("expression").(string),
		Comments:           d.Get("comments").(string),
		Opdata:             d.Get("opdata").(string),
		Status:             zabbix.StatusType(d.Get("status").(int)),
		Type:               d.Get("type").(string),
		Url:                d.Get("url").(string),
		RecoveryMode:       d.Get("recovery_mode").(string),
		RecoveryExpression: d.Get("recovery_expression").(string),
		CorrelationMode:    d.Get("correlation_mode").(string),
		CorrelationTag:     d.Get("correlation_tag").(string),
		ManualClose:        d.Get("manual_close").(string),
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

	d.Set("description", t.Description)
	d.Set("expression", t.Expression)
	d.Set("comments", t.Comments)
	d.Set("opdata", t.Opdata)
	d.Set("status", t.Status)
	d.Set("type", t.Type)
	d.Set("url", t.Url)
	d.Set("recovery_mode", t.RecoveryMode)
	d.Set("correlation_mode", t.CorrelationMode)
	d.Set("correlation_tag", t.CorrelationTag)
	d.Set("manual_close", t.ManualClose)

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
