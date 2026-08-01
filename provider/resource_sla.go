package provider

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

// SLA_PERIOD_LOOKUP maps readable names to the numeric reporting period the API
// expects.
var SLA_PERIOD_LOOKUP = map[string]string{
	"daily":     "0",
	"weekly":    "1",
	"monthly":   "2",
	"quarterly": "3",
	"annually":  "4",
}
var SLA_PERIOD_LOOKUP_REV = map[string]string{}
var SLA_PERIOD_ARR = []string{}

var _ = func() bool {
	for k, v := range SLA_PERIOD_LOOKUP {
		SLA_PERIOD_LOOKUP_REV[v] = k
		SLA_PERIOD_ARR = append(SLA_PERIOD_ARR, k)
	}
	return false
}()

var SLA_STATUS_LOOKUP = map[string]string{
	"disabled": "0",
	"enabled":  "1",
}
var SLA_STATUS_LOOKUP_REV = map[string]string{}
var SLA_STATUS_ARR = []string{}

var _ = func() bool {
	for k, v := range SLA_STATUS_LOOKUP {
		SLA_STATUS_LOOKUP_REV[v] = k
		SLA_STATUS_ARR = append(SLA_STATUS_ARR, k)
	}
	return false
}()

func resourceSLA() *schema.Resource {
	return &schema.Resource{
		Create: resourceSLACreate,
		Read:   resourceSLARead,
		Update: resourceSLAUpdate,
		Delete: resourceSLADelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "SLA name",
			},
			"period": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Reporting period, one of: " + strings.Join(SLA_PERIOD_ARR, ", "),
				ValidateFunc: validation.StringInSlice(SLA_PERIOD_ARR, false),
			},
			"slo": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "SLO percentage (e.g. 99.9)",
			},
			"effective_date": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Effective date as Unix timestamp",
			},
			"timezone": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Timezone string, e.g. UTC or Europe/London",
			},
			"status": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "enabled",
				Description:  "SLA status, one of: " + strings.Join(SLA_STATUS_ARR, ", "),
				ValidateFunc: validation.StringInSlice(SLA_STATUS_ARR, false),
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "SLA description",
			},
			"service_tags": &schema.Schema{
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "Service tags for matching services to this SLA (at least one required)",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"tag": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"operator": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "0",
							Description: "0 - equals, 2 - contains",
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
					},
				},
			},
			"schedule": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Custom schedule entries (period_from/period_to per weekday as seconds from start of week)",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"period_from": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"period_to": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"excluded_downtimes": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Excluded downtimes",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"period_from": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"period_to": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func slaBuildObject(d *schema.ResourceData) zabbix.SLA {
	sla := zabbix.SLA{
		Name:          d.Get("name").(string),
		Period:        SLA_PERIOD_LOOKUP[d.Get("period").(string)],
		SLO:           d.Get("slo").(string),
		EffectiveDate: d.Get("effective_date").(string),
		Timezone:      d.Get("timezone").(string),
		Status:        SLA_STATUS_LOOKUP[d.Get("status").(string)],
		Description:   d.Get("description").(string),
	}

	// service_tags (required, at least 1)
	if v, ok := d.GetOk("service_tags"); ok {
		tags := v.([]interface{})
		sla.ServiceTags = make(zabbix.SLAServiceTags, len(tags))
		for i, raw := range tags {
			t := raw.(map[string]interface{})
			sla.ServiceTags[i] = zabbix.SLAServiceTag{
				Tag:      t["tag"].(string),
				Operator: t["operator"].(string),
				Value:    t["value"].(string),
			}
		}
	}

	// schedule
	if v, ok := d.GetOk("schedule"); ok {
		entries := v.([]interface{})
		sla.Schedule = make(zabbix.SLASchedules, len(entries))
		for i, raw := range entries {
			e := raw.(map[string]interface{})
			sla.Schedule[i] = zabbix.SLASchedule{
				PeriodFrom: e["period_from"].(string),
				PeriodTo:   e["period_to"].(string),
			}
		}
	}

	// excluded_downtimes
	if v, ok := d.GetOk("excluded_downtimes"); ok {
		downtimes := v.([]interface{})
		sla.ExcludedDowntimes = make(zabbix.SLAExcludedDowntimes, len(downtimes))
		for i, raw := range downtimes {
			dt := raw.(map[string]interface{})
			sla.ExcludedDowntimes[i] = zabbix.SLAExcludedDowntime{
				Name:       dt["name"].(string),
				PeriodFrom: dt["period_from"].(string),
				PeriodTo:   dt["period_to"].(string),
			}
		}
	}

	return sla
}

func resourceSLACreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	sla := slaBuildObject(d)
	slas := zabbix.SLAs{sla}

	err := api.SLAsCreate(slas)
	if err != nil {
		return err
	}

	d.SetId(slas[0].SLAID)
	log.Trace("created SLA: %s", slas[0].SLAID)

	return resourceSLARead(d, m)
}

func resourceSLARead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of SLA with id %s", d.Id())

	sla, err := api.SLAGetByID(d.Id())
	if err != nil {
		return err
	}
	if sla == nil {
		d.SetId("")
		return nil
	}

	d.SetId(sla.SLAID)
	d.Set("name", sla.Name)
	d.Set("period", SLA_PERIOD_LOOKUP_REV[sla.Period])
	d.Set("slo", sla.SLO)
	d.Set("effective_date", sla.EffectiveDate)
	d.Set("timezone", sla.Timezone)
	d.Set("status", SLA_STATUS_LOOKUP_REV[sla.Status])
	d.Set("description", sla.Description)

	// service_tags
	serviceTags := make([]map[string]interface{}, len(sla.ServiceTags))
	for i, t := range sla.ServiceTags {
		serviceTags[i] = map[string]interface{}{
			"tag":      t.Tag,
			"operator": t.Operator,
			"value":    t.Value,
		}
	}
	d.Set("service_tags", serviceTags)

	// schedule
	schedule := make([]map[string]interface{}, len(sla.Schedule))
	for i, s := range sla.Schedule {
		schedule[i] = map[string]interface{}{
			"period_from": s.PeriodFrom,
			"period_to":   s.PeriodTo,
		}
	}
	d.Set("schedule", schedule)

	// excluded_downtimes
	downtimes := make([]map[string]interface{}, len(sla.ExcludedDowntimes))
	for i, dt := range sla.ExcludedDowntimes {
		downtimes[i] = map[string]interface{}{
			"name":        dt.Name,
			"period_from": dt.PeriodFrom,
			"period_to":   dt.PeriodTo,
		}
	}
	d.Set("excluded_downtimes", downtimes)

	return nil
}

func resourceSLAUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	sla := slaBuildObject(d)
	sla.SLAID = d.Id()

	// Explicitly clear sub-objects if empty so Zabbix removes them
	if _, ok := d.GetOk("schedule"); !ok {
		sla.Schedule = zabbix.SLASchedules{}
	}
	if _, ok := d.GetOk("excluded_downtimes"); !ok {
		sla.ExcludedDowntimes = zabbix.SLAExcludedDowntimes{}
	}

	if err := api.SLAsUpdate(zabbix.SLAs{sla}); err != nil {
		return err
	}

	return resourceSLARead(d, m)
}

func resourceSLADelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.SLAsDeleteByIds([]string{d.Id()})
}
