package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

func resourceTrigger() *schema.Resource {
	return &schema.Resource{
		Create: resourceTriggerCreate,
		Read:   resourceTriggerRead,
		Update: resourceTriggerUpdate,
		Delete: resourceTriggerDelete,

		Schema: map[string]*schema.Schema{
			"triggerid": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "Zabbix ID",
			},
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
			"status": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"type": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"recovery_mode": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"recovery_expression": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"correlation_mode": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"correlation_tag": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"manual_close": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
		},
	}
}

func resourceTriggerCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := zabbix.Trigger{
		Description:        d.Get("description").(string),
		Expression:         d.Get("expression").(string),
		Comments:           d.Get("comments").(string),
		Opdata:             d.Get("opdata").(string),
		Status:             zabbix.StatusType(d.Get("status").(int)),
		Type:               d.Get("type").(int),
		Url:                d.Get("url").(string),
		RecoveryMode:       d.Get("recovery_mode").(int),
		RecoveryExpression: d.Get("recovery_expression").(string),
		CorrelationMode:    d.Get("correlation_mode").(int),
		CorrelationTag:     d.Get("correlation_tag").(string),
		ManualClose:        d.Get("manual_close").(int),
	}

	items := []zabbix.Trigger{item}

	err := api.TriggersCreate(items)

	if err != nil {
		return err
	}

	log.Trace("crated trigger: %+v", items[0])

	d.Set("triggerid", items[0].TriggerID)
	d.SetId(items[0].TriggerID)

	return resourceTriggerRead(d, m)
}

func resourceTriggerRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	id := d.Get("triggerid").(string)

	log.Debug("Lookup of trigger with id %s", id)

	t, err := api.TriggerGetByID(id)

	if err != nil {
		return err
	}

	log.Debug("Got trigger: %+v", t)

	d.Set("triggerid", t.TriggerID)
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

	return nil
}

func resourceTriggerUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := zabbix.Trigger{
		TriggerID:          d.Id(),
		Description:        d.Get("description").(string),
		Expression:         d.Get("expression").(string),
		Comments:           d.Get("comments").(string),
		Opdata:             d.Get("opdata").(string),
		Status:             zabbix.StatusType(d.Get("status").(int)),
		Type:               d.Get("type").(int),
		Url:                d.Get("url").(string),
		RecoveryMode:       d.Get("recovery_mode").(int),
		RecoveryExpression: d.Get("recovery_expression").(string),
		CorrelationMode:    d.Get("correlation_mode").(int),
		CorrelationTag:     d.Get("correlation_tag").(string),
		ManualClose:        d.Get("manual_close").(int),
	}

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
