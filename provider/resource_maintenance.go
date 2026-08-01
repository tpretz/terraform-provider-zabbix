package provider

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

var MAINTENANCE_TYPE_LOOKUP = map[string]zabbix.MaintenanceType{
	"with_data": zabbix.MaintenanceWithData,
	"no_data":   zabbix.MaintenanceNoData,
}
var MAINTENANCE_TYPE_LOOKUP_REV = map[zabbix.MaintenanceType]string{}
var MAINTENANCE_TYPE_ARR = []string{}

var MAINTENANCE_PERIOD_LOOKUP = map[string]zabbix.TimePeriodType{
	"one_time": zabbix.TimePeriodOneTime,
	"daily":    zabbix.TimePeriodDaily,
	"weekly":   zabbix.TimePeriodWeekly,
	"monthly":  zabbix.TimePeriodMonthly,
}
var MAINTENANCE_PERIOD_LOOKUP_REV = map[zabbix.TimePeriodType]string{}
var MAINTENANCE_PERIOD_ARR = []string{}

// maintenance tag operators, a subset of the generic tag operators
var MAINTENANCE_TAG_OPERATOR_ARR = []string{"0", "2"}

var _ = func() bool {
	for k, v := range MAINTENANCE_TYPE_LOOKUP {
		MAINTENANCE_TYPE_LOOKUP_REV[v] = k
		MAINTENANCE_TYPE_ARR = append(MAINTENANCE_TYPE_ARR, k)
	}
	for k, v := range MAINTENANCE_PERIOD_LOOKUP {
		MAINTENANCE_PERIOD_LOOKUP_REV[v] = k
		MAINTENANCE_PERIOD_ARR = append(MAINTENANCE_PERIOD_ARR, k)
	}
	return false
}()

var maintenanceNumeric = regexp.MustCompile("^[0-9]+$")

// resourceMaintenance terraform resource handler for maintenance windows
func resourceMaintenance() *schema.Resource {
	return &schema.Resource{
		Create: resourceMaintenanceCreate,
		Read:   resourceMaintenanceRead,
		Update: resourceMaintenanceUpdate,
		Delete: resourceMaintenanceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Maintenance window name",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"active_since": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Unix timestamp at which the window becomes active",
				ValidateFunc: validation.StringMatch(maintenanceNumeric, "must be a unix timestamp"),
			},
			"active_till": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Unix timestamp at which the window stops being active",
				ValidateFunc: validation.StringMatch(maintenanceNumeric, "must be a unix timestamp"),
			},
			"maintenance_type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "with_data",
				Description:  "Whether data is collected during maintenance, one of: " + strings.Join(MAINTENANCE_TYPE_ARR, ", "),
				ValidateFunc: validation.StringInSlice(MAINTENANCE_TYPE_ARR, false),
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Maintenance window description",
			},
			"groups": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Host group IDs placed in maintenance",
			},
			"hosts": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Host IDs placed in maintenance",
			},
			"tag": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Problem tags to suppress, only valid when maintenance_type is with_data",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"operator": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "2",
							Description:  "0 = equals, 2 = contains",
							ValidateFunc: validation.StringInSlice(MAINTENANCE_TAG_OPERATOR_ARR, false),
						},
					},
				},
			},
			"timeperiod": &schema.Schema{
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "Periods during which the maintenance is in effect",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "one_time",
							Description:  "Recurrence, one of: " + strings.Join(MAINTENANCE_PERIOD_ARR, ", "),
							ValidateFunc: validation.StringInSlice(MAINTENANCE_PERIOD_ARR, false),
						},
						"period": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "Duration of the period in seconds",
						},
						"start_date": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Unix timestamp the one_time period starts at",
						},
						"start_time": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Seconds past midnight the period starts at, for recurring periods",
						},
						"every": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Recurrence interval, meaning depends on type",
						},
						"dayofweek": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Bit flags of weekdays, for weekly periods",
						},
						"day": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Day of the month, for monthly periods",
						},
						"month": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Bit flags of months, for monthly periods",
						},
					},
				},
			},
		},
	}
}

