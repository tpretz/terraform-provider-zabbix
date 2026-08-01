package provider

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

// SERVICE_ALGORITHM_LOOKUP maps readable names to the numeric values the API
// expects for the status calculation rule.
var SERVICE_ALGORITHM_LOOKUP = map[string]string{
	"set_ok":              "0",
	"most_critical_all":   "1",
	"most_critical_child": "2",
}
var SERVICE_ALGORITHM_LOOKUP_REV = map[string]string{}
var SERVICE_ALGORITHM_ARR = []string{}

var _ = func() bool {
	for k, v := range SERVICE_ALGORITHM_LOOKUP {
		SERVICE_ALGORITHM_LOOKUP_REV[v] = k
		SERVICE_ALGORITHM_ARR = append(SERVICE_ALGORITHM_ARR, k)
	}
	return false
}()

var SERVICE_PROPAGATION_LOOKUP = map[string]string{
	"as_is":       "0",
	"increase_by": "1",
	"decrease_by": "2",
	"ignore":      "3",
	"fixed":       "4",
}
var SERVICE_PROPAGATION_LOOKUP_REV = map[string]string{}
var SERVICE_PROPAGATION_ARR = []string{}

var _ = func() bool {
	for k, v := range SERVICE_PROPAGATION_LOOKUP {
		SERVICE_PROPAGATION_LOOKUP_REV[v] = k
		SERVICE_PROPAGATION_ARR = append(SERVICE_PROPAGATION_ARR, k)
	}
	return false
}()

func resourceService() *schema.Resource {
	return &schema.Resource{
		Create: resourceServiceCreate,
		Read:   resourceServiceRead,
		Update: resourceServiceUpdate,
		Delete: resourceServiceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Service name",
			},
			"algorithm": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Status calculation algorithm, one of: " + strings.Join(SERVICE_ALGORITHM_ARR, ", "),
				ValidateFunc: validation.StringInSlice(SERVICE_ALGORITHM_ARR, false),
			},
			"sortorder": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Position of the service in the tree. 0 for unordered.",
			},
			"weight": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Weight of the service",
			},
			"propagation_rule": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "as_is",
				Description:  "Status propagation rule, one of: " + strings.Join(SERVICE_PROPAGATION_ARR, ", "),
				ValidateFunc: validation.StringInSlice(SERVICE_PROPAGATION_ARR, false),
			},
			"propagation_value": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Status propagation value",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Service description",
			},
			"problem_tags": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Problem tags for matching events to this service",
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
			"tags": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Service tags",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"tag": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
					},
				},
			},
			"children": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "IDs of child services",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"parents": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "IDs of parent services",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"status_rules": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Status rules",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"limit_value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"limit_status": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"new_status": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func serviceBuildObject(d *schema.ResourceData) zabbix.Service {
	svc := zabbix.Service{
		Name:        d.Get("name").(string),
		Algorithm:   SERVICE_ALGORITHM_LOOKUP[d.Get("algorithm").(string)],
		SortOrder:   d.Get("sortorder").(string),
		Weight:      d.Get("weight").(string),
		Description: d.Get("description").(string),
	}

	// Zabbix requires propagation_value whenever propagation_rule is given, and
	// "as is" is its own default, so only send the pair when it changes anything.
	if rule := d.Get("propagation_rule").(string); rule != "" && rule != "as_is" {
		svc.PropagationRule = SERVICE_PROPAGATION_LOOKUP[rule]
		svc.PropagationValue = d.Get("propagation_value").(string)
	}

	// problem_tags
	if v, ok := d.GetOk("problem_tags"); ok {
		tags := v.([]interface{})
		svc.ProblemTags = make(zabbix.ServiceProblemTags, len(tags))
		for i, raw := range tags {
			t := raw.(map[string]interface{})
			svc.ProblemTags[i] = zabbix.ServiceProblemTag{
				Tag:      t["tag"].(string),
				Operator: t["operator"].(string),
				Value:    t["value"].(string),
			}
		}
	}

	// tags
	if v, ok := d.GetOk("tags"); ok {
		tags := v.([]interface{})
		svc.Tags = make(zabbix.ServiceTags, len(tags))
		for i, raw := range tags {
			t := raw.(map[string]interface{})
			svc.Tags[i] = zabbix.ServiceTag{
				Tag:   t["tag"].(string),
				Value: t["value"].(string),
			}
		}
	}

	// children
	if v, ok := d.GetOk("children"); ok {
		ids := v.(*schema.Set).List()
		svc.Children = make(zabbix.ServiceChildren, len(ids))
		for i, id := range ids {
			svc.Children[i] = zabbix.ServiceChild{ServiceID: id.(string)}
		}
	}

	// parents
	if v, ok := d.GetOk("parents"); ok {
		ids := v.(*schema.Set).List()
		svc.Parents = make(zabbix.ServiceParents, len(ids))
		for i, id := range ids {
			svc.Parents[i] = zabbix.ServiceParent{ServiceID: id.(string)}
		}
	}

	// status_rules
	if v, ok := d.GetOk("status_rules"); ok {
		rules := v.([]interface{})
		svc.StatusRules = make(zabbix.ServiceStatusRules, len(rules))
		for i, raw := range rules {
			r := raw.(map[string]interface{})
			svc.StatusRules[i] = zabbix.ServiceStatusRule{
				Type:        r["type"].(string),
				LimitValue:  r["limit_value"].(string),
				LimitStatus: r["limit_status"].(string),
				NewStatus:   r["new_status"].(string),
			}
		}
	}

	return svc
}

func resourceServiceCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	svc := serviceBuildObject(d)
	services := zabbix.Services{svc}

	err := api.ServicesCreate(services)
	if err != nil {
		return err
	}

	d.SetId(services[0].ServiceID)
	log.Trace("created service: %s", services[0].ServiceID)

	return resourceServiceRead(d, m)
}

func resourceServiceRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of service with id %s", d.Id())

	svc, err := api.ServiceGetByID(d.Id())
	if err != nil {
		return err
	}
	if svc == nil {
		d.SetId("")
		return nil
	}

	d.SetId(svc.ServiceID)
	d.Set("name", svc.Name)
	d.Set("algorithm", SERVICE_ALGORITHM_LOOKUP_REV[svc.Algorithm])
	d.Set("sortorder", svc.SortOrder)
	d.Set("weight", svc.Weight)
	d.Set("propagation_rule", SERVICE_PROPAGATION_LOOKUP_REV[svc.PropagationRule])
	d.Set("propagation_value", svc.PropagationValue)
	d.Set("description", svc.Description)

	// problem_tags
	problemTags := make([]map[string]interface{}, len(svc.ProblemTags))
	for i, t := range svc.ProblemTags {
		problemTags[i] = map[string]interface{}{
			"tag":      t.Tag,
			"operator": t.Operator,
			"value":    t.Value,
		}
	}
	d.Set("problem_tags", problemTags)

	// tags
	tags := make([]map[string]interface{}, len(svc.Tags))
	for i, t := range svc.Tags {
		tags[i] = map[string]interface{}{
			"tag":   t.Tag,
			"value": t.Value,
		}
	}
	d.Set("tags", tags)

	// children - just the IDs
	childIDs := make([]string, len(svc.Children))
	for i, c := range svc.Children {
		childIDs[i] = c.ServiceID
	}
	d.Set("children", childIDs)

	// parents - just the IDs
	parentIDs := make([]string, len(svc.Parents))
	for i, p := range svc.Parents {
		parentIDs[i] = p.ServiceID
	}
	d.Set("parents", parentIDs)

	// status_rules
	statusRules := make([]map[string]interface{}, len(svc.StatusRules))
	for i, r := range svc.StatusRules {
		statusRules[i] = map[string]interface{}{
			"type":         r.Type,
			"limit_value":  r.LimitValue,
			"limit_status": r.LimitStatus,
			"new_status":   r.NewStatus,
		}
	}
	d.Set("status_rules", statusRules)

	return nil
}

func resourceServiceUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	svc := serviceBuildObject(d)
	svc.ServiceID = d.Id()

	// If problem_tags is not set (empty), explicitly send empty array to clear them
	if _, ok := d.GetOk("problem_tags"); !ok {
		svc.ProblemTags = zabbix.ServiceProblemTags{}
	}
	if _, ok := d.GetOk("tags"); !ok {
		svc.Tags = zabbix.ServiceTags{}
	}
	if _, ok := d.GetOk("children"); !ok {
		svc.Children = zabbix.ServiceChildren{}
	}
	if _, ok := d.GetOk("parents"); !ok {
		svc.Parents = zabbix.ServiceParents{}
	}
	if _, ok := d.GetOk("status_rules"); !ok {
		svc.StatusRules = zabbix.ServiceStatusRules{}
	}

	if err := api.ServicesUpdate(zabbix.Services{svc}); err != nil {
		return err
	}

	return resourceServiceRead(d, m)
}

func resourceServiceDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.ServicesDeleteByIds([]string{d.Id()})
}