func maintenanceBuildObject(d *schema.ResourceData) (*zabbix.Maintenance, error) {
	m := zabbix.Maintenance{
		Name:            d.Get("name").(string),
		Description:     d.Get("description").(string),
		ActiveSince:     d.Get("active_since").(string),
		ActiveTill:      d.Get("active_till").(string),
		MaintenanceType: MAINTENANCE_TYPE_LOOKUP[d.Get("maintenance_type").(string)],
	}

	groups := d.Get("groups").(*schema.Set).List()
	hosts := d.Get("hosts").(*schema.Set).List()
	if len(groups) == 0 && len(hosts) == 0 {
		return nil, errors.New("a maintenance window requires at least one host or group")
	}

	m.GroupIDs = make([]zabbix.HostGroupID, len(groups))
	for i, g := range groups {
		m.GroupIDs[i] = zabbix.HostGroupID{GroupID: g.(string)}
	}
	m.HostIDs = make([]string, len(hosts))
	for i, h := range hosts {
		m.HostIDs[i] = h.(string)
	}

	// Tags only apply when data is still being collected.
	if m.MaintenanceType == zabbix.MaintenanceWithData {
		tags := d.Get("tag").(*schema.Set).List()
		m.Tags = make(zabbix.MaintenanceTags, len(tags))
		for i, t := range tags {
			tm := t.(map[string]interface{})
			m.Tags[i] = zabbix.MaintenanceTag{
				Tag:      tm["key"].(string),
				Value:    tm["value"].(string),
				Operator: tm["operator"].(string),
			}
		}
	}

	periods := d.Get("timeperiod").([]interface{})
	m.TimePeriods = make(zabbix.MaintenanceTimePeriods, len(periods))
	for i, p := range periods {
		pm := p.(map[string]interface{})
		ptype := pm["type"].(string)
		tp := zabbix.MaintenanceTimePeriod{
			TimePeriodType: MAINTENANCE_PERIOD_LOOKUP[ptype],
			Period:         pm["period"].(string),
		}

		// Each recurrence uses a different subset of fields and Zabbix rejects
		// the ones that do not apply, even when they hold their default value.
		var allowed []string
		switch ptype {
		case "one_time":
			allowed = []string{"start_date"}
		case "daily":
			allowed = []string{"every", "start_time"}
		case "weekly":
			allowed = []string{"every", "dayofweek", "start_time"}
		case "monthly":
			allowed = []string{"month", "day", "every", "dayofweek", "start_time"}
		}

		dst := map[string]*string{
			"start_date": &tp.StartDate,
			"start_time": &tp.StartTime,
			"every":      &tp.Every,
			"dayofweek":  &tp.DayOfWeek,
			"day":        &tp.Day,
			"month":      &tp.Month,
		}
		for _, key := range allowed {
			if v, ok := pm[key].(string); ok && v != "" {
				*dst[key] = v
			}
		}

		// A monthly period is driven either by day of month or by day of week,
		// never both; Zabbix rejects day when dayofweek is in use.
		if ptype == "monthly" && tp.DayOfWeek != "" && tp.DayOfWeek != "0" {
			tp.Day = ""
		}

		m.TimePeriods[i] = tp
	}

	return &m, nil
}

func resourceMaintenanceCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item, err := maintenanceBuildObject(d)
	if err != nil {
		return err
	}

	items := zabbix.Maintenances{*item}
	if err := api.MaintenancesCreate(items); err != nil {
		return err
	}

	log.Trace("created maintenance: %+v", items[0])
	d.SetId(items[0].MaintenanceID)

	return resourceMaintenanceRead(d, m)
}

func resourceMaintenanceRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of maintenance with id %s", d.Id())

	items, err := api.MaintenancesGet(zabbix.Params{
		"maintenanceids":    d.Id(),
		"selectTimeperiods": "extend",
		"selectHostGroups":  "extend",
		"selectHosts":       []string{"hostid"},
		"selectTags":        "extend",
	})
	if err != nil {
		return err
	}
	if len(items) < 1 {
		d.SetId("")
		return nil
	}
	if len(items) > 1 {
		return errors.New("multiple maintenance windows found")
	}
	t := items[0]

	d.SetId(t.MaintenanceID)
	d.Set("name", t.Name)
	d.Set("description", t.Description)
	d.Set("active_since", t.ActiveSince)
	d.Set("active_till", t.ActiveTill)
	d.Set("maintenance_type", MAINTENANCE_TYPE_LOOKUP_REV[t.MaintenanceType])

	groups := make([]string, len(t.HostGroups))
	for i, g := range t.HostGroups {
		groups[i] = g.GroupID
	}
	d.Set("groups", groups)

	tags := make([]interface{}, len(t.Tags))
	for i, tg := range t.Tags {
		op := tg.Operator
		if op == "" {
			op = "2"
		}
		tags[i] = map[string]interface{}{
			"key":      tg.Tag,
			"value":    tg.Value,
			"operator": op,
		}
	}
	d.Set("tag", tags)

	periods := make([]interface{}, len(t.TimePeriods))
	for i, p := range t.TimePeriods {
		periods[i] = map[string]interface{}{
			"id":         p.TimePeriodID,
			"type":       MAINTENANCE_PERIOD_LOOKUP_REV[p.TimePeriodType],
			"period":     p.Period,
			"start_date": p.StartDate,
			"start_time": p.StartTime,
			"every":      p.Every,
			"dayofweek":  p.DayOfWeek,
			"day":        p.Day,
			"month":      p.Month,
		}
	}
	d.Set("timeperiod", periods)

	return nil
}

func resourceMaintenanceUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item, err := maintenanceBuildObject(d)
	if err != nil {
		return err
	}
	item.MaintenanceID = d.Id()

	if err := api.MaintenancesUpdate(zabbix.Maintenances{*item}); err != nil {
		return fmt.Errorf("zabbix_maintenance update: %w", err)
	}

	return resourceMaintenanceRead(d, m)
}

func resourceMaintenanceDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.MaintenancesDeleteByIds([]string{d.Id()})
}
